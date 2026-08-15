package discovery

import (
	"context"
	"fmt"
)

// OutdatedPackages lists Homebrew formulae and casks with newer versions available.
type OutdatedPackages struct {
	Formulae []string
	Casks    []string
}

// DiscoverOutdated lists outdated formulae and casks via `brew outdated -q`.
// Auto-updating casks are only included when Homebrew itself reports them
// outdated (e.g. the app bundle is behind the tap). Stale Caskroom metadata
// alone is not enough — use `brew upgrade --cask --greedy` manually for that.
func DiscoverOutdated(ctx context.Context, runner Runner) (OutdatedPackages, error) {
	formulaeOut, err := runner.Run(ctx, "outdated", "--formula", "-q")
	if err != nil {
		return OutdatedPackages{}, fmt.Errorf("list outdated formulae: %w", err)
	}
	casksOut, err := runner.Run(ctx, "outdated", "--cask", "-q")
	if err != nil {
		return OutdatedPackages{}, fmt.Errorf("list outdated casks: %w", err)
	}
	return OutdatedPackages{
		Formulae: parseBrewList(formulaeOut),
		Casks:    parseBrewList(casksOut),
	}, nil
}
