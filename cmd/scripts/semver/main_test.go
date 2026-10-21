package main

import (
	"strings"
	"testing"

	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
)

// Validation itself is covered in lib; these cover argument handling.
func TestRun(t *testing.T) {
	t.Run("valid bare version", func(t *testing.T) {
		require.NoError(t, run([]string{"1.2.3"}))
	})
	t.Run("valid v-prefixed version", func(t *testing.T) {
		require.NoError(t, run([]string{"v1.2.3"}))
	})
	t.Run("invalid four-segment version", func(t *testing.T) {
		require.Error(t, run([]string{"1.0.4.5"}))
	})
	t.Run("empty version", func(t *testing.T) {
		require.Error(t, run([]string{""}))
	})
	t.Run("no arguments", func(t *testing.T) {
		require.Error(t, run(nil))
	})
	t.Run("too many arguments", func(t *testing.T) {
		require.Error(t, run([]string{"1.0.0", "2.0.0"}))
	})
}

func TestRunCheckServerVersion(t *testing.T) {
	// Strip only the optional "v"; build metadata like "+beta" is significant.
	version := strings.TrimPrefix(lib.SoftwareVersion, "v")
	t.Run("matching bare version", func(t *testing.T) {
		require.NoError(t, run([]string{"--check-server-version", version}))
	})
	t.Run("matching v-prefixed version", func(t *testing.T) {
		require.NoError(t, run([]string{"--check-server-version", "v" + version}))
	})
	t.Run("mismatched version", func(t *testing.T) {
		require.Error(t, run([]string{"--check-server-version", "99.99.99"}))
	})
	t.Run("mismatched build metadata", func(t *testing.T) {
		i := strings.IndexByte(version, '+')
		if i < 0 {
			t.Skip("server version has no build metadata to differ on")
		}
		require.Error(t, run([]string{"--check-server-version", version[:i]}))
	})
	t.Run("skips check when flag absent", func(t *testing.T) {
		require.NoError(t, run([]string{"99.99.99"}))
	})
}
