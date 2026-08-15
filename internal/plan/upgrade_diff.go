package plan

import (
	"sort"
	"strconv"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildUpgradePlan returns upgrade actions for declared packages that are
// already installed and reported outdated by Homebrew or mas.
// Order: formula upgrades, cask upgrades, then mas upgrades; each group sorted
// (names for brew, ascending ID for mas).
//
// A package declared as a formula but installed as a cask (or vice versa) is
// upgraded with the type that matches how it is installed. Names match
// case-insensitively; action names use the installed Homebrew token.
//
// current.OutdatedFormulae / OutdatedCasks must be populated (e.g. via
// DiscoverOutdated). Nil outdated lists mean no upgrades (avoids treating every
// installed package as upgradable when discovery was skipped).
//
// For MAS: when packages.mas is configured, desired IDs ∩ mas.Outdated produce
// mas_upgrade actions (Name = config display name, Value = id string).
// mas.Outdated nil means outdated was not discovered → no mas upgrades.
func BuildUpgradePlan(desired config.Packages, current discovery.BrewState, mas discovery.MasState) Plan {
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

	if desired.MasConfigured && mas.Outdated != nil {
		outdatedIDs := make(map[int64]struct{}, len(mas.Outdated))
		for _, id := range mas.Outdated {
			outdatedIDs[id] = struct{}{}
		}
		var apps []config.MasApp
		for _, app := range desired.Mas {
			if _, ok := outdatedIDs[app.ID]; ok {
				apps = append(apps, app)
			}
		}
		sort.Slice(apps, func(i, j int) bool { return apps[i].ID < apps[j].ID })
		for _, app := range apps {
			actions = append(actions, Action{
				Type:  ActionMasUpgrade,
				Name:  app.Name,
				Value: strconv.FormatInt(app.ID, 10),
			})
		}
	}

	return Plan{Actions: actions}
}
