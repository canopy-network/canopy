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

// This file centralizes all controller locking behind a single managed mutex so lock waits/holds
// are observable and a stuck holder can be identified from a goroutine dump (see watchdog below)

const (
	lockWaitWarnThreshold = 2 * time.Second  // warn if a caller waits longer than this to acquire
	lockHoldWarnThreshold = 5 * time.Second  // watchdog flags a probable stall if held longer than this
	lockWatchdogInterval  = time.Second      // how often the watchdog checks the current holder
	lockReDumpInterval    = 30 * time.Second // re-dump goroutines this often while still stuck
)

// ControllerLock is the controller's mutex; it mirrors *sync.Mutex (Lock/Unlock/TryLock) so it's a
// drop-in embed while tracking the current holder and flagging bad waits/holds
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

// NewControllerLock() creates the managed lock and starts its watchdog
func NewControllerLock(log lib.LoggerI) *ControllerLock {
	l := &ControllerLock{log: log}
	go l.watchdog()
	return l
}

// Lock() acquires the lock, records the holder, and warns on long waits
func (l *ControllerLock) Lock() {
	start := time.Now()
	l.mu.Lock()
	wait := time.Since(start)
	l.setHolder(lockCaller())
	if wait > lockWaitWarnThreshold && l.log != nil {
		l.log.Errorf("controller lock: waited %s to acquire (holder=%s)", wait, l.currentHolder())
	}
}

// TryLock() acquires the lock without blocking, recording the holder on success
func (l *ControllerLock) TryLock() bool {
	if !l.mu.TryLock() {
		return false
	}
	l.setHolder(lockCaller())
	return true
}

// Unlock() releases the lock, warning if it was held too long
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

// Stop() terminates the watchdog goroutine
func (l *ControllerLock) Stop() { l.stopped.Store(true) }

// setHolder() records the goroutine that just acquired the lock
func (l *ControllerLock) setHolder(caller string) {
	l.meta.Lock()
	l.held = true
	l.holderSince = time.Now()
	l.holderCaller = caller
	l.lastDump = time.Time{}
	l.meta.Unlock()
}

// currentHolder() returns a short description of the current holder for logging
func (l *ControllerLock) currentHolder() string {
	l.meta.Lock()
	defer l.meta.Unlock()
	if !l.held {
		return "none"
	}
	return fmt.Sprintf("%s (held %s)", l.holderCaller, time.Since(l.holderSince).Round(time.Millisecond))
}

// watchdog() dumps all goroutine stacks when the lock is held past the threshold, so the stuck holder is identifiable
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
		// dump once on breach, then only every lockReDumpInterval while still stuck
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

// lockCaller() returns the file:line that called Lock()/TryLock()
func lockCaller() string {
	// skip: lockCaller (0) -> Lock/TryLock (1) -> caller (2)
	if _, file, line, ok := runtime.Caller(2); ok {
		return fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}
	return "unknown"
}
