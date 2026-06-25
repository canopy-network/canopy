package lib

import (
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

/*
This file implements process-level Go runtime (garbage collector) tuning.

Profiling of long-lived, state-heavy nodes shows that a large share of CPU is
spent in the garbage collector. The dominant source of allocations is PebbleDB's
"manual" allocator, whose read/cache buffers land on the Go heap (the off-heap
path requires cgo + jemalloc, which is unavailable in the CGO_ENABLED=0 build).
Because per-block read cost grows with accumulated state, GC pressure grows with
the chain, shrinking effective throughput headroom over time.

The mitigation here is purely a runtime knob: collect less frequently (raise
GOGC) while bounding heap growth with a soft memory ceiling (GOMEMLIMIT) sized
to the memory actually available to the process. This changes only the node's
memory/CPU profile - block production, consensus, state, and every on-chain and
RPC behavior remain identical.
*/

// memory-limit detection paths (Linux). Declared as vars so tests can override them.
var (
	cgroupV2MemMaxPath   = "/sys/fs/cgroup/memory.max"
	cgroupV1MemLimitPath = "/sys/fs/cgroup/memory/memory.limit_in_bytes"
	procMemInfoPath      = "/proc/meminfo"
)

// noMemLimitSentinel guards against cgroup "unlimited" values that are reported
// as a very large integer instead of the literal "max".
const noMemLimitSentinel = uint64(1) << 62

// TuneRuntimeGC() applies Go garbage collector tuning to reduce GC CPU overhead.
//
// It is safe and behavior-preserving: only runtime memory management is affected.
// Explicitly set GOGC / GOMEMLIMIT environment variables always take precedence
// and are never overridden, so operators keep full control. Passing a
// non-positive value for either parameter disables that particular tuning.
func TuneRuntimeGC(gcPercent, memLimitPercent int, log LoggerI) {
	// set a soft memory ceiling first so that a raised GOGC can never push heap
	// growth past the memory available to the process
	if memLimitPercent > 0 && os.Getenv("GOMEMLIMIT") == "" {
		if total, ok := detectMemoryLimit(); ok {
			limit := int64(float64(total) * float64(memLimitPercent) / 100.0)
			if limit > 0 {
				debug.SetMemoryLimit(limit)
				logInfof(log, "Set soft memory limit (GOMEMLIMIT) to %d bytes (%d%% of %d detected bytes)",
					limit, memLimitPercent, total)
			}
		} else {
			logWarnf(log, "Could not detect a memory limit; leaving GOMEMLIMIT at its default")
		}
	}
	// raise the GC target percentage so the collector runs less often; the soft
	// memory limit set above bounds the resulting heap growth
	if gcPercent > 0 && os.Getenv("GOGC") == "" {
		previous := debug.SetGCPercent(gcPercent)
		logInfof(log, "Set GC target percentage (GOGC) to %d (was %d)", gcPercent, previous)
	}
}

// detectMemoryLimit() returns the effective memory limit in bytes for the
// current process, preferring the (container-aware) cgroup limit and falling
// back to total host memory. A cgroup limit larger than host memory is capped
// at host memory.
func detectMemoryLimit() (uint64, bool) {
	hostTotal, hostOK := hostMemoryTotal()
	if limit, ok := cgroupMemoryLimit(); ok {
		if hostOK && limit > hostTotal {
			return hostTotal, true
		}
		return limit, true
	}
	if hostOK {
		return hostTotal, true
	}
	return 0, false
}

// cgroupMemoryLimit() reads the container memory limit from cgroup v2, then v1.
func cgroupMemoryLimit() (uint64, bool) {
	if b, err := os.ReadFile(cgroupV2MemMaxPath); err == nil {
		s := strings.TrimSpace(string(b))
		if s != "" && s != "max" {
			if v, err := strconv.ParseUint(s, 10, 64); err == nil && v > 0 && v < noMemLimitSentinel {
				return v, true
			}
		}
	}
	if b, err := os.ReadFile(cgroupV1MemLimitPath); err == nil {
		s := strings.TrimSpace(string(b))
		if v, err := strconv.ParseUint(s, 10, 64); err == nil && v > 0 && v < noMemLimitSentinel {
			return v, true
		}
	}
	return 0, false
}

// hostMemoryTotal() parses MemTotal (reported in kB) from /proc/meminfo.
func hostMemoryTotal() (uint64, bool) {
	b, err := os.ReadFile(procMemInfoPath)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kb * 1024, true
	}
	return 0, false
}

// logInfof() logs at info level when a logger is provided.
func logInfof(log LoggerI, format string, args ...any) {
	if log != nil {
		log.Infof(format, args...)
	}
}

// logWarnf() logs at warn level when a logger is provided.
func logWarnf(log LoggerI, format string, args ...any) {
	if log != nil {
		log.Warnf(format, args...)
	}
}
