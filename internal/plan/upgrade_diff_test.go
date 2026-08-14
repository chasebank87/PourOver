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
		Formulae: []string{"git", "wget", "jq"},
		Casks:    []string{"raycast"},
	}

	p := BuildUpgradePlan(desired, current)
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

func TestBuildUpgradePlan_EmptyWhenNoneInstalled(t *testing.T) {
	desired := config.Packages{Formulae: []string{"fzf"}, Casks: []string{"vlc"}}
	current := discovery.BrewState{}
	p := BuildUpgradePlan(desired, current)
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %+v, want empty", p.Actions)
	}
}

func TestBuildUpgradePlan_CaskDeclaredAsFormula(t *testing.T) {
	desired := config.Packages{Formulae: []string{"git", "raycast"}, Casks: nil}
	current := discovery.BrewState{
		Formulae: []string{"git"},
		Casks:    []string{"raycast"},
	}
	p := BuildUpgradePlan(desired, current)
	if got := ActionNames(p, ActionFormulaUpgrade); len(got) != 1 || got[0] != "git" {
		t.Fatalf("formula upgrades = %v, want [git]", got)
	}
	if got := ActionNames(p, ActionCaskUpgrade); len(got) != 1 || got[0] != "raycast" {
		t.Fatalf("cask upgrades = %v, want [raycast]", got)
	}
}
