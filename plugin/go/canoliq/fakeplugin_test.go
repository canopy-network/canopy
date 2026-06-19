package canoliq

import (
	"bytes"
	"sort"

	"github.com/canopy-network/go-plugin/contract"
)

// fakeStore is a tiny in-memory KV that stands in for the FSM's state for
// unit tests. It satisfies the fakeStoreHook interface declared in plugin.go,
// so a *Plugin holding a *fakeStore answers state reads/writes from memory
// instead of the unix-socket FSM round-trip.
type fakeStore struct {
	data map[string][]byte
}

func newFakeStore() *fakeStore { return &fakeStore{data: map[string][]byte{}} }

func (s *fakeStore) get(key []byte) []byte    { return s.data[string(key)] }
func (s *fakeStore) set(key, value []byte)    { s.data[string(key)] = value }
func (s *fakeStore) del(key []byte)           { delete(s.data, string(key)) }

func (s *fakeStore) read(req *contract.PluginStateReadRequest) *contract.PluginStateReadResponse {
	results := make([]*contract.PluginReadResult, 0, len(req.Keys)+len(req.Ranges))
	for _, k := range req.Keys {
		v := s.get(k.Key)
		if v == nil {
			results = append(results, &contract.PluginReadResult{QueryId: k.QueryId})
			continue
		}
		results = append(results, &contract.PluginReadResult{
			QueryId: k.QueryId,
			Entries: []*contract.PluginStateEntry{{Key: k.Key, Value: v}},
		})
	}
	// Range reads: mirror fsm/state.go::StateRead's iterator. Collect all keys
	// with the given prefix, sort lexicographically (reverse if requested),
	// honor Limit, return as Entries. Sort matters: the stuck-redemption alert
	// (and any future range-based reader) assumes ascending lex order maps to
	// the desired sort key (e.g. mature_height in the key prefix).
	for _, r := range req.Ranges {
		var matches []string
		for k := range s.data {
			if bytes.HasPrefix([]byte(k), r.Prefix) {
				matches = append(matches, k)
			}
		}
		if r.Reverse {
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
		} else {
			sort.Strings(matches)
		}
		limit := r.Limit
		if limit == 0 {
			limit = ^uint64(0) // "0 limit" semantics in fsm = unlimited
		}
		var entries []*contract.PluginStateEntry
		for i, k := range matches {
			if uint64(i) >= limit {
				break
			}
			entries = append(entries, &contract.PluginStateEntry{
				Key:   []byte(k),
				Value: s.data[k],
			})
		}
		results = append(results, &contract.PluginReadResult{
			QueryId: r.QueryId,
			Entries: entries,
		})
	}
	return &contract.PluginStateReadResponse{Results: results}
}

func (s *fakeStore) write(req *contract.PluginStateWriteRequest) *contract.PluginStateWriteResponse {
	for _, op := range req.Sets {
		s.set(op.Key, op.Value)
	}
	for _, op := range req.Deletes {
		s.del(op.Key)
	}
	return &contract.PluginStateWriteResponse{}
}

// newTestCanoliq returns a Canoliq wired to a fake plugin/store. Tests use
// the store directly to pre-seed accounts/pools/params and to assert on the
// resulting state after handlers run.
//
// A generous default Canopy Supply is seeded so the spec-default percentage
// TVL cap (DefaultParams.TvlCapBps = 3300 = 33% of canopy_stake) doesn't
// fail-closed in every non-cap test. The cap-specific tests in
// t3_tvlcap_test.go either override the Supply value explicitly or delete
// the key to exercise the absent/fail-closed branch.
func newTestCanoliq() (*Canoliq, *fakeStore) {
	store := newFakeStore()
	cfg := Config{ChainId: 2, DataDirPath: "/tmp/canoliq-test"}
	p := &Plugin{fakeStore: store, config: cfg}
	c := &Canoliq{
		Config: cfg,
		plugin: p,
		fsmId:  1,
	}
	// Generous default — effective cap at 33% is 33e15 uCNPY (~33 trillion
	// CNPY in millionths), far above any test's deposit volume.
	bz, _ := contract.Marshal(&contract.Supply{Staked: 100_000_000_000_000_000})
	store.set(contract.KeyForSupply(), bz)
	return c, store
}

// setHeight sets the plugin's current height for tests that depend on it.
func (s *fakeStore) setHeight(p *Plugin, h uint64) { p.setHeight(h) }
