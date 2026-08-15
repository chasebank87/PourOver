package plan

import (
	"slices"
	"sort"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildBrewPlan computes install/remove actions for taps, formulae, and casks.
// Package names are matched case-insensitively (Homebrew tokens are lowercase).
//
// Presence is cross-type: a package declared as a formula but installed as a
// cask (or vice versa) counts as installed and is not removed as undeclared.
// Formula removes only consider formulae installed on request (not dependencies).
// Tap removes never include homebrew/core or homebrew/cask.
//
// Action order is stable: tap adds, tap trusts, formula installs, cask installs,
// tap removes, formula removes, cask removes; each group sorted alphabetically.
// Formula removes never include runtime deps of desired formulae
// (BrewState.ProtectedFormulae from `brew deps --union`).
func BuildBrewPlan(desired config.Packages, current discovery.BrewState) Plan {
	var actions []Action

	desiredAny := sliceSet(append(append([]string{}, desired.Formulae...), desired.Casks...))
	installedAny := sliceSet(append(append([]string{}, current.Formulae...), current.Casks...))
	currentTaps := sliceSet(current.Taps)
	trustedTaps := sliceSet(current.TrustedTaps)

	for _, tap := range sortedMissingTaps(desired.Taps, currentTaps) {
		actions = append(actions, Action{
			Type:    ActionTapAdd,
			Name:    brewToken(tap.Name),
			Trusted: tap.Trusted,
		})
	}
	var needTrust []string
	for _, tap := range desired.Taps {
		if !tap.Trusted {
			continue
		}
		token := brewToken(tap.Name)
		if !discovery.NeedsExplicitTrust(token) {
			continue
		}
		if _, tapped := currentTaps[token]; !tapped {
			continue // tap_add will trust after tapping
		}
		if _, ok := trustedTaps[token]; ok {
			continue
		}
		needTrust = append(needTrust, token)
	}
	sort.Strings(needTrust)
	for _, name := range needTrust {
		actions = append(actions, Action{Type: ActionTapTrust, Name: name})
	}
	for _, name := range sortedMissingFromSet(desired.Formulae, installedAny) {
		actions = append(actions, Action{Type: ActionFormulaInstall, Name: brewToken(name)})
	}
	for _, name := range sortedMissingFromSet(desired.Casks, installedAny) {
		actions = append(actions, Action{Type: ActionCaskInstall, Name: brewToken(name)})
	}
	for _, name := range sortedMissingFromSet(current.RemovableTaps(), sliceSet(desired.TapNames())) {
		actions = append(actions, Action{Type: ActionTapRemove, Name: brewToken(name)})
	}
	protected := packageKeySet(current.ProtectedFormulae)
	for _, name := range sortedMissingFromSet(current.RemovableFormulae(), desiredAny) {
		if _, ok := protected[discovery.PackageKey(name)]; ok {
			continue
		}
		actions = append(actions, Action{Type: ActionFormulaRemove, Name: name})
	}
	for _, name := range sortedMissingFromSet(current.Casks, desiredAny) {
		actions = append(actions, Action{Type: ActionCaskRemove, Name: name})
	}

	return Plan{Actions: actions}
}

// sortedMissing returns names in want that are not in have, sorted alphabetically.
func sortedMissing(want, have []string) []string {
	return sortedMissingFromSet(want, sliceSet(have))
}

func sortedMissingTaps(want []config.TapSpec, have map[string]struct{}) []config.TapSpec {
	var missing []config.TapSpec
	for _, tap := range want {
		if _, ok := have[brewToken(tap.Name)]; !ok {
			missing = append(missing, tap)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return brewToken(missing[i].Name) < brewToken(missing[j].Name)
	})
	return missing
}

func sortedMissingFromSet(want []string, have map[string]struct{}) []string {
	var missing []string
	for _, name := range want {
		if _, ok := have[brewToken(name)]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func sliceSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[brewToken(item)] = struct{}{}
	}
	return set
}

func packageKeySet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := discovery.PackageKey(item)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func brewToken(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// ActionTypes returns the action type sequence for tests and assertions.
func ActionTypes(p Plan) []ActionType {
	types := make([]ActionType, len(p.Actions))
	for i, a := range p.Actions {
		types[i] = a.Type
	}
	return types
}

// ActionNames returns package names for actions of the given type.
func ActionNames(p Plan, typ ActionType) []string {
	var names []string
	for _, a := range p.Actions {
		if a.Type == typ {
			names = append(names, a.Name)
		}
	}
	return names
}

// IsSorted checks names are in ascending order (test helper).
func IsSorted(names []string) bool {
	return slices.IsSorted(names)
}
