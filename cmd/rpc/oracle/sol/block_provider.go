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
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
)

const processBlocksTimeLimitS = 12

var _ oracle.BlockProvider = &SolBlockProvider{}

// SolanaRpcClient is the minimal RPC surface the provider needs. The production adapter (see cli
// wiring) fetches a raw block and decodes it into *Block; tests inject a fake.
type SolanaRpcClient interface {
	GetSlot(ctx context.Context) (uint64, error)
	GetBlock(ctx context.Context, slot uint64) (*Block, error)
	// GetFirstAvailableBlock returns the lowest slot not yet purged from the ledger. Used to
	// recover from a pruned-slot error by jumping the whole gap in one shot.
	GetFirstAvailableBlock(ctx context.Context) (uint64, error)
	Close()
}

// rpcErrCode extracts the JSON-RPC error code from err, if err (or something it wraps) is a
// *jsonrpc.RPCError. NOTE: jsonrpc.RPCError.Code is a struct field, not a method - an earlier
// version of this file declared a `coder interface{ Code() int }` and used errors.As against
// that, which can never match a real *jsonrpc.RPCError (no such method exists) and silently
// always returned false. That bug meant isSkippedSlotErr/isPrunedSlotErr never fired in
// production - only in tests, which used a hand-rolled type that actually had a Code() method.
// A pruned-slot error was consequently always misclassified as a generic transient error and
// retried forever, since a pruned slot never becomes available again (see git history for the
// live incident this was caught from).
func rpcErrCode(err error) (int, bool) {
	var rpcErr *jsonrpc.RPCError
	if errors.As(err, &rpcErr) {
		return rpcErr.Code, true
	}
	return 0, false
}

// isSkippedSlotErr reports whether err indicates a skipped slot (jsonrpc -32007 or -32009).
func isSkippedSlotErr(err error) bool {
	code, ok := rpcErrCode(err)
	return ok && (code == -32007 || code == -32009)
}

// isPrunedSlotErr reports whether err indicates the slot has been pruned from the ledger
// (jsonrpc -32001, BlockCleanedUp). Unlike a skipped slot, a pruned slot never becomes
// available again, so the caller must jump forward to the first available block instead of
// just advancing by one.
func isPrunedSlotErr(err error) bool {
	code, ok := rpcErrCode(err)
	return ok && code == -32001
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

// backoff blocks for the configured retry delay, or until ctx is canceled (returning false in
// that case). Used on genuine RPC failures so repeated errors don't hammer the endpoint at the
// bare poll interval (400ms) with no growth, e.g. during an outage or rate-limit (429) storm.
func (p *SolBlockProvider) backoff(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Duration(p.config.RetryDelay) * time.Second):
		return true
	}
}

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
			if !p.backoff(ctx) {
				return
			}
			continue
		}
		// metrics update - SetChainHeadHeight is optional for testing
		if p.metrics != nil {
			p.metrics.SetChainHeadHeight(current)
		}
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
			if isPrunedSlotErr(err) {
				first, ferr := p.client.GetFirstAvailableBlock(timeoutCtx)
				if ferr != nil {
					p.logger.Errorf("[SOL-RPC] GetFirstAvailableBlock failed: %v; retrying", ferr)
					p.backoff(ctx)
					return // retry same slot next tick, do not advance
				}
				if first <= p.nextSlot {
					// first available already caught up past our target between the two calls;
					// just step forward one to avoid spinning on the same pruned slot.
					first = p.nextSlot + 1
				}
				p.logger.Warnf("[SOL-BLOCK] slot %d pruned; jumping to first available slot %d", p.nextSlot, first)
				p.nextSlot = first
				continue
			}
			if code, ok := rpcErrCode(err); ok {
				p.logger.Errorf("[SOL-RPC] GetBlock(%d) failed with unclassified rpc code %d: %v; retrying", p.nextSlot, code, err)
			} else {
				p.logger.Errorf("[SOL-RPC] GetBlock(%d) failed: %v; retrying", p.nextSlot, err)
			}
			p.backoff(ctx)
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
			// a tx that failed on-chain could still carry an order memo (e.g. a close whose SPL
			// transfer failed on insufficient balance) - log it so that case isn't indistinguishable
			// from "no order transaction ever showed up" during debugging.
			c.logger.Debugf("[SOL-TX] skipping failed on-chain tx in slot %d: %v", slot, twm.Meta.Err)
			continue
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
		if err := tx.parseInstructions(c.validator, c.logger); err != nil {
			c.logger.Warnf("[SOL-TX] parse error in slot %d tx %s: %v", slot, sig, err)
			tx.clearOrder()
			continue
		}
		if tx.Order() == nil {
			continue // not an order transaction
		}
		tx.order.WitnessedHeight = slot
		if tx.isTransfer {
			// bare SPL Transfer (tag 3) doesn't carry the mint inline; resolve it from the
			// destination token account before decimals can be looked up (non-fatal on failure)
			if tx.needsMintLookup {
				if destPK, e := solana.PublicKeyFromBase58(tx.destination); e == nil {
					if mintPK, e2 := resolveTokenAccountMint(ctx, c.rpc, destPK); e2 == nil {
						tx.mint = mintPK.String()
					} else {
						c.logger.Warnf("[SOL-TX] failed to resolve mint for token account %s in slot %d: %v", tx.destination, slot, e2)
					}
				}
			}
			// resolve SPL decimals (display-only; non-fatal on failure)
			if tx.mint != "" {
				if mintPK, e := solana.PublicKeyFromBase58(tx.mint); e == nil {
					if d, e2 := c.mints.Decimals(ctx, c.rpc, mintPK); e2 == nil {
						tx.decimals = d
					} else {
						c.logger.Warnf("[SOL-TX] failed to resolve decimals for mint %s in slot %d: %v", tx.mint, slot, e2)
					}
				} else {
					c.logger.Warnf("[SOL-TX] failed to parse mint pubkey %q in slot %d: %v", tx.mint, slot, e)
				}
			}
		}
		txs = append(txs, tx)
	}
	return newBlock(slot, res.Blockhash.String(), res.PreviousBlockhash.String(), txs), nil
}

func (c *realClient) GetFirstAvailableBlock(ctx context.Context) (uint64, error) {
	return c.rpc.GetFirstAvailableBlock(ctx)
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
