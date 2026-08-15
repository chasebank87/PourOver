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

// CaskRename is a declared cask token that Homebrew resolves to a different
// installed token (retired name / old_tokens shim).
type CaskRename struct {
	From string // name in config
	To   string // current Homebrew token
}

// DetectCaskRenames reports declared cask names that are missing from the
// installed list but resolve via `brew info` to a different token that is
// already installed. Callers should advise updating the config instead of
// installing the old name.
func DetectCaskRenames(ctx context.Context, runner Runner, declared []string, installed []string) ([]CaskRename, error) {
	have := map[string]struct{}{}
	for _, name := range installed {
		have[brewToken(name)] = struct{}{}
	}
	var missing []string
	for _, name := range declared {
		token := brewToken(name)
		if token == "" {
			continue
		}
		if _, ok := have[token]; ok {
			continue
		}
		missing = append(missing, token)
	}
	if len(missing) == 0 {
		return nil, nil
	}

	args := append([]string{"info", "--json=v2", "--cask"}, missing...)
	out, err := runner.Run(ctx, args...)
	if err != nil {
		// Best-effort: rename detection should not block planning installs.
		return nil, nil
	}
	resolved := parseCaskInfoTokens(out)
	var renames []CaskRename
	seen := map[string]struct{}{}
	for _, from := range missing {
		to, ok := resolved[from]
		if !ok || to == "" || to == from {
			continue
		}
		if _, installed := have[to]; !installed {
			continue
		}
		if _, dup := seen[from]; dup {
			continue
		}
		seen[from] = struct{}{}
		renames = append(renames, CaskRename{From: from, To: to})
	}
	return renames, nil
}

type caskInfoJSON struct {
	Casks []struct {
		Token     string   `json:"token"`
		OldTokens []string `json:"old_tokens"`
	} `json:"casks"`
}

// parseCaskInfoTokens maps each requested name (token or old_token) to the
// canonical token returned by brew info.
func parseCaskInfoTokens(out []byte) map[string]string {
	raw := extractJSONObject(out)
	if raw == nil {
		return nil
	}
	var doc caskInfoJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	resolved := map[string]string{}
	for _, c := range doc.Casks {
		token := brewToken(c.Token)
		if token == "" {
			continue
		}
		resolved[token] = token
		for _, o := range c.OldTokens {
			o = brewToken(o)
			if o != "" {
				resolved[o] = token
			}
		}
	}
	return resolved
}

func extractJSONObject(out []byte) []byte {
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return nil
	}
	return []byte(text[start : end+1])
}

type trustJSON struct {
	Taps []string `json:"taps"`
}

func parseTrustTapsJSON(out []byte) []string {
	raw := extractJSONObject(out)
	if raw == nil {
		return nil
	}
	var doc trustJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	if len(doc.Taps) == 0 {
		return nil
	}
	return doc.Taps
}
