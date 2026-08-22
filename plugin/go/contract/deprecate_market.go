package contract

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// checkMarketNotDeprecated is the shared admission guard for
// MarketStatus_DEPRECATED, called ONLY by handlers that create NEW
// exposure: deposit, borrow, deposit_collateral.
//
// DELIBERATELY NARROWER than checkMarketNotPaused. Deprecation is
// permanent (ARCM Section 19.2, "permanently retired") with no resume
// path, unlike pause. This makes the correct admission rule the OPPOSITE
// shape from pause's, not a stricter copy of it: withdraw, repay,
// withdraw_collateral, and liquidate_position must all remain OPEN on a
// deprecated market, mirroring ARCM Section 9.2 J1's reasoning for
// Insolvent markets ("freezing them would strand recoverable debt") with
// even greater force here, since a deprecated market can never be resumed
// the way a paused one can -- trapping funds in it would be permanent, not
// temporary. Only genuinely NEW exposure (a fresh deposit, a fresh borrow,
// additional collateral backing a position in a market winding down) is
// blocked. Existing positions must always have a way out.
func checkMarketNotDeprecated(market *Market, marketID string) *PluginError {
	if market.Status == MarketStatus_DEPRECATED {
		return ErrMarketDeprecated(marketID)
	}
	return nil
}

// marketDeprecateAuthority reuses the identical placeholder authority as
// marketPauseAuthority and assetTierAuthority -- see pause_market.go's
// comment on why a single shared stand-in address is used rather than a
// separate one per action, pending the real governance multisig.
var marketDeprecateAuthority []byte

func init() {
	const marketDeprecateAuthorityHex = "7961113f844bcf86dfd79570f23a8e3a59b10751"
	addr, err := hex.DecodeString(marketDeprecateAuthorityHex)
	if err != nil {
		panic(fmt.Sprintf("deprecate_market: invalid hardcoded authority address: %v", err))
	}
	if len(addr) != 20 {
		panic(fmt.Sprintf("deprecate_market: hardcoded authority address must be 20 bytes, got %d", len(addr)))
	}
	marketDeprecateAuthority = addr
}

// CheckMessageDeprecateMarket statelessly validates a 'deprecate_market' message.
func (c *Contract) CheckMessageDeprecateMarket(msg *MessageDeprecateMarket) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageDeprecateMarket applies a validated, PERMANENT retirement.
// Idempotent (deprecating an already-deprecated market is a no-op success),
// matching pause_market.go's idempotency stance -- but unlike pause, there
// is no corresponding "undeprecate" transaction anywhere in
// SupportedTransactions, and none should ever be added: "permanently
// retired" is the entire point of this status per ARCM Section 19.2. A
// market that should not have been deprecated is a governance mistake to
// be handled by creating a fresh replacement market, not by reversing this
// one -- reversibility here would silently contradict the status's own
// name and the spec's own wording.
func (c *Contract) DeliverMessageDeprecateMarket(msg *MessageDeprecateMarket, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, marketDeprecateAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	market, found, err := GetMarket(c, msg.MarketId)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}

	market.Status = MarketStatus_DEPRECATED
	if sErr := SaveMarket(c, msg.MarketId, market); sErr != nil {
		return &PluginDeliverResponse{Error: sErr}
	}
	_ = fee
	return &PluginDeliverResponse{}
}
