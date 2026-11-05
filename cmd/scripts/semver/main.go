// Command semver validates that its argument is valid semver via
// lib.IsValidVersion (the same check the auto-updater uses), so the release
// GitHub Actions fail fast on malformed versions before a release is created.
//
// Usage: semver [--check-server-version] [--stream core|plugin] <version>
// (a leading "v" is optional; exit 0 means valid).
//
// When --check-server-version is set (used by the main release workflow, not the
// plugin releases), it additionally asserts that <version> matches
// lib.SoftwareVersion, ignoring only the optional "v" prefix (build metadata
// such as "+beta" must match).
//
// When --stream is set, it additionally asserts that <version> is strictly
// greater than the latest released version of that stream (fetched from the
// GitHub API), so releases can only move versions upward. "core" compares
// against GET /releases/latest; "plugin" compares against the version embedded
// in the rolling "latest-plugin" release name. This network check runs only
// when --stream is provided; plain validation stays offline.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/canopy-network/canopy/lib"
	"golang.org/x/mod/semver"
)

// githubAPIBaseURL is the GitHub API root. It is a var so tests can point it at
// a local httptest server.
var githubAPIBaseURL = "https://api.github.com"

// pluginReleaseTag is the fixed tag of the rolling plugin release; its name
// embeds the plugin version (e.g. "plugin-go-v1.2.3").
const pluginReleaseTag = "latest-plugin"

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "semver: %v\n", err)
		os.Exit(1)
	}
}

// run validates the single version argument, optionally cross-checking it
// against lib.SoftwareVersion and/or the latest released version of a stream.
func run(args []string) error {
	fs := flag.NewFlagSet("semver", flag.ContinueOnError)
	checkServerVersion := fs.Bool("check-server-version", false,
		"also require the version to match lib.SoftwareVersion")
	stream := fs.String("stream", "",
		"also require the version to be strictly greater than the latest released version of this stream (core|plugin)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || fs.Arg(0) == "" {
		return fmt.Errorf("usage: semver [--check-server-version] [--stream core|plugin] <version>")
	}
	version := fs.Arg(0)
	if !lib.IsValidVersion(version) {
		return fmt.Errorf("invalid semantic version: %q (expected MAJOR.MINOR.PATCH, e.g. 1.2.3 or v1.2.3)", version)
	}
	if *checkServerVersion && !lib.MatchesSoftwareVersion(version) {
		return fmt.Errorf("version mismatch: remember to update the version number in lib/version.go")
	}
	if *stream != "" {
		if err := enforceVersionIncreases(*stream, version); err != nil {
			return err
		}
	}
	return nil
}

// enforceVersionIncreases requires version to be strictly greater than the
// latest released version of the given stream. The first release of a stream
// (no latest release found) is allowed.
func enforceVersionIncreases(stream, version string) error {
	latest, found, err := latestReleasedVersion(stream)
	if err != nil {
		return err
	}
	if !found {
		// no prior release for this stream: the first release is allowed
		return nil
	}
	if semver.Compare(normalize(version), normalize(latest)) <= 0 {
		return fmt.Errorf("version %q must be strictly greater than the latest %s release %q",
			version, stream, latest)
	}
	return nil
}

// latestReleasedVersion returns the latest released version of the stream and
// whether one exists. "core" reads GET /releases/latest (tag_name); "plugin"
// reads the rolling "latest-plugin" release and parses the version from its
// name. A 404 means no prior release (found=false).
func latestReleasedVersion(stream string) (version string, found bool, err error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		repo = "canopy-network/canopy"
	}
	var endpoint string
	switch stream {
	case "core":
		endpoint = fmt.Sprintf("%s/repos/%s/releases/latest", githubAPIBaseURL, repo)
	case "plugin":
		endpoint = fmt.Sprintf("%s/repos/%s/releases/tags/%s", githubAPIBaseURL, repo, pluginReleaseTag)
	default:
		return "", false, fmt.Errorf("unknown stream %q (expected core or plugin)", stream)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("User-Agent", "canopy-semver/1.0")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil // no prior release for this stream
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("unexpected status code %d fetching latest %s release",
			resp.StatusCode, stream)
	}
	var rel struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", false, err
	}
	if stream == "core" {
		return rel.TagName, true, nil
	}
	return pluginVersionFromName(rel.Name), true, nil
}

// pluginVersionFromName extracts the semver from a rolling plugin release name
// such as "plugin-go-v1.2.3" -> "v1.2.3". If no "-v" segment is present the
// name is returned unchanged.
func pluginVersionFromName(name string) string {
	if strings.Contains(name, "-v") {
		parts := strings.Split(name, "-v")
		return "v" + parts[len(parts)-1]
	}
	return name
}

// normalize adds the leading "v" that golang.org/x/mod/semver requires.
func normalize(version string) string {
	if version == "" || version[0] == 'v' {
		return version
	}
	return "v" + version
}
