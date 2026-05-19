package discovery

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverBrew lists installed formulae and casks via the runner.
func DiscoverBrew(ctx context.Context, runner Runner) (BrewState, error) {
	formulaeOut, err := runner.Run(ctx, "list", "--formula")
	if err != nil {
		return BrewState{}, fmt.Errorf("list formulae: %w", err)
	}
	casksOut, err := runner.Run(ctx, "list", "--cask")
	if err != nil {
		return BrewState{}, fmt.Errorf("list casks: %w", err)
	}

	return BrewState{
		Formulae: parseBrewList(formulaeOut),
		Casks:    parseBrewList(casksOut),
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
