package plan

import (
	"slices"
	"sort"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildBrewPlan computes install/remove actions for formulae and casks.
// Package names are matched case-insensitively (Homebrew tokens are lowercase).
//
// Presence is cross-type: a package declared as a formula but installed as a
// cask (or vice versa) counts as installed and is not removed as undeclared.
// Formula removes only consider formulae installed on request (not dependencies).
//
// Action order is stable: formula installs, cask installs, formula removes, cask removes;
// each group sorted alphabetically by name.
func BuildBrewPlan(desired config.Packages, current discovery.BrewState) Plan {
	var actions []Action

	desiredAny := sliceSet(append(append([]string{}, desired.Formulae...), desired.Casks...))
	installedAny := sliceSet(append(append([]string{}, current.Formulae...), current.Casks...))

	for _, name := range sortedMissingFromSet(desired.Formulae, installedAny) {
		actions = append(actions, Action{Type: ActionFormulaInstall, Name: brewToken(name)})
	}
	for _, name := range sortedMissingFromSet(desired.Casks, installedAny) {
		actions = append(actions, Action{Type: ActionCaskInstall, Name: brewToken(name)})
	}
	for _, name := range sortedMissingFromSet(current.RemovableFormulae(), desiredAny) {
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
