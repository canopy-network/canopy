package canoliq

import (
	"bytes"
	"log"

	"github.com/canopy-network/go-plugin/contract"
)

// ownedbond.go declares canoLiq's first owned bond on mainnet committee 29.
//
// Since the reward-ownership fix (rewardfix.go), R counts only the growth of
// Canopy validator/delegate records whose output is listed in
// params.stake_output_addresses, and that list shipped empty. This file adds
// one address to it at a pinned height, the same way the fix itself was
// activated, instead of through a CPLQ vote.
//
// The address has no key. It is the first 20 bytes of
// sha256("canoliq/owned-reward-bond/v1"), which anyone can recompute, and no
// party can sign for it. The canoLiq team funds a delegate record on
// committee 29 with compound=true and this address as output. Because Canopy
// requires the current output's signature to change the output
// (fsm/message.go, HandleMessageEditStake), the output can never be
// redirected. An unstake returns the bond to this address, where nobody can
// spend it. The bond and everything it compounds are therefore locked
// permanently, and the only spendable copy of that reward is the one this
// plugin credits to cCNPY holders. Reward is not counted twice, and nothing
// has to be burned by hand.
//
// The record's operator key can still unstake the bond, or change its
// committees, and so stop the reward. It cannot move the funds.

// MainnetOwnedBondHeight is the committee-29 block at which the owned-bond
// output joins params.stake_output_addresses. As with MainnetRewardFixHeight,
// it must be above the height at which every committee-29 node runs this
// release.
const MainnetOwnedBondHeight uint64 = 15_000

// OwnedBondOutput is the keyless output address of canoLiq's owned bond.
var OwnedBondOutput = mustHex20("1b48db4c75e6f935c5bfe62ea13550592df5f24c")

// ownedBondMaxRewardBps is persisted alongside the address. The on-chain
// params record predates max_reward_bps_per_block and stores 0. Reads backfill
// 0 to the default, so this is only about the stored record matching its
// effective value.
const ownedBondMaxRewardBps = 100

// applyMainnetOwnedBond runs once, in the BeginBlock of MainnetOwnedBondHeight
// on a mainnet profile. It is idempotent: it adds the address only if it is
// missing.
//
// When the flip happens, the bond is already a committee member, so the
// registry baseline already holds its live stake. Its first owned block
// therefore reads as zero growth, and the bond is not mistaken for reward.
func (c *Canoliq) applyMainnetOwnedBond(height uint64) *contract.PluginError {
	if c.Config.Profile != ProfileMainnet || height != MainnetOwnedBondHeight {
		return nil
	}
	g, err := c.LoadGlobals()
	if err != nil {
		return err
	}
	if !g.GenesisComplete {
		return nil
	}
	params, err := c.LoadParams()
	if err != nil {
		return err
	}
	for _, a := range params.StakeOutputAddresses {
		if bytes.Equal(a, OwnedBondOutput) {
			return nil
		}
	}
	params.StakeOutputAddresses = append(params.StakeOutputAddresses, OwnedBondOutput)
	if params.MaxRewardBpsPerBlock == 0 {
		params.MaxRewardBpsPerBlock = ownedBondMaxRewardBps
	}
	if err := c.SaveParams(params); err != nil {
		// A params record this release cannot validate must not halt the
		// chain; leave attribution off and say so.
		log.Printf("canoliq: ERROR owned-bond activation at height %d skipped: %v", height, err)
		return nil
	}
	log.Printf("canoliq: owned bond output %x added to stake_output_addresses at height %d", OwnedBondOutput, height)
	return nil
}
