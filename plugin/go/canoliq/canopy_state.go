package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// canopy_state.go provides plugin-side readers for Canopy core state that
// canoLiq needs but does not own — chiefly the network-wide Supply record
// (lib.Supply, key = contract.KeyForSupply()).
//
// Introduced for the v1.2 spec-alignment work
// (docs/canoliq-v1_2-implementation-plan.md, Phases A + B):
//
//   - readCanopySupply backs the deposit handler's percentage TVL cap check
//     (Whitepaper §9.4). The handler needs to distinguish 'Supply absent'
//     (fail-closed) from 'Supply present, Staked == 0' (accept, uncapped),
//     so it consumes the *Supply directly rather than the .Staked uint64.
//   - The snapshot path inlines its own KeyForSupply read in
//     refreshSnapshot's Batch 1; that's why there is no standalone
//     readCanopyTotalStake helper any more.
//
// Returns (nil, nil) when the key is absent. The deposit handler treats
// nil as fail-closed (deliver.go); other callers (none today) decide for
// themselves.

// readCanopySupply reads the Canopy network-wide Supply singleton and
// unmarshals it. Returns (nil, nil) when the key is absent.
func (c *Canoliq) readCanopySupply() (*contract.Supply, *contract.PluginError) {
	q := qid()
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

// Per-operator lib.Validator reads (for the restaking exposure derivation
// in WP §7) live inline in refreshSnapshot's Batch 2 — see snapshot.go's
// qCanopyVal branch. Keeping the unmarshal there shares a round-trip with
// the proposal / spend / staker / validator-incentive reads instead of
// spending one per operator.
