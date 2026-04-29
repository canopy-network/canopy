package canoliq

import (
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
	results := make([]*contract.PluginReadResult, 0, len(req.Keys))
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
func newTestCanoliq() (*Canoliq, *fakeStore) {
	store := newFakeStore()
	p := &Plugin{fakeStore: store}
	c := &Canoliq{
		Config: Config{ChainId: 2, DataDirPath: "/tmp/canoliq-test"},
		plugin: p,
		fsmId:  1,
	}
	return c, store
}

// setHeight sets the plugin's current height for tests that depend on it.
func (s *fakeStore) setHeight(p *Plugin, h uint64) { p.setHeight(h) }
