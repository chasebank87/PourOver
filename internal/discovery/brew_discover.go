package discovery

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverBrew lists installed taps, formulae, and casks via the runner.
//
// Formulae includes all installed formulae (for presence checks).
// FormulaeRequested is formulae that may be removed when undeclared:
// `brew list --formula --installed-on-request`, falling back to `brew leaves`
// if that fails. Never falls back to the full formula list (that would treat
// dependencies as undeclared packages to uninstall).
func DiscoverBrew(ctx context.Context, runner Runner) (BrewState, error) {
	tapsOut, err := runner.Run(ctx, "tap")
	if err != nil {
		return BrewState{}, fmt.Errorf("list taps: %w", err)
	}
	formulaeOut, err := runner.Run(ctx, "list", "--formula")
	if err != nil {
		return BrewState{}, fmt.Errorf("list formulae: %w", err)
	}
	casksOut, err := runner.Run(ctx, "list", "--cask")
	if err != nil {
		return BrewState{}, fmt.Errorf("list casks: %w", err)
	}

	var requested []string
	if requestedOut, err := runner.Run(ctx, "list", "--formula", "--installed-on-request"); err == nil {
		requested = parseBrewList(requestedOut)
	} else if leavesOut, leavesErr := runner.Run(ctx, "leaves"); leavesErr == nil {
		requested = parseBrewList(leavesOut)
	} else {
		// Safer than uninstalling every dependency: no formula removes.
		requested = []string{}
	}
	if requested == nil {
		requested = []string{}
	}

	return BrewState{
		Taps:              parseBrewList(tapsOut),
		Formulae:          parseBrewList(formulaeOut),
		FormulaeRequested: requested,
		Casks:             parseBrewList(casksOut),
	}, nil
}

// parseBrewList splits brew list stdout into package names (one per line).
func parseBrewList(out []byte) []string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	return names
}
