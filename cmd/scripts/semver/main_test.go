package main

import (
	"testing"

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
