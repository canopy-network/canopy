package contract

import (
"bytes"
"encoding/hex"
"fmt"
)

// marketParamsAuthority reuses the identical placeholder authority as
// marketPauseAuthority, marketDeprecateAuthority, and assetTierAuthority --
// see pause_market.go's comment on why a single shared stand-in address is
// used rather than a separate one per governance action, pending the real
// governance multisig.
var marketParamsAuthority []byte

func init() {
const marketParamsAuthorityHex = "7961113f844bcf86dfd79570f23a8e3a59b10751"
addr, err := hex.DecodeString(marketParamsAuthorityHex)
if err != nil {
panic(fmt.Sprintf("update_market_params: invalid hardcoded authority address: %v", err))
}
if len(addr) != 20 {
panic(fmt.Sprintf("update_market_params: hardcoded authority address must be 20 bytes, got %d", len(addr)))
}
marketParamsAuthority = addr
}

// CheckMessageUpdateMarketParams statelessly validates an
// 'update_market_params' message. Bounds-checks ReserveFactorBps using the
// IDENTICAL 200-3000 bps range create_market.go's own CheckMessageCreateMarket
// enforces (AYIS Section 13) -- not a separately maintained copy of the
// same constants, to avoid the two ever silently drifting apart.
func (c *Contract) CheckMessageUpdateMarketParams(msg *MessageUpdateMarketParams) *PluginCheckResponse {
if err := ValidateMarketID(msg.MarketId); err != nil {
return &PluginCheckResponse{Error: ErrInvalidMarketID(err)}
}
if len(msg.Authority) != 20 {
return &PluginCheckResponse{Error: ErrInvalidAddress()}
}
if msg.ReserveFactorBps < 200 || msg.ReserveFactorBps > 3000 {
return &PluginCheckResponse{Error: ErrReserveFactorOutOfBounds()}
}
return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageUpdateMarketParams applies a validated parameter update.
// Currently only ReserveFactorBps is settable -- MessageUpdateMarketParams
// carries no other field per arbor.pb.go's real struct definition, so this
// handler does not (and cannot) touch LTV, tier, or any other market
// parameter; those remain fixed at create_market time as of this version.
//
// NOT gated by checkMarketNotPaused or checkMarketNotDeprecated. A
// parameter update is neither new exposure nor an exit path -- it is a
// governance action on the market's own configuration, and there is no
// stated reason in ARCM Section 13/15 why a paused market's reserve factor
// should be frozen alongside user-facing transactions, or why a
// deprecated (permanently retired) market would ever need its parameters
// changed at all in practice. Left ungated rather than guessing at a rule
// neither section states; DEPRECATED markets are simply not expected to
// receive this transaction in normal operation.
func (c *Contract) DeliverMessageUpdateMarketParams(msg *MessageUpdateMarketParams, fee uint64) *PluginDeliverResponse {
if err := ValidateMarketID(msg.MarketId); err != nil {
return &PluginDeliverResponse{Error: ErrInvalidMarketID(err)}
}
if len(msg.Authority) != 20 {
return &PluginDeliverResponse{Error: ErrInvalidAddress()}
}
if msg.ReserveFactorBps < 200 || msg.ReserveFactorBps > 3000 {
return &PluginDeliverResponse{Error: ErrReserveFactorOutOfBounds()}
}
if !bytes.Equal(msg.Authority, marketParamsAuthority) {
return &PluginDeliverResponse{Error: ErrUnauthorized()}
}

market, found, err := GetMarket(c, msg.MarketId)
if err != nil {
return &PluginDeliverResponse{Error: err}
}
if !found {
return &PluginDeliverResponse{Error: ErrMarketNotFound(msg.MarketId)}
}

market.ReserveFactorBps = msg.ReserveFactorBps
if sErr := SaveMarket(c, msg.MarketId, market); sErr != nil {
return &PluginDeliverResponse{Error: sErr}
}
_ = fee
return &PluginDeliverResponse{}
}
