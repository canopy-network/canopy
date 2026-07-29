package contract

import "bytes"

// debug_enqueue_loss_factor.go -- TEMPORARY, DEBUG-ONLY, NOT FOR MAINNET.
//
// Exists solely to seed a real {28} LossFactorQueueEntry on devnet so
// ProcessLossFactorQueue's BeginBlock drain path (loss_factor_queue.go) can
// be exercised end-to-end, since its real caller -- the Layer 4
// protocol-owned backstop path -- does not exist yet. This file and its
// wiring in contract.go are intended to be reverted as a single commit
// once the live-fire test is complete. Do not build on top of this.
//
// Reuses treasuryCutAuthority (set_treasury_cut.go) rather than defining a
// fifth placeholder authority constant, since this is throwaway scaffolding
// with no mainnet lifetime of its own.
func (c *Contract) CheckMessageDebugEnqueueLossFactor(msg *MessageDebugEnqueueLossFactor) *PluginCheckResponse {
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.MarketId == "" {
		return &PluginCheckResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

func (c *Contract) DeliverMessageDebugEnqueueLossFactor(msg *MessageDebugEnqueueLossFactor, fee uint64) *PluginDeliverResponse {
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, treasuryCutAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}
	if _, found, gErr := GetMarket(c, msg.MarketId); gErr != nil {
		return &PluginDeliverResponse{Error: gErr}
	} else if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	if eErr := EnqueueLossFactorApplication(c, msg.MarketId, msg.BadDebt, c.plugin.CurrentHeight()); eErr != nil {
		return &PluginDeliverResponse{Error: eErr}
	}
	_ = fee
	return &PluginDeliverResponse{}
}
