package plan

import (
	"slices"
	"sort"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildBrewPlan computes install/remove actions for formulae and casks.
// Package names are matched case-sensitively (Homebrew convention).
//
// Action order is stable: formula installs, cask installs, formula removes, cask removes;
// each group sorted alphabetically by name.
func BuildBrewPlan(desired config.Packages, current discovery.BrewState) Plan {
	var actions []Action

	for _, name := range sortedMissing(desired.Formulae, current.Formulae) {
		actions = append(actions, Action{Type: ActionFormulaInstall, Name: name})
	}
	for _, name := range sortedMissing(desired.Casks, current.Casks) {
		actions = append(actions, Action{Type: ActionCaskInstall, Name: name})
	}
	for _, name := range sortedMissing(current.Formulae, desired.Formulae) {
		actions = append(actions, Action{Type: ActionFormulaRemove, Name: name})
	}
	for _, name := range sortedMissing(current.Casks, desired.Casks) {
		actions = append(actions, Action{Type: ActionCaskRemove, Name: name})
	}

	return Plan{Actions: actions}
}

// sortedMissing returns names in want that are not in have, sorted alphabetically.
func sortedMissing(want, have []string) []string {
	haveSet := sliceSet(have)
	var missing []string
	for _, name := range want {
		if _, ok := haveSet[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func sliceSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
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
