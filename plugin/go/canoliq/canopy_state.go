package canoliq

import (
	"math/rand"

	"github.com/canopy-network/go-plugin/contract"
)

// canopy_state.go provides plugin-side readers for Canopy core state that
// canoLiq needs but does not own — chiefly the network-wide Supply record
// (lib.Supply, key = contract.KeyForSupply()).
//
// These helpers are introduced for the v1.2 spec-alignment work
// (docs/canoliq-v1_2-implementation-plan.md Phase A):
//
//   - readCanopyTotalStake feeds the percentage TVL cap (Whitepaper §9.4),
//     replacing the v1.1 absolute tvl_cap_ucnpy with a "33% of total Canopy
//     stake" check.
//   - readCanopySupply gives Phase C (restaking optimizer, Whitepaper §7)
//     access to Supply.committee_staked — the pre-aggregated per-committee
//     stake totals — without needing per-committee iteration.
//
// Both wrap a single StateRead of contract.KeyForSupply(). They return
// (nil, nil) / (0, nil) when the key is absent — interpretable as "Canopy
// has not initialised supply yet" or "the FSM does not expose supply
// state to this plugin"; callers decide whether that is fail-open or
// fail-closed. The percentage TVL cap deliberately fails closed (rejects
// the deposit) per Phase B.

// readCanopySupply reads the Canopy network-wide Supply singleton and
// unmarshals it. Returns (nil, nil) when the key is absent.
func (c *Canoliq) readCanopySupply() (*contract.Supply, *contract.PluginError) {
	q := rand.Uint64()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: contract.KeyForSupply()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 {
		return nil, nil
	}
	supply := new(contract.Supply)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, supply); e != nil {
		return nil, e
	}
	return supply, nil
}

// readCanopyTotalStake returns Supply.staked (total locked tokens across
// all committees, including delegations). Returns (0, nil) when Supply is
// absent — callers in fail-closed paths must treat this as
// "unavailable" and reject rather than silently allow.
func (c *Canoliq) readCanopyTotalStake() (uint64, *contract.PluginError) {
	supply, err := c.readCanopySupply()
	if err != nil {
		return 0, err
	}
	if supply == nil {
		return 0, nil
	}
	return supply.Staked, nil
}

// Per-operator lib.Validator reads (for the restaking exposure derivation
// in WP §7) live inline in refreshSnapshot's Batch 2 — see snapshot.go's
// qCanopyVal branch. Keeping the unmarshal there shares a round-trip with
// the proposal / spend / staker / validator-incentive reads instead of
// spending one per operator.
