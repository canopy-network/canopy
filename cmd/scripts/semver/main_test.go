package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Without --stream the command validates offline and never contacts GitHub.
func TestRunValidatesOfflineWithoutStream(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })

	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	require.NoError(t, run([]string{"1.2.3"}))
	require.Error(t, run([]string{"not-a-version"}))
	require.False(t, hit, "no network call expected without --stream")
}

// Core stream: the new version must be strictly greater than /releases/latest.
func TestRunCoreStreamEnforcesIncrease(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })
	t.Setenv("GITHUB_REPOSITORY", "my-org/my-repo")
	t.Setenv("GITHUB_TOKEN", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/my-org/my-repo/releases/latest", r.URL.Path)
		_, _ = fmt.Fprint(w, `{"tag_name": "v0.1.32+beta"}`)
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	require.NoError(t, run([]string{"--stream", "core", "0.1.33"}))
	require.Error(t, run([]string{"--stream", "core", "0.1.32"})) // equal (build metadata ignored)
	require.Error(t, run([]string{"--stream", "core", "0.1.31"})) // lower
}

// Plugin stream: the latest version is read from the rolling release name.
func TestRunPluginStreamEnforcesIncrease(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })
	t.Setenv("GITHUB_REPOSITORY", "my-org/my-repo")
	t.Setenv("GITHUB_TOKEN", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/my-org/my-repo/releases/tags/latest-plugin", r.URL.Path)
		_, _ = fmt.Fprint(w, `{"tag_name": "latest-plugin", "name": "plugin-go-v1.2.3"}`)
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	require.NoError(t, run([]string{"--stream", "plugin", "1.2.4"}))
	require.Error(t, run([]string{"--stream", "plugin", "1.2.3"})) // equal
	require.Error(t, run([]string{"--stream", "plugin", "1.0.0"})) // lower
}

// A missing latest release (404) means the first release of the stream and is allowed.
func TestRunStreamAllowsFirstRelease(t *testing.T) {
	previousBaseURL := githubAPIBaseURL
	t.Cleanup(func() { githubAPIBaseURL = previousBaseURL })
	t.Setenv("GITHUB_REPOSITORY", "my-org/my-repo")
	t.Setenv("GITHUB_TOKEN", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	githubAPIBaseURL = server.URL

	require.NoError(t, run([]string{"--stream", "core", "0.0.1"}))
	require.NoError(t, run([]string{"--stream", "plugin", "0.0.1"}))
}
