package canoliq

import "sync/atomic"

// queryCounter backs qid(). StateRead QueryIds only need to be unique within a
// single read batch and are never persisted to consensus state, so a
// process-wide monotonic counter is sufficient — and deterministic by
// construction, unlike the math/rand IDs it replaces (L2). The atomic keeps it
// race-free even though plugin StateReads are already serialized on the
// consensus / EndBlock thread.
var queryCounter atomic.Uint64

// qid returns a fresh StateRead correlation id.
func qid() uint64 { return queryCounter.Add(1) }
