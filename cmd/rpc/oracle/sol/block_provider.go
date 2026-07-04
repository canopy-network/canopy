package sol

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle"
	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const processBlocksTimeLimitS = 12

var _ oracle.BlockProvider = &SolBlockProvider{}

// SolanaRpcClient is the minimal RPC surface the provider needs. The production adapter (see cli
// wiring) fetches a raw block and decodes it into *Block; tests inject a fake.
type SolanaRpcClient interface {
	GetSlot(ctx context.Context) (uint64, error)
	GetBlock(ctx context.Context, slot uint64) (*Block, error)
	Close()
}

// coder is implemented by the SDK's jsonrpc error and by the test skipErr; used to read the code.
type coder interface{ Code() int }

// isSkippedSlotErr reports whether err indicates a skipped slot (jsonrpc -32007 or -32009).
func isSkippedSlotErr(err error) bool {
	var c coder
	if errors.As(err, &c) {
		code := c.Code()
		return code == -32007 || code == -32009
	}
	return false
}

// resolveInstructions converts a message's compiled instructions into resolved instructions
// using the message's static account key list. Requires transactions with all accounts listed
// inline (no Address Lookup Tables — see docs/SOLANA_ORACLE_REQUIREMENTS.md).
func resolveInstructions(msg *solana.Message) ([]instruction, error) {
	out := make([]instruction, 0, len(msg.Instructions))
	for _, ci := range msg.Instructions {
		if int(ci.ProgramIDIndex) >= len(msg.AccountKeys) {
			return nil, fmt.Errorf("program id index %d out of range", ci.ProgramIDIndex)
		}
		accts := make([]solana.PublicKey, 0, len(ci.Accounts))
		for _, ai := range ci.Accounts {
			if int(ai) >= len(msg.AccountKeys) {
				return nil, fmt.Errorf("account index %d out of range", ai)
			}
			accts = append(accts, msg.AccountKeys[ai])
		}
		out = append(out, instruction{
			programID: msg.AccountKeys[ci.ProgramIDIndex],
			accounts:  accts,
			data:      []byte(ci.Data),
		})
	}
	return out, nil
}

// SolBlockProvider polls a Solana RPC endpoint at finalized commitment and delivers blocks.
type SolBlockProvider struct {
	config    lib.SolBlockProviderConfig
	client    SolanaRpcClient
	validator OrderValidator
	logger    lib.LoggerI
	metrics   *lib.Metrics
	blockChan chan types.BlockI
	nextSlot  uint64
	synced    bool
}

// newSolBlockProviderWithClient builds a provider around an injected client (used by tests + cli adapter).
func newSolBlockProviderWithClient(cfg lib.SolBlockProviderConfig, client SolanaRpcClient, v OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) *SolBlockProvider {
	return &SolBlockProvider{
		config:    cfg,
		client:    client,
		validator: v,
		logger:    logger,
		metrics:   metrics,
		blockChan: make(chan types.BlockI), // unbuffered: backpressure like eth
	}
}

// BlockCh returns the block delivery channel.
func (p *SolBlockProvider) BlockCh() chan types.BlockI { return p.blockChan }

// IsSynced reports whether the provider has caught up to the finalized tip.
func (p *SolBlockProvider) IsSynced() bool { return p.synced }

// Start begins the poll loop at the given slot.
func (p *SolBlockProvider) Start(ctx context.Context, height uint64) {
	p.nextSlot = height
	p.logger.Info("[SOL-CONN] starting solana block provider")
	go p.run(ctx)
}

func (p *SolBlockProvider) run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.config.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	defer p.client.Close()
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("[SOL-CONN] shutting down solana block provider")
			return
		case <-ticker.C:
		}
		current, err := p.client.GetSlot(ctx)
		if err != nil {
			p.logger.Errorf("[SOL-RPC] GetSlot failed: %v", err)
			continue
		}
		// metrics update - SetChainHeadHeight is optional for testing
		// if p.metrics != nil {
		// 	p.metrics.SetChainHeadHeight(current)
		// }
		if current < p.nextSlot {
			// under finalized commitment slots are monotonic; log and wait
			p.logger.Warnf("[SOL-SYNC] finalized slot %d below nextSlot %d", current, p.nextSlot)
			continue
		}
		p.synced = false
		p.processSlots(ctx, current)
		if p.nextSlot > current {
			p.synced = true
		}
	}
}

// processSlots delivers blocks for [nextSlot, current], honoring skip vs retry semantics.
func (p *SolBlockProvider) processSlots(ctx context.Context, current uint64) {
	timeoutCtx, cancel := context.WithTimeout(ctx, processBlocksTimeLimitS*time.Second)
	defer cancel()
	for p.nextSlot <= current {
		select {
		case <-timeoutCtx.Done():
			return
		default:
		}
		block, err := p.client.GetBlock(timeoutCtx, p.nextSlot)
		if err != nil {
			if isSkippedSlotErr(err) {
				p.logger.Infof("[SOL-BLOCK] slot %d skipped, advancing", p.nextSlot)
				p.nextSlot++ // advance past skipped slot, do not retry
				continue
			}
			p.logger.Errorf("[SOL-RPC] GetBlock(%d) failed: %v; retrying", p.nextSlot, err)
			return // retry same slot next tick, do not advance
		}
		select {
		case p.blockChan <- block:
		case <-timeoutCtx.Done():
			return // do not advance; retry this slot
		}
		p.nextSlot++
	}
}

// realClient adapts *rpc.Client to SolanaRpcClient, decoding raw blocks into *Block.
type realClient struct {
	rpc       *rpc.Client
	validator OrderValidator
	mints     *mintCache
	logger    lib.LoggerI
}

func (c *realClient) GetSlot(ctx context.Context) (uint64, error) {
	return c.rpc.GetSlot(ctx, rpc.CommitmentFinalized)
}

func (c *realClient) GetBlock(ctx context.Context, slot uint64) (*Block, error) {
	maxVer := uint64(0)
	res, err := c.rpc.GetBlockWithOpts(ctx, slot, &rpc.GetBlockOpts{
		Commitment:                     rpc.CommitmentFinalized,
		MaxSupportedTransactionVersion: &maxVer,
	})
	if err != nil {
		return nil, err // caller classifies skip vs transient
	}
	txs := make([]*Transaction, 0, len(res.Transactions))
	for i := range res.Transactions {
		twm := res.Transactions[i]
		if twm.Meta != nil && twm.Meta.Err != nil {
			continue // failed transaction, ignore
		}
		solTx, err := twm.GetTransaction()
		if err != nil || solTx == nil {
			c.logger.Warnf("[SOL-TX] failed to decode transaction in slot %d: %v", slot, err)
			continue
		}
		instrs, err := resolveInstructions(&solTx.Message)
		if err != nil {
			c.logger.Warnf("[SOL-TX] failed to resolve instructions in slot %d: %v", slot, err)
			continue
		}
		feePayer := ""
		if len(solTx.Message.AccountKeys) > 0 {
			feePayer = solTx.Message.AccountKeys[0].String()
		}
		sig := ""
		if len(solTx.Signatures) > 0 {
			sig = solTx.Signatures[0].String()
		}
		tx := newTransaction(sig, feePayer, instrs)
		if err := tx.parseInstructions(c.validator); err != nil {
			c.logger.Warnf("[SOL-TX] parse error in slot %d tx %s: %v", slot, sig, err)
			tx.clearOrder()
			continue
		}
		if tx.Order() == nil {
			continue // not an order transaction
		}
		tx.order.WitnessedHeight = slot
		// resolve SPL decimals (display-only; non-fatal on failure)
		if tx.isTransfer && tx.mint != "" {
			if mintPK, e := solana.PublicKeyFromBase58(tx.mint); e == nil {
				if d, e2 := c.mints.Decimals(ctx, c.rpc, mintPK); e2 == nil {
					tx.decimals = d
				}
			}
		}
		txs = append(txs, tx)
	}
	return newBlock(slot, res.Blockhash.String(), res.PreviousBlockhash.String(), txs), nil
}

func (c *realClient) Close() {} // *rpc.Client has no persistent connection to close

// NewSolBlockProvider constructs a production Solana block provider.
func NewSolBlockProvider(cfg lib.SolBlockProviderConfig, v OrderValidator, logger lib.LoggerI, metrics *lib.Metrics) (*SolBlockProvider, error) {
	client := &realClient{
		rpc:       rpc.New(cfg.NodeUrl),
		validator: v,
		mints:     newMintCache(1024),
		logger:    logger,
	}
	logger.Infof("[SOL-CONN] created solana block provider with rpc: %s", cfg.NodeUrl)
	return newSolBlockProviderWithClient(cfg, client, v, logger, metrics), nil
}
