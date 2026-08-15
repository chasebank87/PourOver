package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildUpgradePlan_OnlyInstalledDeclared(t *testing.T) {
	desired := config.Packages{
		Formulae: []string{"git", "fzf", "wget"},
		Casks:    []string{"raycast", "vlc"},
	}
	current := discovery.BrewState{
		Formulae:         []string{"git", "wget", "jq"},
		Casks:            []string{"raycast"},
		OutdatedFormulae: []string{"git", "wget"},
		OutdatedCasks:    []string{"raycast"},
	}

	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if got := ActionNames(p, ActionFormulaUpgrade); len(got) != 2 || got[0] != "git" || got[1] != "wget" {
		t.Fatalf("formula upgrades = %v, want [git wget]", got)
	}
	if got := ActionNames(p, ActionCaskUpgrade); len(got) != 1 || got[0] != "raycast" {
		t.Fatalf("cask upgrades = %v, want [raycast]", got)
	}
	// fzf not installed -> no upgrade; vlc not installed -> no upgrade
	if types := ActionTypes(p); len(types) != 3 {
		t.Fatalf("types = %v", types)
	}
}

func TestBuildUpgradePlan_SkipsCurrentPackages(t *testing.T) {
	desired := config.Packages{
		Formulae: []string{"git", "wget"},
		Casks:    []string{"warp", "zen"},
	}
	current := discovery.BrewState{
		Formulae:         []string{"git", "wget"},
		Casks:            []string{"warp", "zen"},
		OutdatedFormulae: []string{"git"}, // wget current
		OutdatedCasks:    []string{},      // both casks current
	}
	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if got := ActionNames(p, ActionFormulaUpgrade); len(got) != 1 || got[0] != "git" {
		t.Fatalf("formula upgrades = %v, want [git]", got)
	}
	if got := ActionNames(p, ActionCaskUpgrade); len(got) != 0 {
		t.Fatalf("cask upgrades = %v, want none", got)
	}
}

func TestBuildUpgradePlan_NilOutdatedMeansNoUpgrades(t *testing.T) {
	desired := config.Packages{Formulae: []string{"git"}}
	current := discovery.BrewState{Formulae: []string{"git"}} // Outdated* nil
	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want empty when outdated not discovered", p.Actions)
	}
}

func TestBuildUpgradePlan_EmptyWhenNoneInstalled(t *testing.T) {
	desired := config.Packages{Formulae: []string{"fzf"}, Casks: []string{"vlc"}}
	current := discovery.BrewState{
		OutdatedFormulae: []string{"fzf"},
		OutdatedCasks:    []string{"vlc"},
	}
	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want empty", p.Actions)
	}
}

func TestBuildUpgradePlan_CaskDeclaredAsFormula(t *testing.T) {
	desired := config.Packages{Formulae: []string{"git", "raycast"}, Casks: nil}
	current := discovery.BrewState{
		Formulae:         []string{"git"},
		Casks:            []string{"raycast"},
		OutdatedFormulae: []string{"git"},
		OutdatedCasks:    []string{"raycast"},
	}
	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if got := ActionNames(p, ActionFormulaUpgrade); len(got) != 1 || got[0] != "git" {
		t.Fatalf("formula upgrades = %v, want [git]", got)
	}
	if got := ActionNames(p, ActionCaskUpgrade); len(got) != 1 || got[0] != "raycast" {
		t.Fatalf("cask upgrades = %v, want [raycast]", got)
	}
}

func TestBuildUpgradePlan_CaseInsensitive(t *testing.T) {
	desired := config.Packages{Casks: []string{"Raycast", "Warp"}}
	current := discovery.BrewState{
		Casks:         []string{"raycast", "warp"},
		OutdatedCasks: []string{"raycast", "warp"},
	}
	p := BuildUpgradePlan(desired, current, discovery.MasState{})
	if got := ActionNames(p, ActionCaskUpgrade); len(got) != 2 || got[0] != "raycast" || got[1] != "warp" {
		t.Fatalf("cask upgrades = %v, want [raycast warp]", got)
	}
}

func TestBuildUpgradePlan_MasDeclaredOutdated(t *testing.T) {
	desired := config.Packages{
		MasConfigured: true,
		Mas: []config.MasApp{
			{Name: "Xcode", ID: 497799835},
		},
	}
	mas := discovery.MasState{
		Outdated: []int64{497799835, 310633997}, // undeclared ID ignored
	}
	p := BuildUpgradePlan(desired, discovery.BrewState{}, mas)
	if got := ActionTypes(p); len(got) != 1 || got[0] != ActionMasUpgrade {
		t.Fatalf("types = %v, want [mas_upgrade]", got)
	}
	a := p.Actions[0]
	if a.Name != "Xcode" || a.Value != "497799835" {
		t.Fatalf("action = %+v, want Name=Xcode Value=497799835", a)
	}
}

func TestBuildUpgradePlan_MasNilOutdatedMeansNoUpgrades(t *testing.T) {
	desired := config.Packages{
		MasConfigured: true,
		Mas:           []config.MasApp{{Name: "Xcode", ID: 497799835}},
	}
	p := BuildUpgradePlan(desired, discovery.BrewState{}, discovery.MasState{}) // Outdated nil
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want empty when mas outdated not discovered", p.Actions)
	}
}

func TestBuildUpgradePlan_MasUnmanagedIgnoresOutdated(t *testing.T) {
	desired := config.Packages{
		MasConfigured: false,
		Mas:           []config.MasApp{{Name: "Xcode", ID: 497799835}},
	}
	mas := discovery.MasState{Outdated: []int64{497799835}}
	p := BuildUpgradePlan(desired, discovery.BrewState{}, mas)
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want empty when mas unmanaged", p.Actions)
	}
}

func TestBuildUpgradePlan_MasUpgradesSortedByID(t *testing.T) {
	desired := config.Packages{
		MasConfigured: true,
		Mas: []config.MasApp{
			{Name: "Xcode", ID: 497799835},
			{Name: "Keynote", ID: 409183694},
		},
	}
	mas := discovery.MasState{Outdated: []int64{497799835, 409183694}}
	p := BuildUpgradePlan(desired, discovery.BrewState{}, mas)
	if got := ActionNames(p, ActionMasUpgrade); len(got) != 2 || got[0] != "Keynote" || got[1] != "Xcode" {
		t.Fatalf("mas upgrades = %v, want [Keynote Xcode] by ascending ID", got)
	}
	if p.Actions[0].Value != "409183694" || p.Actions[1].Value != "497799835" {
		t.Fatalf("values = %s,%s", p.Actions[0].Value, p.Actions[1].Value)
	}
}
