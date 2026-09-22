package controller

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// This file centralizes ALL controller-level locking behind a single managed mutex. The Controller
// embeds *ControllerLock (in place of *sync.Mutex), so every c.Lock()/c.Unlock()/c.TryLock() call —
// including the BFT loop, which locks through the Controller interface — funnels through here.
//
// Why: the controller mutex is a single global lock shared by block handling, consensus, the BFT
// loop, the mempool proposal path and RPC readers. If any holder blocks while holding it (a blocking
// channel send, a P2P send, or a hung root-chain RPC), the block-inbox consumer can never acquire it
// and the node silently stops processing blocks until a restart — with no error logs, because a
// goroutine blocked *waiting* on a mutex never logs, and the holder isn't tracked. Centralizing makes
// waits and (critically) long holds observable, and a watchdog dumps all goroutine stacks so the
// exact holder/deadlock is identifiable.

const (
	// how long a caller may wait to acquire the lock before we log a warning
	lockWaitWarnThreshold = 2 * time.Second
	// how long the lock may be held before the watchdog flags a probable stall/deadlock
	lockHoldWarnThreshold = 5 * time.Second
	// how often the watchdog checks the current holder
	lockWatchdogInterval = time.Second
	// re-dump goroutines this often while the lock stays stuck, to keep tracking a live deadlock
	lockReDumpInterval = 30 * time.Second
)

// ControllerLock is the single, centrally-managed mutex for the Controller. It satisfies the same
// Lock()/Unlock()/TryLock() surface as *sync.Mutex so it is a drop-in embed, while tracking the
// current holder and flagging pathological waits/holds.
type ControllerLock struct {
	mu  sync.Mutex  // the actual lock
	log lib.LoggerI // logger for wait/hold diagnostics

	meta         sync.Mutex // guards the holder metadata below
	held         bool       // whether the lock is currently held
	holderSince  time.Time  // when the current holder acquired the lock
	holderCaller string     // file:line of the current holder's acquisition site
	lastDump     time.Time  // last time the watchdog dumped goroutines for the current holder

	stopped atomic.Bool // set on Stop() to end the watchdog
}

// NewControllerLock() creates the managed controller lock and starts its watchdog goroutine.
func NewControllerLock(log lib.LoggerI) *ControllerLock {
	l := &ControllerLock{log: log}
	go l.watchdog()
	return l
}

// Lock() acquires the controller lock, recording the holder and warning on long waits.
func (l *ControllerLock) Lock() {
	start := time.Now()
	l.mu.Lock()
	wait := time.Since(start)
	l.setHolder(lockCaller())
	if wait > lockWaitWarnThreshold && l.log != nil {
		l.log.Errorf("controller lock: waited %s to acquire (holder=%s)", wait, l.currentHolder())
	}
}

// TryLock() attempts to acquire the lock without blocking, recording the holder on success.
func (l *ControllerLock) TryLock() bool {
	if !l.mu.TryLock() {
		return false
	}
	l.setHolder(lockCaller())
	return true
}

// Unlock() releases the controller lock, warning if it was held for a suspiciously long time.
func (l *ControllerLock) Unlock() {
	l.meta.Lock()
	since, caller := l.holderSince, l.holderCaller
	l.held = false
	l.holderCaller = ""
	l.meta.Unlock()
	if hold := time.Since(since); hold > lockHoldWarnThreshold && l.log != nil {
		l.log.Errorf("controller lock: released after being held %s by %s", hold, caller)
	}
	l.mu.Unlock()
}

// Stop() terminates the watchdog goroutine (used on controller shutdown / in tests).
func (l *ControllerLock) Stop() { l.stopped.Store(true) }

// setHolder() records the metadata of the goroutine that just acquired the lock.
func (l *ControllerLock) setHolder(caller string) {
	l.meta.Lock()
	l.held = true
	l.holderSince = time.Now()
	l.holderCaller = caller
	l.lastDump = time.Time{}
	l.meta.Unlock()
}

// currentHolder() returns a short description of the current holder for logging.
func (l *ControllerLock) currentHolder() string {
	l.meta.Lock()
	defer l.meta.Unlock()
	if !l.held {
		return "none"
	}
	return fmt.Sprintf("%s (held %s)", l.holderCaller, time.Since(l.holderSince).Round(time.Millisecond))
}

// watchdog() periodically checks whether the lock has been held past the threshold and, if so, dumps
// every goroutine's stack so the exact holder and any deadlock cycle can be identified from the logs.
func (l *ControllerLock) watchdog() {
	ticker := time.NewTicker(lockWatchdogInterval)
	defer ticker.Stop()
	for range ticker.C {
		if l.stopped.Load() {
			return
		}
		l.meta.Lock()
		held, since, caller, lastDump := l.held, l.holderSince, l.holderCaller, l.lastDump
		l.meta.Unlock()
		if !held {
			continue
		}
		hold := time.Since(since)
		if hold < lockHoldWarnThreshold {
			continue
		}
		// dump on first breach and then at a slower cadence while it remains stuck
		if !lastDump.IsZero() && time.Since(lastDump) < lockReDumpInterval {
			continue
		}
		l.meta.Lock()
		l.lastDump = time.Now()
		l.meta.Unlock()
		if l.log == nil {
			continue
		}
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		l.log.Errorf(
			"controller lock HELD for %s by %s — probable stall/deadlock; dumping all goroutines:\n%s",
			hold.Round(time.Millisecond), caller, string(buf[:n]),
		)
	}
}

// lockCaller() returns the file:line of the code that called Lock()/TryLock().
func lockCaller() string {
	// skip: lockCaller (0) -> Lock/TryLock (1) -> the actual caller (2)
	if _, file, line, ok := runtime.Caller(2); ok {
		return fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	return "unknown"
}
