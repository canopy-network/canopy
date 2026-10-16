package contract

import "bytes"

// oracle_admin.go implements the manual-override admin transactions for
// NASM Consolidated Spec Section 9.2's oracle safety state -- set_emergency_mode
// and set_circuit_breaker. Both reuse asset_tier.go's own assetTierAuthority
// (the single hardcoded "risk-committee" stand-in address), rather than
// declaring a second, separate authority constant for a role this codebase
// already treats as the same actor conceptually (ARCM Section 13's
// risk-committee multisig, not yet built as real governance for either use).

// CheckMessageSetEmergencyMode statelessly validates a 'set_emergency_mode'
// message. Mirrors CheckMessageSetAssetTier's own shape exactly.
func (c *Contract) CheckMessageSetEmergencyMode(msg *MessageSetEmergencyMode) *PluginCheckResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageSetEmergencyMode applies a validated Emergency Mode
// set/clear. active=true always sets Trigger=EMERGENCY_TRIGGER_OVERRIDE
// (this message is the manual-override path only, per its own proto doc
// comment) and records TriggeredBlock/TriggeredBy for observability.
// active=false clears the flag entirely (zero-value record) rather than
// merely setting Active=false with a stale Trigger/TriggeredBlock left
// behind -- a cleared flag should read as "never triggered," matching
// GetEmergencyMode's own found=false-is-normal convention for an asset
// that has genuinely never had Emergency Mode invoked.
func (c *Contract) DeliverMessageSetEmergencyMode(msg *MessageSetEmergencyMode, fee uint64) *PluginDeliverResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, assetTierAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	var flag *EmergencyModeFlag
	if msg.Active {
		flag = &EmergencyModeFlag{
			AssetId:        msg.AssetId,
			Active:         true,
			Trigger:        EmergencyModeTrigger_EMERGENCY_TRIGGER_OVERRIDE,
			TriggeredBlock: c.plugin.CurrentHeight(),
			TriggeredBy:    msg.Authority,
		}
	} else {
		flag = &EmergencyModeFlag{
			AssetId: msg.AssetId,
			Active:  false,
			Trigger: EmergencyModeTrigger_EMERGENCY_TRIGGER_NONE,
		}
	}

	if wErr := SetEmergencyMode(c, flag); wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	return &PluginDeliverResponse{}
}

// CheckMessageSetCircuitBreaker statelessly validates a
// 'set_circuit_breaker' message. Mirrors CheckMessageSetEmergencyMode
// exactly.
func (c *Contract) CheckMessageSetCircuitBreaker(msg *MessageSetCircuitBreaker) *PluginCheckResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginCheckResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginCheckResponse{Error: ErrInvalidAddress()}
	}
	return &PluginCheckResponse{AuthorizedSigners: [][]byte{msg.Authority}}
}

// DeliverMessageSetCircuitBreaker applies a validated Circuit Breaker
// set/clear. Mirrors DeliverMessageSetEmergencyMode's own active/clear
// shape exactly -- see that function's own doc comment for the clear-to-
// zero-value rationale.
func (c *Contract) DeliverMessageSetCircuitBreaker(msg *MessageSetCircuitBreaker, fee uint64) *PluginDeliverResponse {
	if err := ValidateAssetID(msg.AssetId); err != nil {
		return &PluginDeliverResponse{Error: ErrInvalidAssetID(err)}
	}
	if len(msg.Authority) != 20 {
		return &PluginDeliverResponse{Error: ErrInvalidAddress()}
	}
	if !bytes.Equal(msg.Authority, assetTierAuthority) {
		return &PluginDeliverResponse{Error: ErrUnauthorized()}
	}

	var state *CircuitBreakerState
	if msg.Active {
		state = &CircuitBreakerState{
			AssetId:        msg.AssetId,
			Active:         true,
			TriggeredBlock: c.plugin.CurrentHeight(),
			TriggeredBy:    msg.Authority,
		}
	} else {
		state = &CircuitBreakerState{
			AssetId: msg.AssetId,
			Active:  false,
		}
	}

	if wErr := SetCircuitBreakerState(c, state); wErr != nil {
		return &PluginDeliverResponse{Error: wErr}
	}
	return &PluginDeliverResponse{}
}
