package sol

import (
	"context"
	"errors"
	"time"

	"github.com/canopy-network/canopy/cmd/rpc/oracle"
	"github.com/canopy-network/canopy/cmd/rpc/oracle/types"
	"github.com/canopy-network/canopy/lib"
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
