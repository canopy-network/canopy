package fsm

import (
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/store"
	"github.com/stretchr/testify/require"
)

// TestStateReadWithProof exercises the detached-query proof path added to StateRead: when a key read
// sets prove=true, the FSM attaches a sparse-merkle-tree membership (or non-membership) proof that
// verifies against the committed state root. This is what lets the plugin serve trustless CNS lookups
// over the socket without a separate RPC call.
func TestStateReadWithProof(t *testing.T) {
	log := lib.NewDefaultLogger()
	db, err := store.NewStoreInMemory(log)
	require.NoError(t, err)

	// write a plugin-style custom key (prefix 100 is outside the core-reserved range). Keys MUST be
	// length-prefixed like all Canopy state keys, or the store's key decoder rejects them.
	key, val := lib.JoinLenPrefix([]byte{100}, []byte("alice")), []byte("hello-cns")
	require.NoError(t, db.Set(key, val))
	root, err := db.Commit()
	require.NoError(t, err)

	// open a read-only snapshot at the committed height (as the detached query path does)
	ro, err := db.NewReadOnly(db.Version())
	require.NoError(t, err)
	// initialize the SMT (prerequisite for GetProof) and confirm the snapshot root matches
	roRoot, err := ro.Root()
	require.NoError(t, err)
	require.Equal(t, root, roRoot)

	// minimal StateMachine backed by the read-only snapshot (StateRead only needs the store)
	sm := &StateMachine{store: ro}

	// --- membership proof for an existing key ---
	resp, err := sm.StateRead(&lib.PluginStateReadRequest{
		Keys: []*lib.PluginKeyRead{{QueryId: 1, Key: key, Prove: true}},
	})
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	res := resp.Results[0]
	require.True(t, res.Exists)
	require.NotEmpty(t, res.Proof)
	require.Equal(t, val, res.Entries[0].Value)

	valid, err := ro.VerifyProof(key, val, true, root, toNodes(res.Proof))
	require.NoError(t, err)
	require.True(t, valid, "membership proof should verify against the committed root")

	// --- non-membership proof for an absent key ---
	absent := lib.JoinLenPrefix([]byte{100}, []byte("ghost"))
	resp2, err := sm.StateRead(&lib.PluginStateReadRequest{
		Keys: []*lib.PluginKeyRead{{QueryId: 2, Key: absent, Prove: true}},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Results, 1)
	res2 := resp2.Results[0]
	require.False(t, res2.Exists)
	require.NotEmpty(t, res2.Proof)

	valid2, err := ro.VerifyProof(absent, nil, false, root, toNodes(res2.Proof))
	require.NoError(t, err)
	require.True(t, valid2, "non-membership proof should verify against the committed root")

	// --- without prove=true, no proof is attached (unchanged behavior) ---
	resp3, err := sm.StateRead(&lib.PluginStateReadRequest{
		Keys: []*lib.PluginKeyRead{{QueryId: 3, Key: key}},
	})
	require.NoError(t, err)
	require.Empty(t, resp3.Results[0].Proof)
}

// toNodes converts the plugin wire proof nodes back into store proof nodes for verification.
func toNodes(proof []*lib.ProofNode) []*lib.Node {
	out := make([]*lib.Node, len(proof))
	for i, n := range proof {
		out[i] = &lib.Node{Key: n.Key, Value: n.Value, Bitmask: n.Bitmask}
	}
	return out
}
