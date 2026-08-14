package plan

import (
	"sort"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildUpgradePlan returns upgrade actions for declared packages that are already installed.
// Order: formula upgrades then cask upgrades, each group sorted by name.
func BuildUpgradePlan(desired config.Packages, current discovery.BrewState) Plan {
	haveFormulae := sliceSet(current.Formulae)
	haveCasks := sliceSet(current.Casks)

	var formulae, casks []string
	for _, name := range desired.Formulae {
		if _, ok := haveFormulae[name]; ok {
			formulae = append(formulae, name)
		}
	}
	for _, name := range desired.Casks {
		if _, ok := haveCasks[name]; ok {
			casks = append(casks, name)
		}
	}
	sort.Strings(formulae)
	sort.Strings(casks)

	actions := make([]Action, 0, len(formulae)+len(casks))
	for _, name := range formulae {
		actions = append(actions, Action{Type: ActionFormulaUpgrade, Name: name})
	}
	for _, name := range casks {
		actions = append(actions, Action{Type: ActionCaskUpgrade, Name: name})
	}
	return Plan{Actions: actions}
}
