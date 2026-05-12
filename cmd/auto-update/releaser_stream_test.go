package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLatestPluginReleaseFiltersByPluginAssetStream(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() {
		githubAPIBaseURL = previousBaseURL
	})

	assetName := "go-plugin-" + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/my-org/my-plugin-repo/releases", r.URL.Path)
		_, _ = w.Write([]byte(fmt.Sprintf(`[
			{
				"tag_name": "v9.9.9",
				"assets": [{"name": "cli-linux-amd64", "browser_download_url": "https://example.com/cli"}],
				"draft": false,
				"prerelease": false
			},
			{
				"tag_name": "plugin-go-v2026.78.100",
				"assets": [{"name": "%s", "browser_download_url": "https://example.com/go-old"}],
				"draft": false,
				"prerelease": false
			},
			{
				"tag_name": "plugin-go-v2026.79.200",
				"assets": [{"name": "%s", "browser_download_url": "https://example.com/go-new"}],
				"draft": false,
				"prerelease": false
			}
		]`, assetName, assetName)))
	}))
	defer server.Close()

	githubAPIBaseURL = server.URL

	rm := NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypePlugin,
		RepoOwner: "my-org",
		RepoName:  "my-plugin-repo",
		PluginDir: "plugin/go",
		PluginConfig: &PluginReleaseConfig{
			AssetName:    "go-plugin-%s-%s.tar.gz",
			ArchSpecific: true,
		},
	}, "v0.0.0", true)

	release, err := rm.GetLatestRelease()
	require.NoError(t, err)
	require.Equal(t, "plugin-go-v2026.78.100", release.Version)
	require.Equal(t, "https://example.com/go-old", release.DownloadURL)
}

func TestGetLatestCLIReleaseIgnoresPluginOnlyReleases(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() {
		githubAPIBaseURL = previousBaseURL
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/canopy-network/canopy/releases", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{
				"tag_name": "plugin-go-v2026.82.1774298104",
				"assets": [{"name": "go-plugin-linux-amd64.tar.gz", "browser_download_url": "https://example.com/plugin"}],
				"draft": false,
				"prerelease": false
			},
			{
				"tag_name": "v0.1.18+beta",
				"assets": [{"name": "cli-linux-amd64", "browser_download_url": "https://example.com/cli"}],
				"draft": false,
				"prerelease": false
			}
		]`))
	}))
	defer server.Close()

	githubAPIBaseURL = server.URL

	rm := NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypeCLI,
		RepoOwner: "canopy-network",
		RepoName:  "canopy",
	}, "v0.1.17", true)

	release, err := rm.GetLatestRelease()
	require.NoError(t, err)
	require.Equal(t, "v0.1.18+beta", release.Version)
	require.Equal(t, "https://example.com/cli", release.DownloadURL)
}

