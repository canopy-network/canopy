package canoliq

import (
	"bytes"
	"log"

	"github.com/canopy-network/go-plugin/contract"
)

// accountaddress.go makes every account record carry its own address, from a
// pinned mainnet height.
//
// Canopy core stores Account.Address inside each account record, and indexers
// attribute a balance by that field, not by the state key. The plugin's send
// and treasury-spend paths credited a recipient that had no record yet without
// setting it, so the record was written with an empty Address: the balance is
// correct on chain, but an indexer files it under the empty address and the
// coins don't show for the real one.
//
// Setting the field changes the bytes of the account record, which is
// consensus-visible: a node replaying earlier blocks with this release would
// compute a different state for every such credit. So, as with
// MainnetRewardFixHeight, mainnet only switches at a pinned height, and every
// earlier block stays byte-identical. At that height a one-time migration
// writes the address into every record that is missing it, so accounts
// credited before the fix show up without needing a new transaction.

// MainnetAccountAddressFixHeight is the committee-29 block from which credited
// accounts carry their address. It must be above the height at which every
// committee-29 node runs this release; pick it with margin for
// pluginAutoUpdate.
const MainnetAccountAddressFixHeight uint64 = 130_500

// accountAddressFixActive reports whether credited account records carry
// their address at `height`. Other profiles have it from genesis.
func (c *Canoliq) accountAddressFixActive(height uint64) bool {
	if c.Config.Profile != ProfileMainnet {
		return true
	}
	return height >= MainnetAccountAddressFixHeight
}

// accountsPrefix is the store prefix every KeyForAccount key starts with
// (JoinLenPrefix skips the nil address segment).
func accountsPrefix() []byte { return contract.KeyForAccount(nil) }

// addressFromAccountKey returns the address a KeyForAccount key was built
// from, or nil if key is not exactly that shape.
func addressFromAccountKey(key []byte) []byte {
	prefix := accountsPrefix()
	if len(key) != len(prefix)+1+20 || !bytes.HasPrefix(key, prefix) || key[len(prefix)] != 20 {
		return nil
	}
	addr := key[len(prefix)+1:]
	if !bytes.Equal(key, contract.KeyForAccount(addr)) {
		return nil
	}
	return addr
}

// applyMainnetAccountAddressMigration runs once, in the BeginBlock of
// MainnetAccountAddressFixHeight on a mainnet profile: it sets Address on every
// account record that has none. Balances are untouched. Idempotent: a second
// run finds nothing to fix.
func (c *Canoliq) applyMainnetAccountAddressMigration(height uint64) *contract.PluginError {
	if c.Config.Profile != ProfileMainnet || height != MainnetAccountAddressFixHeight {
		return nil
	}
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Ranges: []*contract.PluginRangeRead{{QueryId: q, Prefix: accountsPrefix()}},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	var sets []*contract.PluginSetOp
	for _, r := range resp.Results {
		for _, e := range r.Entries {
			addr := addressFromAccountKey(e.Key)
			if addr == nil {
				continue
			}
			acct := new(contract.Account)
			if err := contract.Unmarshal(e.Value, acct); err != nil {
				return err
			}
			if len(acct.Address) != 0 {
				continue
			}
			acct.Address = addr
			bz, err := contract.Marshal(acct)
			if err != nil {
				return err
			}
			sets = append(sets, &contract.PluginSetOp{Key: e.Key, Value: bz})
		}
	}
	if len(sets) == 0 {
		return nil
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{Sets: sets}); err != nil {
		return err
	}
	log.Printf("canoliq: set the address on %d account records at height %d", len(sets), height)
	return nil
}
