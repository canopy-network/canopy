// Command semver validates that its argument is valid semver via
// lib.IsValidVersion (the same check the auto-updater uses), so the release
// GitHub Actions fail fast on malformed versions before a release is created.
//
// Usage: semver [--check-server-version] <version> (a leading "v" is optional;
// exit 0 means valid). When --check-server-version is set (used by the main
// release workflow, not the plugin releases), it additionally asserts that
// <version> matches lib.SoftwareVersion, ignoring only the optional "v" prefix
// (build metadata such as "+beta" must match).
package main

import (
	"flag"
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

// run validates the single version argument, optionally cross-checking it
// against lib.SoftwareVersion.
func run(args []string) error {
	fs := flag.NewFlagSet("semver", flag.ContinueOnError)
	checkServerVersion := fs.Bool("check-server-version", false,
		"also require the version to match lib.SoftwareVersion")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || fs.Arg(0) == "" {
		return fmt.Errorf("usage: semver [--check-server-version] <version>")
	}
	version := fs.Arg(0)
	if !lib.IsValidVersion(version) {
		return fmt.Errorf("invalid semantic version: %q (expected MAJOR.MINOR.PATCH, e.g. 1.2.3 or v1.2.3)", version)
	}
	if *checkServerVersion && !lib.MatchesSoftwareVersion(version) {
		return fmt.Errorf("version mismatch: remember to update the version number in lib/version.go")
	}
	return nil
}
