// Package version holds build-time identity for the pourover binary.
package version

// Version is the release version (e.g. "0.1.0"), set via -ldflags at release time.
// Dev builds default to "dev".
var Version = "dev"

// Commit is the short git commit, set via -ldflags at release time.
var Commit = "none"

// String returns a human-readable version string.
func String() string {
	if Commit != "" && Commit != "none" {
		return Version + " (" + Commit + ")"
	}
	return Version
}
