// Command semver validates that its argument is valid semver via
// lib.IsValidVersion (the same check the auto-updater uses), so the release
// GitHub Actions fail fast on malformed versions before a release is created.
// Usage: semver <version> (a leading "v" is optional; exit 0 means valid).
package main

import (
	"fmt"
	"os"

	"github.com/canopy-network/canopy/lib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "semver: %v\n", err)
		os.Exit(1)
	}
}

// run validates the single version argument.
func run(args []string) error {
	if len(args) != 1 || args[0] == "" {
		return fmt.Errorf("usage: semver <version>")
	}
	if !lib.IsValidVersion(args[0]) {
		return fmt.Errorf("invalid semantic version: %q (expected MAJOR.MINOR.PATCH, e.g. 1.2.3 or v1.2.3)", args[0])
	}
	return nil
}
