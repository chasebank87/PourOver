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

func TestBuildBrewPlan_TapAddBeforeInstalls(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{
			Taps:     []config.TapSpec{{Name: "homebrew/cask-fonts"}},
			Formulae: []string{"font-hack"},
		},
		discovery.BrewState{
			Taps:     []string{"homebrew/core", "homebrew/cask"},
			Formulae: nil,
		},
	)
	got := ActionTypes(plan)
	want := []ActionType{ActionTapAdd, ActionFormulaInstall}
	if len(got) != len(want) {
		t.Fatalf("types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("types[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if names := ActionNames(plan, ActionTapAdd); len(names) != 1 || names[0] != "homebrew/cask-fonts" {
		t.Fatalf("tap adds = %v", names)
	}
}

func TestBuildBrewPlan_TrustAlreadyTapped(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Taps: []config.TapSpec{{Name: "nikitabobko/tap"}}},
		discovery.BrewState{
			Taps:        []string{"homebrew/core", "nikitabobko/tap"},
			TrustedTaps: nil,
		},
	)
	if names := ActionNames(plan, ActionTapTrust); len(names) != 1 || names[0] != "nikitabobko/tap" {
		t.Fatalf("tap trusts = %v", names)
	}
	if names := ActionNames(plan, ActionTapAdd); len(names) != 0 {
		t.Fatalf("unexpected tap adds = %v", names)
	}
}

func TestBuildBrewPlan_SkipTrustWhenAlreadyTrusted(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Taps: []config.TapSpec{{Name: "nikitabobko/tap"}}},
		discovery.BrewState{
			Taps:        []string{"nikitabobko/tap"},
			TrustedTaps: []string{"nikitabobko/tap"},
		},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no actions, got %v", plan.Actions)
	}
}

func TestBuildBrewPlan_NeverUntapCore(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{},
		discovery.BrewState{
			Taps: []string{"homebrew/core", "homebrew/cask", "homebrew/cask-fonts"},
		},
	)
	if names := ActionNames(plan, ActionTapRemove); len(names) != 1 || names[0] != "homebrew/cask-fonts" {
		t.Fatalf("tap removes = %v, want only cask-fonts", names)
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
		Taps:     []config.TapSpec{{Name: "zzz/tap"}, {Name: "aaa/tap"}},
		Formulae: []string{"zebra", "alpha", "mid"},
		Casks:    []string{"zzz", "aaa"},
	}
	current := discovery.BrewState{
		Taps:     []string{"homebrew/core", "old/tap"},
		Formulae: []string{"remove-z", "remove-a"},
		Casks:    []string{"old-cask"},
	}

	plan := BuildBrewPlan(desired, current)

	wantTypes := []ActionType{
		ActionTapAdd, ActionTapAdd,
		ActionFormulaInstall, ActionFormulaInstall, ActionFormulaInstall,
		ActionCaskInstall, ActionCaskInstall,
		ActionTapRemove,
		ActionFormulaRemove, ActionFormulaRemove,
		ActionCaskRemove,
	}
	got := ActionTypes(plan)
	if len(got) != len(wantTypes) {
		t.Fatalf("len(actions) = %d, want %d; got %v", len(got), len(wantTypes), got)
	}
	for i := range wantTypes {
		if got[i] != wantTypes[i] {
			t.Fatalf("types[%d] = %v, want %v", i, got[i], wantTypes[i])
		}
	}

	if want := []string{"aaa/tap", "zzz/tap"}; !slicesEqual(ActionNames(plan, ActionTapAdd), want) {
		t.Errorf("tap adds = %v, want %v", ActionNames(plan, ActionTapAdd), want)
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
	if want := []string{"old/tap"}; !slicesEqual(ActionNames(plan, ActionTapRemove), want) {
		t.Errorf("tap removes = %v, want %v", ActionNames(plan, ActionTapRemove), want)
	}
	if want := []string{"remove-a", "remove-z"}; !slicesEqual(ActionNames(plan, ActionFormulaRemove), want) {
		t.Errorf("formula removes = %v, want %v", ActionNames(plan, ActionFormulaRemove), want)
	}
}

func TestBuildBrewPlan_CaseInsensitive(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Casks: []string{"Raycast", "Warp"}},
		discovery.BrewState{Casks: []string{"raycast", "warp"}},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no actions for case-only mismatch, got %v", plan.Actions)
	}
}

func TestBuildBrewPlan_CaskDeclaredAsFormula_StillInstalled(t *testing.T) {
	// brew install raycast from formulae list installs a cask; must not remove it.
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"git", "raycast", "warp"}, Casks: nil},
		discovery.BrewState{
			Formulae:          []string{"git", "gettext", "pcre2"},
			FormulaeRequested: []string{"git"},
			Casks:             []string{"raycast", "warp"},
		},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no actions, got %v", plan.Actions)
	}
}

func TestBuildBrewPlan_SkipsDependencyRemoves(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"git"}},
		discovery.BrewState{
			Formulae:          []string{"git", "gettext", "json-c", "libunistring", "pcre2"},
			FormulaeRequested: []string{"git"},
			Casks:             nil,
		},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no dependency removes, got %v", plan.Actions)
	}
}

func TestBuildBrewPlan_RemovesOnlyRequestedUndeclared(t *testing.T) {
	plan := BuildBrewPlan(
		config.Packages{Formulae: []string{"git"}},
		discovery.BrewState{
			Formulae:          []string{"git", "gettext", "wget"},
			FormulaeRequested: []string{"git", "wget"},
			Casks:             nil,
		},
	)
	if names := ActionNames(plan, ActionFormulaRemove); len(names) != 1 || names[0] != "wget" {
		t.Fatalf("formula removes = %v, want [wget]", names)
	}
}

func TestBuildBrewPlan_ColumnarListDoesNotFalseInstall(t *testing.T) {
	// After whitespace-splitting a remote multi-column `brew list --cask` grid,
	// already-installed declared casks must not be planned for install.
	desired := config.Packages{Casks: []string{"devin-desktop", "omnissa-horizon-client", "warp"}}
	current := discovery.BrewState{
		Casks: []string{
			"anaconda", "epic-games", "kitty", "sigmaos",
			"devin-desktop", "jump-desktop-connect", "pika", "vlc",
			"omnissa-horizon-client", "transmission", "warp", "zen",
		},
	}
	p := BuildBrewPlan(desired, current)
	if got := ActionNames(p, ActionCaskInstall); len(got) != 0 {
		t.Fatalf("cask installs = %v, want none (all already installed)", got)
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
