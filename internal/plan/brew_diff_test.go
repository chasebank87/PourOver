package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildBrewPlan_InstallsOnly(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"fzf", "git"}, Casks: []string{"raycast"}},
		discovery.BrewState{Formulae: []string{"git"}, Casks: nil},
	)

	if got := ActionTypes(plan); len(got) != 2 || got[0] != ActionFormulaInstall || got[1] != ActionCaskInstall {
		t.Fatalf("action types = %v", got)
	}
	if names := ActionNames(plan, ActionFormulaInstall); len(names) != 1 || names[0] != "fzf" {
		t.Fatalf("formula installs = %v", names)
	}
	if names := ActionNames(plan, ActionCaskInstall); len(names) != 1 || names[0] != "raycast" {
		t.Fatalf("cask installs = %v", names)
	}
}

func TestBuildBrewPlan_RemovesOnly(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"git"}},
		discovery.BrewState{Formulae: []string{"git", "wget", "zz-old"}, Casks: []string{"slack"}},
	)

	types := ActionTypes(plan)
	wantTypes := []ActionType{ActionFormulaRemove, ActionFormulaRemove, ActionCaskRemove}
	if len(types) != len(wantTypes) {
		t.Fatalf("action types = %v, want %v", types, wantTypes)
	}
	for i, wt := range wantTypes {
		if types[i] != wt {
			t.Fatalf("types[%d] = %v, want %v", i, types[i], wt)
		}
	}
	if names := ActionNames(plan, ActionFormulaRemove); !IsSorted(names) || len(names) != 2 {
		t.Fatalf("formula removes = %v", names)
	}
}

func TestBuildBrewPlan_NoChanges(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"git"}, Casks: []string{"raycast"}},
		discovery.BrewState{Formulae: []string{"git"}, Casks: []string{"raycast"}},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no actions, got %v", plan.Actions)
	}
}

func TestBuildBrewPlan_DeterministicOrder(t *testing.T) {
	desired := config.Packages{
		Formulae: []string{"zebra", "alpha", "mid"},
		Casks:    []string{"zzz", "aaa"},
	}
	current := discovery.BrewState{
		Formulae: []string{"remove-z", "remove-a"},
		Casks:    []string{"old-cask"},
	}

	plan := BuildBrewPlan(desired, current)

	wantTypes := []ActionType{
		ActionFormulaInstall, ActionFormulaInstall, ActionFormulaInstall,
		ActionCaskInstall, ActionCaskInstall,
		ActionFormulaRemove, ActionFormulaRemove,
		ActionCaskRemove,
	}
	got := ActionTypes(plan)
	if len(got) != len(wantTypes) {
		t.Fatalf("len(actions) = %d, want %d", len(got), len(wantTypes))
	}
	for i := range wantTypes {
		if got[i] != wantTypes[i] {
			t.Fatalf("types[%d] = %v, want %v", i, got[i], wantTypes[i])
		}
	}

	if !IsSorted(ActionNames(plan, ActionFormulaInstall)) {
		t.Errorf("formula installs not sorted: %v", ActionNames(plan, ActionFormulaInstall))
	}
	if want := []string{"alpha", "mid", "zebra"}; !slicesEqual(ActionNames(plan, ActionFormulaInstall), want) {
		t.Errorf("formula installs = %v, want %v", ActionNames(plan, ActionFormulaInstall), want)
	}
	if want := []string{"aaa", "zzz"}; !slicesEqual(ActionNames(plan, ActionCaskInstall), want) {
		t.Errorf("cask installs = %v, want %v", ActionNames(plan, ActionCaskInstall), want)
	}
	if want := []string{"remove-a", "remove-z"}; !slicesEqual(ActionNames(plan, ActionFormulaRemove), want) {
		t.Errorf("formula removes = %v, want %v", ActionNames(plan, ActionFormulaRemove), want)
	}
}

func TestBuildBrewPlan_CaseSensitive(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"Git"}},
		discovery.BrewState{Formulae: []string{"git"}},
	)
	if len(plan.Actions) != 2 {
		t.Fatalf("expected install+remove for case mismatch, got %d actions: %v", len(plan.Actions), plan.Actions)
	}
	if plan.Actions[0].Type != ActionFormulaInstall || plan.Actions[0].Name != "Git" {
		t.Errorf("first action = %+v", plan.Actions[0])
	}
	if plan.Actions[1].Type != ActionFormulaRemove || plan.Actions[1].Name != "git" {
		t.Errorf("second action = %+v", plan.Actions[1])
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
