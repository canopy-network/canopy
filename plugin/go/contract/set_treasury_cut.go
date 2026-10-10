package contract

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// treasuryCutAuthority reuses the identical placeholder authority as
// marketParamsAuthority, marketPauseAuthority, marketDeprecateAuthority, and
// assetTierAuthority -- see pause_market.go's comment on why a single
// shared stand-in address is used rather than a separate one per governance
// action, pending the real governance multisig. Devnet-only; will be
// replaced across all five call sites together once the real multisig
// exists, not just this one.
var treasuryCutAuthority []byte

func init() {
	const treasuryCutAuthorityHex = "7961113f844bcf86dfd79570f23a8e3a59b10751"
	addr, err := hex.DecodeString(treasuryCutAuthorityHex)
	if err != nil {
		panic(fmt.Sprintf("set_treasury_cut: invalid hardcoded authority address: %v", err))
	}
	if len(addr) != 20 {
		panic(fmt.Sprintf("set_treasury_cut: hardcoded authority address must be 20 bytes, got %d", len(addr)))
	}
	treasuryCutAuthority = addr
}

// CheckMessageSetTreasuryCut statelessly validates a 'set_treasury_cut'
// message. Bounds-checks TreasuryCutBps to 25-150 (0.25%-1.5%) -- see
// MessageSetTreasuryCut's own doc comment in arbor.proto for the full
// rationale behind this range, narrower than update_market_params's
// ReserveFactorBps 200-3000 range. No MarketId field to validate: this is a
// single GLOBAL parameter (GovernanceParams, {22}), unlike
// MessageUpdateMarketParams which is per-market.
func (c *Contract) CheckMessageSetTreasuryCut(msg *MessageSetTreasuryCut) *PluginCheckResponse {
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	if msg.TreasuryCutBps < 25 || msg.TreasuryCutBps > 150 {
		return &PluginCheckResponse{Error: ErrTreasuryCutOutOfBounds()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageSetTreasuryCut applies a validated treasury_cut_bps update.
// Read-modify-write against the single global GovernanceParams record
// ({22}) -- see GetGovernanceParams's own doc comment on found=false being
// the expected, normal steady state before any governance parameter has
// ever been set (unlike GetMarket, where not-found is an error). Not
// gated by checkMarketNotPaused/checkMarketNotDeprecated -- there is no
// market to gate against; this is a protocol-wide governance action,
// analogous to update_market_params's own ungated status for the same
// class of reasoning (see that file's doc comment).
func (c *Contract) DeliverMessageSetTreasuryCut(msg *MessageSetTreasuryCut, fee uint64) *PluginDeliverResponse {
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if msg.TreasuryCutBps < 25 || msg.TreasuryCutBps > 150 {
		return &PluginDeliverResponse{Error: ErrTreasuryCutOutOfBounds()}
	}
	if !bytes.Equal(msg.Authority, treasuryCutAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	params, _, err := GetGovernanceParams(c)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if params == nil {
		params = &GovernanceParams{}
	}

	params.TreasuryCutBps = msg.TreasuryCutBps
	if sErr := SaveGovernanceParams(c, params); sErr != nil {
		return &PluginDeliverResponse{Error: sErr}
	}
	_ = fee
	return &PluginDeliverResponse{}
}
