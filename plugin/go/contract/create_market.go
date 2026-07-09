package contract

import (
	"math/big"
	"math/rand"
)

// CheckMessageCreateMarket statelessly validates a 'create_market' message
// (ARCM Section 3, Section 19.2). No state reads here -- existence check
// happens at DeliverTx, matching the send handler's stateless/stateful split.
func (c *Contract) CheckMessageCreateMarket(msg *MessageCreateMarket) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.CollateralAssetId) == 0 || len(msg.DebtAssetId) == 0 {
		return &PluginCheckResponse{Error: ErrInvalidAmount()}
	}
	if msg.AssetTier > 4 {
		return &PluginCheckResponse{Error: ErrInvalidAssetTier()}
	}
	// AYIS Section 13: 200-3000 bps
	if msg.ReserveFactorBps < 200 || msg.ReserveFactorBps > 3000 {
		return &PluginCheckResponse{Error: ErrReserveFactorOutOfBounds()}
	}
	if len(msg.Creator) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	// ARCM Section 10: MinReporters governs price-quorum acceptance; a
	// market whose authorized_submitters never meets this floor can never
	// satisfy quorum for update_price, silently dead for pricing forever.
	// Hardcoded to the Section 15 default (3) pending {22}'s governance
	// param store, which does not exist yet (same precedent as
	// interest_rate.go's hardcoded rate constants).
	const minReporters = 3
	if len(msg.AuthorizedSubmitters) < minReporters {
		return &PluginCheckResponse{Error: ErrInsufficientSubmitters(msg.MarketId, len(msg.AuthorizedSubmitters), minReporters)}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Creator}}
}

// DeliverMessageCreateMarket handles a 'create_market' message. Initializes
// all five keys AYIS Section 4.5 requires at market creation: {16} Market
// record, {18} R_fund at zero, {25} B_index at RAY, {26} SupplyIndexRecord
// with s_rate=RAY and total_shares_outstanding=0, {27} loss_factor at RAY.
func (c *Contract) DeliverMessageCreateMarket(msg *MessageCreateMarket, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	// ARCM Section 10: MinReporters, re-checked here since DeliverTx does not
	// call CheckMessageCreateMarket and cannot assume it ran (a block may
	// arrive from a proposer whose mempool never ran this node's CheckTx).
	const minReporters = 3
	if len(msg.AuthorizedSubmitters) < minReporters {
		return &PluginDeliverResponse{Error: ErrInsufficientSubmitters(msg.MarketId, len(msg.AuthorizedSubmitters), minReporters)}
	}

	marketKey := KeyForMarket(msg.MarketId)

	// Existence check -- create_market must not silently overwrite an
	// existing market's state (would corrupt every accumulator this document
	// initializes below for a market that already has real, non-initial
	// values). A single query suffices; the read result being non-empty is
	// itself the check.
	existsQueryId := rand.Uint64()
	existsResp, err := c.plugin.StateRead(c, &PluginStateReadRequest{
		Keys: []*PluginKeyRead{{QueryId: existsQueryId, Key: marketKey}},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if existsResp.Error != nil {
		return &PluginDeliverResponse{Error: existsResp.Error}
	}
	if len(existsResp.Results) > 0 && len(existsResp.Results[0].Entries) > 0 &&
		len(existsResp.Results[0].Entries[0].Value) > 0 {
		return &PluginDeliverResponse{Error: ErrMarketAlreadyExists(msg.MarketId)}
	}

	// Build the Market record ({16}). status defaults to ACTIVE (proto zero
	// value). index_overflow_halted defaults false. layer4_pending_count
	// defaults 0. layer4_pending_bad_debt_total initializes as encoded zero,
	// matching every other uint128 accumulator's zero-init (not nil/absent --
	// an absent value and an explicit zero must not be conflated at read
	// sites downstream).
	zeroUint128, _ := TryEncodeUint128(big0())
	market := &Market{
		MarketId:                  msg.MarketId,
		CollateralAssetId:         msg.CollateralAssetId,
		DebtAssetId:               msg.DebtAssetId,
		AssetTier:                 msg.AssetTier,
		ReserveFactorBps:          msg.ReserveFactorBps,
		Creator:                   msg.Creator,
		Status:                    MarketStatus_ACTIVE,
		IndexOverflowHalted:       false,
		Layer4PendingCount:        0,
		Layer4PendingBadDebtTotal: zeroUint128,
		TotalBorrowed:             0,
		TotalSupplied:             0,
		LastAccrualBlock:          c.plugin.CurrentHeight(),
		AuthorizedSubmitters:      msg.AuthorizedSubmitters,
	}
	marketBytes, err := Marshal(market)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	// {18} R_fund initializes at zero (ARCM Section 12.1: non-negative at all
	// times; zero is the correct initial value, not merely a valid one).
	rFundBytes, ok := TryEncodeUint128(big0())
	if !ok {
		// unreachable: zero always encodes successfully. Guarded anyway per
		// this project's own standard that every TryEncodeUint128 call site
		// checks its ok value explicitly, never assumes success.
		return &PluginDeliverResponse{Error: ErrUint128EncodingOverflow("0")}
	}

	// {25} B_index initializes at RAY (AYIS Section 4.5).
	bIndexBytes, ok := TryEncodeUint128(RAY)
	if !ok {
		return &PluginDeliverResponse{Error: ErrUint128EncodingOverflow(RAY.String())}
	}

	// {26} SupplyIndexRecord: s_rate=RAY, total_shares_outstanding=0 (AYIS
	// Section 4.5). s_rate's own encoding uses the reverting wrapper since
	// this is a DeliverTx-context write with a transaction available to
	// revert (Principle 14) -- though RAY itself can never fail to encode,
	// matching this document's own reasoning for why loss_factor uses the
	// same wrapper (I8-bounded, never needs the freeze path).
	sRateBytes, err := EncodeUint128(RAY)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	supplyIndexBytes := EncodeSupplyIndexRecord(sRateBytes, 0)

	// {27} loss_factor initializes at RAY (AYIS Section 4.5, G2).
	lossFactorBytes, err := EncodeUint128(RAY)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: marketKey, Value: marketBytes},
			{Key: KeyForReserveFund(msg.MarketId), Value: rFundBytes},
			{Key: KeyForBorrowIndex(msg.MarketId), Value: bIndexBytes},
			{Key: KeyForSupplyIndex(msg.MarketId), Value: supplyIndexBytes},
			{Key: KeyForLossFactor(msg.MarketId), Value: lossFactorBytes},
		},
	})
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if writeResp.Error != nil {
		return &PluginDeliverResponse{Error: writeResp.Error}
	}
	return &PluginDeliverResponse{}
}

// big0 returns a fresh zero-valued big.Int. Small helper to avoid repeating
// big.NewInt(0) at every zero-init call site above.
func big0() *big.Int {
	return big.NewInt(0)
}
