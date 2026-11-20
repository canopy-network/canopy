package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// goAssetName is the platform-specific go plugin asset the updater looks for.
func goAssetName() string {
	return "go-plugin-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
}

// newGoPluginManager builds a plugin ReleaseManager for the go plugin stream.
func newGoPluginManager() *ReleaseManager {
	return NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypePlugin,
		RepoOwner: "my-org",
		RepoName:  "my-plugin-repo",
		PluginDir: "plugin/go",
		PluginConfig: &PluginReleaseConfig{
			AssetName:    "go-plugin-%s-%s.tar.gz",
			ArchSpecific: true,
		},
	}, "v0.0.0")
}

// The plugin updater reads the rolling release by its fixed "latest-plugin" tag
// and sources the version from the release name (not the tag).
func TestGetLatestPluginReleaseFetchesRollingTag(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/my-org/my-plugin-repo/releases/tags/latest-plugin", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{
			"tag_name": "latest-plugin",
			"name": "plugin-go-v2026.79.200",
			"draft": false,
			"prerelease": true,
			"assets": [
				{"name": "python-plugin.tar.gz", "browser_download_url": "https://example.com/python"},
				{"name": "%s", "browser_download_url": "https://example.com/go-new"}
			]
		}`, goAssetName())
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	release, err := newGoPluginManager().GetLatestRelease()
	require.NoError(t, err)
	// Version comes from the embedded release name, not the fixed tag.
	require.Equal(t, "plugin-go-v2026.79.200", release.Version)
	require.Equal(t, "https://example.com/go-new", release.DownloadURL)
}

// If the rolling release does not carry this platform's asset, it errors.
func TestGetLatestPluginReleaseMissingAssetErrors(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{
			"tag_name": "latest-plugin",
			"name": "plugin-python-v1.0.0",
			"assets": [
				{"name": "python-plugin.tar.gz", "browser_download_url": "https://example.com/python"}
			]
		}`)
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	_, err := newGoPluginManager().GetLatestRelease()
	require.Error(t, err)
}

// ShouldUpdate parses the embedded version (prefixed form) and updates when it
// is newer than the current version.
func TestShouldUpdateAcceptsPluginPrefixedSemver(t *testing.T) {
	rm := NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypePlugin,
		PluginDir: "plugin/go",
	}, "v2026.78.100")

	release := &Release{Version: "plugin-go-v2026.79.200"}
	err := rm.ShouldUpdate(release)
	require.NoError(t, err)
	require.True(t, release.ShouldUpdate)
	require.Equal(t, "v2026.79.200", release.Version)
}

// ShouldUpdate does not downgrade when the embedded version is older-or-equal.
func TestShouldUpdateRejectsOlderPluginVersion(t *testing.T) {
	rm := NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypePlugin,
		PluginDir: "plugin/go",
	}, "v2026.79.200")

	release := &Release{Version: "plugin-go-v2026.78.100"}
	err := rm.ShouldUpdate(release)
	require.NoError(t, err)
	require.False(t, release.ShouldUpdate)
}
