package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// LoadParams reads the canoLiq parameters from state, falling back to
// DefaultParams() if the key is unset. Genesis is responsible for writing
// the initial set so steady-state reads always observe the persisted value.
func (c *Canoliq) LoadParams() (*contract.CanoliqParams, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForParams()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	// FSM StateRead returns Entry{Value: nil} for missing keys (not zero
	// entries), so we must check the value length too. Unmarshaling an empty
	// byte slice yields a zero-valued CanoliqParams that ValidateParams would
	// (correctly) reject with "split sum 0 != 10000".
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return DefaultParams(), nil
	}
	params := new(contract.CanoliqParams)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, params); e != nil {
		return nil, e
	}
	backfillParams(params)
	if err := ValidateParams(params); err != nil {
		return nil, err
	}
	return params, nil
}

// backfillParams fills in fields that a params record persisted by an older
// build cannot carry, so that adding a required field does not invalidate
// every record already in state.
//
// The hazard this exists to prevent: LoadParams runs ValidateParams on every
// read, and processProposals reads params from BeginBlock. A field added after
// genesis decodes as zero forever, so a new "must be non-zero" invariant would
// make LoadParams fail chain-wide the moment the binary is upgraded, taking
// ApplyBlock down with it. That is a permanent halt with no on-chain remedy.
//
// Only fields for which zero is never a legitimate operator choice belong
// here. The tier *rates* are deliberately absent: zero is how governance
// disables one tier while leaving the other running, so backfilling them would
// silently re-enable a tier the DAO turned off.
//
// This mirrors the tolerance ValidateParams already grants the
// voting-period/unstaking pair for exactly the same reason.
func backfillParams(p *contract.CanoliqParams) {
	if p == nil {
		return
	}
	d := DefaultParams()
	// OTC tier terms (proto fields 36/37) landed after the first genesis.
	// Zero would mature a position in the block it opened, which is why
	// ValidateParams rejects it, so a zero here is always "absent", never
	// "chosen".
	if p.OtcTier90Blocks == 0 {
		p.OtcTier90Blocks = d.OtcTier90Blocks
	}
	if p.OtcTier120Blocks == 0 {
		p.OtcTier120Blocks = d.OtcTier120Blocks
	}
	// The minimum position size is only meaningful when at least one tier is
	// live. Gate on that so a record with both tiers disabled stays untouched
	// and still validates.
	if p.OtcMinLockUccnpy == 0 && (p.OtcTier90Bps > 0 || p.OtcTier120Bps > 0) {
		p.OtcMinLockUccnpy = d.OtcMinLockUccnpy
	}
	// canoliq_transfer_fee (proto field 38) landed after genesis on any chain
	// already running — same class as the OTC tier fields above: a params
	// record persisted before this field existed decodes it as zero forever,
	// which would otherwise mean free cCNPY transfers by omission rather than
	// governance choice. ValidateParams does not require fees to be
	// non-zero, so unlike the OTC tiers this isn't correctness-critical, but
	// backfilling keeps a zero here meaning the same thing it means for
	// every other fee field: "never explicitly set."
	if p.CanoliqTransferFee == 0 {
		p.CanoliqTransferFee = d.CanoliqTransferFee
	}
	// max_reward_bps_per_block (proto field 40) landed with the reward
	// ownership fix. A params record written before it existed decodes it as
	// zero, and zero means "clamp disabled" — the one value you never want to
	// inherit by omission, since it is the safety net on the very bug that
	// motivated the field. Backfill it to the default.
	if p.MaxRewardBpsPerBlock == 0 {
		p.MaxRewardBpsPerBlock = d.MaxRewardBpsPerBlock
	}
	// stake_output_addresses (proto field 39) is deliberately NOT backfilled.
	// Empty is both the default and a legitimate governance choice, and it is
	// the fail-safe direction: backfilling could only ever turn attribution on
	// for bonds nobody declared. There is nothing to inherit here.
}

// SaveParams writes the canoLiq parameters to state after validation.
func (c *Canoliq) SaveParams(params *contract.CanoliqParams) *contract.PluginError {
	if err := ValidateParams(params); err != nil {
		return err
	}
	bz, e := contract.Marshal(params)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForParams(), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}

// LoadGlobals reads the singleton globals record, returning an empty struct
// if it is not yet present. Callers must persist any mutations via SaveGlobals.
func (c *Canoliq) LoadGlobals() (*contract.CanoliqGlobals, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForGlobals()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	g := new(contract.CanoliqGlobals)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 &&
		len(resp.Results[0].Entries[0].Value) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, g); e != nil {
			return nil, e
		}
	}
	return g, nil
}

// SaveGlobals persists the globals record under the singleton key.
func (c *Canoliq) SaveGlobals(g *contract.CanoliqGlobals) *contract.PluginError {
	bz, e := contract.Marshal(g)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForGlobals(), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}
