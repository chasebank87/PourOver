package plan

import (
	"sort"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildUpgradePlan returns upgrade actions for declared packages that are
// already installed and reported outdated by Homebrew.
// Order: formula upgrades then cask upgrades, each group sorted by name.
//
// A package declared as a formula but installed as a cask (or vice versa) is
// upgraded with the type that matches how it is installed. Names match
// case-insensitively; action names use the installed Homebrew token.
//
// current.OutdatedFormulae / OutdatedCasks must be populated (e.g. via
// DiscoverOutdated). Nil outdated lists mean no upgrades (avoids treating every
// installed package as upgradable when discovery was skipped).
func BuildUpgradePlan(desired config.Packages, current discovery.BrewState) Plan {
	haveFormulae := map[string]string{} // lower -> canonical installed name
	for _, name := range current.Formulae {
		haveFormulae[brewToken(name)] = name
	}
	haveCasks := map[string]string{}
	for _, name := range current.Casks {
		haveCasks[brewToken(name)] = name
	}
	outdatedF := sliceSet(current.OutdatedFormulae)
	outdatedC := sliceSet(current.OutdatedCasks)
	if current.OutdatedFormulae == nil {
		outdatedF = nil
	}
	if current.OutdatedCasks == nil {
		outdatedC = nil
	}

	var formulae, casks []string
	seen := map[string]struct{}{}

	consider := func(name string) {
		key := brewToken(name)
		if _, ok := seen[key]; ok {
			return
		}
		if installed, ok := haveFormulae[key]; ok {
			if outdatedF != nil {
				if _, out := outdatedF[key]; out {
					formulae = append(formulae, installed)
					seen[key] = struct{}{}
				}
			}
			return
		}
		if installed, ok := haveCasks[key]; ok {
			if outdatedC != nil {
				if _, out := outdatedC[key]; out {
					casks = append(casks, installed)
					seen[key] = struct{}{}
				}
			}
		}
	}

	for _, name := range desired.Formulae {
		consider(name)
	}
	for _, name := range desired.Casks {
		consider(name)
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
