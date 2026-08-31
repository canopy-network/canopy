package controller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePluginCtlPath(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(wd))
	})

	path, err := resolvePluginCtlPath("go")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(path))
	require.FileExists(t, path)
	require.Equal(t, "pluginctl.sh", filepath.Base(path))
	require.Equal(t, "go", filepath.Base(filepath.Dir(path)))
}

func TestResolvePluginCtlPathMissing(t *testing.T) {
	_, err := resolvePluginCtlPath("missing-plugin")
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin launcher not found")
}
