package p2p

import (
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// lockWithTrace acquires a mutex and logs if the wait exceeds 100ms.
// The returned func must be called to unlock.
func lockWithTrace(name string, mux *sync.RWMutex, logger lib.LoggerI) func() {
	start := time.Now()
	mux.Lock()
	if wait := time.Since(start); wait > 100*time.Millisecond {
		if pc, file, line, ok := runtime.Caller(1); ok {
			logger.Warnf("%s mux lock wait: %s caller=%s:%d (%s)\n%s", name, wait, file, line, runtime.FuncForPC(pc).Name(), debug.Stack())
		} else {
			logger.Warnf("%s mux lock wait: %s\n%s", name, wait, debug.Stack())
		}
	}
	return func() { mux.Unlock() }
}

// rlockWithTrace acquires a read lock and logs if the wait exceeds 100ms.
// The returned func must be called to unlock.
func rlockWithTrace(name string, mux *sync.RWMutex, logger lib.LoggerI) func() {
	start := time.Now()
	mux.RLock()
	if wait := time.Since(start); wait > 100*time.Millisecond {
		if pc, file, line, ok := runtime.Caller(1); ok {
			logger.Warnf("%s mux rlock wait: %s caller=%s:%d (%s)\n%s", name, wait, file, line, runtime.FuncForPC(pc).Name(), debug.Stack())
		} else {
			logger.Warnf("%s mux rlock wait: %s\n%s", name, wait, debug.Stack())
		}
	}
	return func() { mux.RUnlock() }
}
