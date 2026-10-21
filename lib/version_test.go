package lib

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// valid, bare (as entered in the manual release flow)
		{"bare patch", "1.0.0", true},
		{"bare large", "10.20.30", true},
		{"bare prerelease", "1.0.0-rc.1", true},
		{"bare prerelease zero", "1.2.3-0", true},
		{"bare prerelease alnum", "1.2.3-alpha.1", true},
		// valid, v-prefixed (as passed by old callers / full tags)
		{"v patch", "v1.0.0", true},
		{"v build metadata", "v1.2.3+build.5", true},
		// semver.IsValid accepts partial versions; documented here so the
		// behavior is intentional and does not surprise future readers.
		{"v major only", "v1", true},
		{"v major minor", "v1.2", true},
		// invalid: the original bug that motivated this check
		{"four segments bare", "1.0.4.5", false},
		{"four segments v", "v1.0.4.5", false},
		// invalid prerelease/build forms rejected by x/mod/semver
		{"leading zero prerelease", "1.2.3-01", false},
		{"empty prerelease identifiers", "1.2.3-..", false},
		{"trailing dot prerelease", "1.2.3-alpha.", false},
		// invalid: not versions at all
		{"garbage", "foo", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsValidVersion(tt.input))
		})
	}
}

func TestMatchesSoftwareVersion(t *testing.T) {
	// Strip only the optional "v"; build metadata like "+beta" is significant.
	version := strings.TrimPrefix(SoftwareVersion, "v")
	require.True(t, MatchesSoftwareVersion(version))
	require.True(t, MatchesSoftwareVersion("v"+version))
	require.False(t, MatchesSoftwareVersion("99.99.99"))
	if i := strings.IndexByte(version, '+'); i >= 0 {
		require.False(t, MatchesSoftwareVersion(version[:i]))
	}
}

func TestNormalizeVersion(t *testing.T) {
	require.Equal(t, "v1.2.3", normalizeVersion("1.2.3"))
	require.Equal(t, "v1.2.3", normalizeVersion("v1.2.3"))
	require.Equal(t, "", normalizeVersion(""))
}
