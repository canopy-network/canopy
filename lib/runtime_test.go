package lib

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// withMemPaths overrides the memory-limit detection paths for the duration of a test.
func withMemPaths(t *testing.T, v2, v1, meminfo string) {
	t.Helper()
	origV2, origV1, origMem := cgroupV2MemMaxPath, cgroupV1MemLimitPath, procMemInfoPath
	cgroupV2MemMaxPath, cgroupV1MemLimitPath, procMemInfoPath = v2, v1, meminfo
	t.Cleanup(func() {
		cgroupV2MemMaxPath, cgroupV1MemLimitPath, procMemInfoPath = origV2, origV1, origMem
	})
}

const missingPath = "/nonexistent/canopy/test/path"

func TestHostMemoryTotal(t *testing.T) {
	meminfo := "MemTotal:       16384 kB\nMemFree:         1024 kB\n"
	withMemPaths(t, missingPath, missingPath, writeTempFile(t, "meminfo", meminfo))
	got, ok := hostMemoryTotal()
	require.True(t, ok)
	require.Equal(t, uint64(16384*1024), got)
}

func TestHostMemoryTotalMissing(t *testing.T) {
	withMemPaths(t, missingPath, missingPath, missingPath)
	_, ok := hostMemoryTotal()
	require.False(t, ok)
}

func TestCgroupV2MemoryLimit(t *testing.T) {
	withMemPaths(t, writeTempFile(t, "memory.max", "2147483648\n"), missingPath, missingPath)
	got, ok := cgroupMemoryLimit()
	require.True(t, ok)
	require.Equal(t, uint64(2147483648), got)
}

func TestCgroupV2MemoryLimitMax(t *testing.T) {
	// "max" means unlimited and should fall through to v1 (also absent here)
	withMemPaths(t, writeTempFile(t, "memory.max", "max\n"), missingPath, missingPath)
	_, ok := cgroupMemoryLimit()
	require.False(t, ok)
}

func TestCgroupV1MemoryLimitSentinel(t *testing.T) {
	// cgroup v1 reports a huge sentinel value when unlimited; it must be ignored
	withMemPaths(t, missingPath, writeTempFile(t, "limit", "9223372036854771712\n"), missingPath)
	_, ok := cgroupMemoryLimit()
	require.False(t, ok)
}

func TestCgroupV1MemoryLimit(t *testing.T) {
	withMemPaths(t, missingPath, writeTempFile(t, "limit", "1073741824\n"), missingPath)
	got, ok := cgroupMemoryLimit()
	require.True(t, ok)
	require.Equal(t, uint64(1073741824), got)
}

func TestDetectMemoryLimitCapsAtHost(t *testing.T) {
	// cgroup limit (8GiB) larger than host total (4GiB) is capped at host total
	meminfo := "MemTotal:       4194304 kB\n" // 4 GiB
	withMemPaths(t, writeTempFile(t, "memory.max", "8589934592\n"), missingPath, writeTempFile(t, "meminfo", meminfo))
	got, ok := detectMemoryLimit()
	require.True(t, ok)
	require.Equal(t, uint64(4194304*1024), got)
}

func TestDetectMemoryLimitFallbackToHost(t *testing.T) {
	meminfo := "MemTotal:       8388608 kB\n" // 8 GiB
	withMemPaths(t, missingPath, missingPath, writeTempFile(t, "meminfo", meminfo))
	got, ok := detectMemoryLimit()
	require.True(t, ok)
	require.Equal(t, uint64(8388608*1024), got)
}

func TestTuneRuntimeGCRespectsEnv(t *testing.T) {
	// when GOGC is set explicitly, TuneRuntimeGC must not change the GC percentage
	t.Setenv("GOGC", "123")
	t.Setenv("GOMEMLIMIT", "off")
	before := debug.SetGCPercent(150) // capture+set a known value
	defer debug.SetGCPercent(before)
	TuneRuntimeGC(400, 90, nil)
	current := debug.SetGCPercent(before) // reading returns the value left by TuneRuntimeGC
	require.Equal(t, 150, current, "GOGC env should prevent override")
}

func TestTuneRuntimeGCAppliesGCPercent(t *testing.T) {
	t.Setenv("GOGC", "")
	t.Setenv("GOMEMLIMIT", "off")
	original := debug.SetGCPercent(100)
	defer debug.SetGCPercent(original)
	TuneRuntimeGC(300, 0, nil)
	applied := debug.SetGCPercent(original)
	require.Equal(t, 300, applied)
}
