package canoliq

import (
	"bytes"
	"log"

	"github.com/canopy-network/go-plugin/contract"
)

// proposerincentive.go pays the validator-incentive slice to the operators
// that actually run committee 29, from a pinned mainnet height.
//
// The slice (params.validator_bps of the protocol fee) is meant to pay the
// operators running the committee — see distributeValidatorShare. That
// function weights the plugin's ValidatorRegistry, which is built from
// validator records on this chain. On a nested chain those are not the
// block producers: committee 29 is produced by root-chain validators bonded
// for it on chain 1, and they have no record here. So the slice went to
// chain-29 stakers instead, delegates included, and mostly to canoLiq's own
// owned bond, while the operators producing every block received nothing.
//
// Canopy passes each block's proposer to EndBlock (fsm/automatic.go), and the
// proposer is drawn from the root-chain committee by stake weight. Paying
// each block's slice to its proposer therefore pays exactly the operators
// running the chain, in proportion to their bonds over time, with no
// cross-chain read. The slice is credited straight to the proposer's CNLQ
// balance, the same way committee rewards and treasury spends reach an
// account, so there is nothing to claim.
//
// A block with no proposer address, or whose proposer was ejected by
// governance (F12 tombstone), sends the slice to the DAO treasury instead.
//
// The ledger the old rule accrued can never be withdrawn — no message pays
// it out — and it was attributed to the wrong addresses. At activation it is
// moved once into the DAO treasury and the entries are deleted.

// MainnetProposerIncentiveHeight is the committee-29 block from which the
// validator slice pays each block's proposer. As with MainnetRewardFixHeight,
// it must be above the height at which every committee-29 node runs this
// release.
const MainnetProposerIncentiveHeight uint64 = 60_000

// proposerIncentiveActive reports whether the validator slice pays the block
// proposer. Only the mainnet profile switches; other profiles keep the
// registry-weighted ledger their tests and devnets were built on.
func (c *Canoliq) proposerIncentiveActive(height uint64) bool {
	return c.Config.Profile == ProfileMainnet && height >= MainnetProposerIncentiveHeight
}

// payValidatorShareToProposer returns the set ops that credit `share` to the
// proposer's account, and the part of it that goes to the treasury instead
// (all of it when there is no eligible proposer).
func (c *Canoliq) payValidatorShareToProposer(share uint64, proposer []byte) ([]*contract.PluginSetOp, uint64, *contract.PluginError) {
	if share == 0 {
		return nil, 0, nil
	}
	if len(proposer) != 20 || c.readBytes(KeyForEjectedValidator(proposer)) != nil {
		return nil, share, nil
	}
	key := contract.KeyForAccount(proposer)
	acct, err := c.loadAccount(key)
	if err != nil {
		return nil, 0, err
	}
	acct.Address = proposer
	acct.Amount += share
	bz, e := contract.Marshal(acct)
	if e != nil {
		return nil, 0, e
	}
	return []*contract.PluginSetOp{{Key: key, Value: bz}}, 0, nil
}

// applyMainnetIncentiveMigration runs once, in the BeginBlock of
// MainnetProposerIncentiveHeight on a mainnet profile: it moves every
// validator-incentive ledger entry into the DAO treasury and deletes them.
// Physical CNLQ is untouched; the amounts only change which protocol-held
// bucket they are accounted in. Idempotent: a second run finds no entries.
func (c *Canoliq) applyMainnetIncentiveMigration(height uint64) *contract.PluginError {
	if c.Config.Profile != ProfileMainnet || height != MainnetProposerIncentiveHeight {
		return nil
	}
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Ranges: []*contract.PluginRangeRead{{QueryId: q, Prefix: ValidatorIncentivesPrefix()}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	var moved uint64
	var deletes []*contract.PluginDeleteOp
	for _, r := range resp.Results {
		for _, e := range r.Entries {
			// The prefix also covers the validator registry singleton
			// (KeyForValidatorRegistry), a proto that must survive. Only a
			// per-address incentive scalar has exactly this shape.
			if !isValidatorIncentiveKey(e.Key) {
				continue
			}
			moved += DecodeUint64(e.Value)
			deletes = append(deletes, &contract.PluginDeleteOp{Key: e.Key})
		}
	}
	if len(deletes) == 0 {
		return nil
	}
	treasuryKey := KeyForTreasuryCNPY()
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets:    []*contract.PluginSetOp{{Key: treasuryKey, Value: EncodeUint64(c.readScalar(treasuryKey) + moved)}},
		Deletes: deletes,
	}); err != nil {
		return err
	}
	log.Printf("canoliq: moved %d uCNPY of validator incentives from %d ledger entries to the treasury at height %d", moved, len(deletes), height)
	return nil
}

// isValidatorIncentiveKey reports whether key is KeyForValidatorIncentives of
// some 20-byte address, and not another key sharing the domain.
func isValidatorIncentiveKey(key []byte) bool {
	prefix := ValidatorIncentivesPrefix()
	if len(key) != len(prefix)+1+20 || !bytes.HasPrefix(key, prefix) || key[len(prefix)] != 20 {
		return false
	}
	return bytes.Equal(key, KeyForValidatorIncentives(key[len(prefix)+1:]))
}
