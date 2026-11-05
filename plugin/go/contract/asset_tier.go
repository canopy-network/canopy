package contract

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// assetTierAuthority is the single hardcoded address authorized to call
// set_asset_tier. Decoded once at package init.
//
// DISCLOSED, TEMPORARY CENTRALIZATION POINT -- matches arbor.proto's own
// comment on MessageSetAssetTier: "single hardcoded address for now,
// standing in for the risk-committee multisig ARCM Section 13 names as the
// intended actor for tier promotion/demotion." Same disclosed-placeholder
// pattern as authorized_submitters. Real governance-gated multisig/timelock
// is a future milestone, not this one.
var assetTierAuthority []byte

func init() {
	const assetTierAuthorityHex = "7961113f844bcf86dfd79570f23a8e3a59b10751"
	addr, err := hex.DecodeString(assetTierAuthorityHex)
	if err != nil {
		panic(fmt.Sprintf("asset_tier: invalid hardcoded authority address: %v", err))
	}
	if len(addr) != 20 {
		panic(fmt.Sprintf("asset_tier: hardcoded authority address must be 20 bytes, got %d", len(addr)))
	}
	assetTierAuthority = addr
}

// CheckMessageSetAssetTier statelessly validates a 'set_asset_tier'
// message. Tier RANGE is intentionally NOT checked here -- arbor.proto's
// own comment states "0-3 only; checked at DeliverTx".
func (c *Contract) CheckMessageSetAssetTier(msg *MessageSetAssetTier) *PluginCheckResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageSetAssetTier applies a validated tier assignment.
func (c *Contract) DeliverMessageSetAssetTier(msg *MessageSetAssetTier, fee uint64) *PluginDeliverResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, assetTierAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}
	if err := validateAssetTierRange(msg.Tier); err != nil {
		return &PluginDeliverResponse{Error: err.(*PluginError)}
	}

	tierBytes := EncodeAssetTierRecord(uint8(msg.Tier))

	writeResp, err := c.plugin.StateWrite(c, &PluginStateWriteRequest{
		Sets: []*PluginSetOp{
			{Key: KeyForAssetTier(msg.AssetId), Value: tierBytes},
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

// validateAssetTierRange is the pure, stateless check that a proposed tier
// value is in the valid 0-3 range (Tier 4/Blacklisted is never stored --
// see state_keys.go PrefixAssetTier's own comment). Extracted from
// DeliverMessageSetAssetTier so this specific invariant -- rejection
// happens before any narrowing cast to uint8 -- can be tested directly,
// without needing a full *Contract/*Plugin fixture this codebase does not
// yet have (see interest_rate_test.go for the only existing test file,
// which tests pure functions the same way).
func validateAssetTierRange(tier uint32) error {
	if tier > 3 {
		return ErrTierOutOfRange(tier)
	}
	return nil
}
