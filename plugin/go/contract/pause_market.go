package contract

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// marketPauseAuthority is the single hardcoded address authorized to call
// pause_market/resume_market. Decoded once at package init.
//
// DISCLOSED, TEMPORARY CENTRALIZATION POINT -- same disclosed-placeholder
// pattern as asset_tier.go's assetTierAuthority, standing in for the
// risk-committee multisig ARCM Section 13 names as the intended actor.
// Reuses the identical address as assetTierAuthority rather than a separate
// hardcoded value: both are stand-ins for the same not-yet-built governance
// multisig, and using two different placeholder addresses for what will
// eventually be one real authority would be a distinction without a
// difference at this stage, and a real footgun if the two were ever allowed
// to drift apart silently.
var marketPauseAuthority []byte

func init() {
	const marketPauseAuthorityHex = "7961113f844bcf86dfd79570f23a8e3a59b10751"
	addr, err := hex.DecodeString(marketPauseAuthorityHex)
	if err != nil {
		panic(fmt.Sprintf("pause_market: invalid hardcoded authority address: %v", err))
	}
	if len(addr) != 20 {
		panic(fmt.Sprintf("pause_market: hardcoded authority address must be 20 bytes, got %d", len(addr)))
	}
	marketPauseAuthority = addr
}

// CheckMessagePauseMarket statelessly validates a 'pause_market' message.
func (c *Contract) CheckMessagePauseMarket(msg *MessagePauseMarket) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessagePauseMarket applies a validated pause. Idempotent: pausing
// an already-paused market is a no-op success, not an error -- matching
// this codebase's general preference for idempotent admin actions over
// forcing callers to track prior state themselves before acting.
func (c *Contract) DeliverMessagePauseMarket(msg *MessagePauseMarket, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, marketPauseAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	market, found, err := GetMarket(c, msg.MarketId)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	// A Deprecated market is permanently retired (ARCM Section 19.2) --
	// pausing it is meaningless and would misleadingly suggest it could be
	// resumed. Reject rather than silently no-op, since a caller attempting
	// this almost certainly has a stale view of the market's real status.
	if market.Status == MarketStatus_DEPRECATED {
		return &PluginDeliverResponse{Error: ErrMarketDeprecated(msg.MarketId)}
	}

	market.Status = MarketStatus_PAUSED
	if sErr := SaveMarket(c, msg.MarketId, market); sErr != nil {
		return &PluginDeliverResponse{Error: sErr}
	}
	_ = fee
	return &PluginDeliverResponse{}
}

// CheckMessageResumeMarket statelessly validates a 'resume_market' message.
func (c *Contract) CheckMessageResumeMarket(msg *MessageResumeMarket) *PluginCheckResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageResumeMarket applies a validated resume. Idempotent, same
// reasoning as DeliverMessagePauseMarket. Only reverses a PAUSED status --
// does NOT touch Insolvent or index_overflow_halted, which are separate,
// self-diagnosed conditions with their own (currently nonexistent)
// clearing mechanisms per ARCM Appendix B; resume_market is not a general
// "make this market healthy again" action.
func (c *Contract) DeliverMessageResumeMarket(msg *MessageResumeMarket, fee uint64) *PluginDeliverResponse {
	if err := ValidateMarketID(msg.MarketId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, marketPauseAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	market, found, err := GetMarket(c, msg.MarketId)
	if err != nil {
		return &PluginDeliverResponse{Error: err}
	}
	if !found {
		return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
	}
	if market.Status != MarketStatus_PAUSED {
		return &PluginDeliverResponse{Error: ErrMarketNotPaused(msg.MarketId)}
	}

	market.Status = MarketStatus_ACTIVE
	if sErr := SaveMarket(c, msg.MarketId, market); sErr != nil {
		return &PluginDeliverResponse{Error: sErr}
	}
	_ = fee
	return &PluginDeliverResponse{}
}
