package engine

import (
	"github.com/chasebank87/PourOver/internal/selfupdate"
	"github.com/chasebank87/PourOver/internal/version"
)

// SelfUpdate checks GitHub Releases and replaces the binary when newer.
// Empty opts.Current defaults to version.Version.
func SelfUpdate(opts selfupdate.Options) (selfupdate.Result, error) {
	if opts.Current == "" {
		opts.Current = version.Version
	}
	return selfupdate.CheckAndApply(opts)
}
