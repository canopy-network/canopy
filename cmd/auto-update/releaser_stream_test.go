package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// The CLI updater lists the repository releases and ignores plugin releases
// (they do not carry a cli-* asset), selecting the newest CLI release.
func TestGetLatestCLIReleaseIgnoresPluginOnlyReleases(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() {
		githubAPIBaseURL = previousBaseURL
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/canopy-network/canopy/releases", r.URL.Path)
		_, _ = fmt.Fprintf(w, `[
			{
				"tag_name": "plugin-go-v2026.82.1774298104",
				"assets": [{"name": "go-plugin-%s-%s.tar.gz", "browser_download_url": "https://example.com/plugin"}],
				"draft": false,
				"prerelease": false
			},
			{
				"tag_name": "v0.1.18+beta",
				"assets": [{"name": "cli-%s-%s", "browser_download_url": "https://example.com/cli"}],
				"draft": false,
				"prerelease": false
			}
		]`, runtime.GOOS, runtime.GOARCH, runtime.GOOS, runtime.GOARCH)
	}))
	defer server.Close()

	githubAPIBaseURL = server.URL

	rm := NewReleaseManager(&ReleaseManagerConfig{
		Type:      ReleaseTypeCLI,
		RepoOwner: "canopy-network",
		RepoName:  "canopy",
	}, "v0.1.17")

	release, err := rm.GetLatestRelease()
	require.NoError(t, err)
	require.Equal(t, "v0.1.18+beta", release.Version)
	require.Equal(t, "https://example.com/cli", release.DownloadURL)
}
