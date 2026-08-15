package discovery

import (
	"context"
	"encoding/json"
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
//
// TrustedTaps comes from `brew trust --json=v1` when available (Homebrew 6+);
// older brew without trust support yields an empty trusted list.
func DiscoverBrew(ctx context.Context, runner Runner) (BrewState, error) {
	tapsOut, err := runner.Run(ctx, "tap")
	if err != nil {
		return BrewState{}, fmt.Errorf("list taps: %w", err)
	}
	// -1 forces one name per line; without it some brew versions print columns
	// and newline-only parsing would treat "foo  bar" as a single token, so
	// apply would try to reinstall packages that are already present.
	formulaeOut, err := runner.Run(ctx, "list", "--formula", "-1")
	if err != nil {
		return BrewState{}, fmt.Errorf("list formulae: %w", err)
	}
	casksOut, err := runner.Run(ctx, "list", "--cask", "-1")
	if err != nil {
		return BrewState{}, fmt.Errorf("list casks: %w", err)
	}

	var trusted []string
	if trustOut, trustErr := runner.Run(ctx, "trust", "--json=v1"); trustErr == nil {
		trusted = parseTrustTapsJSON(trustOut)
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
		TrustedTaps:       trusted,
		Formulae:          parseBrewList(formulaeOut),
		FormulaeRequested: requested,
		Casks:             parseBrewList(casksOut),
	}, nil
}

// parseBrewList splits brew list/tap stdout into package names.
// Splits on any whitespace so multi-column `brew list` output cannot glue
// two tokens into one (which would make installed packages look missing).
func parseBrewList(out []byte) []string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}
	return fields
}

type trustJSON struct {
	Taps []string `json:"taps"`
}

func parseTrustTapsJSON(out []byte) []string {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	// brew may print progress lines before JSON; take the last {...} block.
	start := strings.LastIndex(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil
	}
	var doc trustJSON
	if err := json.Unmarshal([]byte(text[start:end+1]), &doc); err != nil {
		return nil
	}
	if len(doc.Taps) == 0 {
		return nil
	}
	return doc.Taps
}
