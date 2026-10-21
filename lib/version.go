package lib

import (
	"strings"

	"golang.org/x/mod/semver"
)

// SoftwareVersion is the running Canopy software version. It follows golang's
// semver naming convention (https://pkg.go.dev/golang.org/x/mod/semver).
const SoftwareVersion = "v0.1.31+beta"

// IsValidVersion reports whether version is valid semver, accepting an optional
// leading "v". Shared so the release script and auto-updater validate alike.
func IsValidVersion(version string) bool {
	return semver.IsValid(normalizeVersion(version))
}

// MatchesSoftwareVersion reports whether version refers to the running
// SoftwareVersion, accepting an optional leading "v" on either. Build metadata
// (e.g. "+beta") is significant.
func MatchesSoftwareVersion(version string) bool {
	return strings.TrimPrefix(version, "v") == strings.TrimPrefix(SoftwareVersion, "v")
}

// normalizeVersion adds the leading "v" that golang.org/x/mod/semver requires.
func normalizeVersion(version string) string {
	if version == "" || version[0] == 'v' {
		return version
	}
	return "v" + version
}
