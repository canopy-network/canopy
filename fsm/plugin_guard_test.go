package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
)

// TestAssertPluginKeyWritable verifies the core guard that prevents plugins from writing records
// under reserved core state prefixes (which would collide with consensus state on range reads).
func TestAssertPluginKeyWritable(t *testing.T) {
	addr := make([]byte, 20)

	// allowed: accounts (1) and pools (2) are shared with plugins, and any prefix OUTSIDE the
	// core-reserved range (1-15) is plugin-owned (e.g. faucet=100, reward=101)
	allowed := map[string][]byte{
		"accounts":      lib.JoinLenPrefix(accountPrefix, addr),
		"pools":         lib.JoinLenPrefix(poolPrefix, addr),
		"plugin-faucet": lib.JoinLenPrefix([]byte{100}, addr),
		"plugin-reward": lib.JoinLenPrefix([]byte{101}, addr),
	}
	for name, key := range allowed {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("assertPluginKeyWritable panicked on allowed %s key: %v", name, r)
				}
			}()
			assertPluginKeyWritable(key)
		}()
	}

	// forbidden: writing under any reserved consensus/system prefix must panic
	forbidden := map[string][]byte{
		"validators": lib.JoinLenPrefix(validatorPrefix, addr),
		"committees": lib.JoinLenPrefix(committeePrefix, addr),
		"params":     lib.JoinLenPrefix(paramsPrefix, addr),
		"supply":     lib.JoinLenPrefix(supplyPrefix, addr),
		"dex":        lib.JoinLenPrefix(dexPrefix, addr),
	}
	for name, key := range forbidden {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("assertPluginKeyWritable did NOT panic on reserved %s key", name)
				}
			}()
			assertPluginKeyWritable(key)
		}()
	}
}
