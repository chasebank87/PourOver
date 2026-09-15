package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestExpandAutoUpdateTap_AddsWhenConfigured(t *testing.T) {
	pkgs := config.Packages{
		Formulae:   []string{"git"},
		AutoUpdate: config.AutoUpdate{Configured: true, Enable: true},
	}
	got := ExpandAutoUpdateTap(pkgs, nil, false)
	if len(got.Taps) != 1 || got.Taps[0].Name != config.AutoUpdateTap || !got.Taps[0].Trusted {
		t.Fatalf("Taps = %#v", got.Taps)
	}
}

func TestExpandAutoUpdateTap_SkipsWhenAlreadyTapped(t *testing.T) {
	pkgs := config.Packages{
		Taps:       []config.TapSpec{{Name: "homebrew/autoupdate", Trusted: true}},
		AutoUpdate: config.AutoUpdate{Configured: true, Enable: true},
	}
	got := ExpandAutoUpdateTap(pkgs, nil, false)
	if len(got.Taps) != 1 {
		t.Fatalf("Taps = %#v, want existing only", got.Taps)
	}
	got = ExpandAutoUpdateTap(config.Packages{AutoUpdate: config.AutoUpdate{Configured: true, Enable: true}}, []string{"domt4/autoupdate"}, false)
	if len(got.Taps) != 0 {
		t.Fatalf("already current: %#v", got.Taps)
	}
}

func TestExpandAutoUpdateTap_Unconfigured(t *testing.T) {
	pkgs := config.Packages{Formulae: []string{"git"}}
	got := ExpandAutoUpdateTap(pkgs, nil, false)
	if len(got.Taps) != 0 {
		t.Fatalf("unconfigured added tap: %#v", got.Taps)
	}
}

func TestExpandAutoUpdateTap_DisabledWithoutInstallSkipsTap(t *testing.T) {
	pkgs := config.Packages{AutoUpdate: config.AutoUpdate{Configured: true, Enable: false}}
	got := ExpandAutoUpdateTap(pkgs, nil, false)
	if len(got.Taps) != 0 {
		t.Fatalf("disabled unused tap: %#v", got.Taps)
	}
	got = ExpandAutoUpdateTap(pkgs, nil, true)
	if len(got.Taps) != 1 || got.Taps[0].Name != config.AutoUpdateTap {
		t.Fatalf("disabled but installed should keep tap: %#v", got.Taps)
	}
}

func TestBuildAutoUpdatePlan_StartWhenMissing(t *testing.T) {
	p := BuildAutoUpdatePlan(config.AutoUpdate{Configured: true, Enable: true, Interval: "12h", Upgrade: true}, discovery.AutoupdateState{})
	if len(p.Actions) != 1 || p.Actions[0].Type != ActionBrewAutoupdateStart {
		t.Fatalf("actions = %#v", p.Actions)
	}
	if p.Actions[0].Name != "12h" || p.Actions[0].Value != "--upgrade" {
		t.Fatalf("start action = %+v", p.Actions[0])
	}
}

func TestBuildAutoUpdatePlan_NoopWhenMatchingAndRunning(t *testing.T) {
	desired := config.AutoUpdate{Configured: true, Enable: true, Interval: "86400", Cleanup: true}
	current := discovery.AutoupdateState{
		Installed: true,
		Running:   true,
		Interval:  config.AutoUpdateInterval{Seconds: 86400},
		Cleanup:   true,
	}
	p := BuildAutoUpdatePlan(desired, current)
	if len(p.Actions) != 0 {
		t.Fatalf("actions = %#v, want none", p.Actions)
	}
}

func TestBuildAutoUpdatePlan_RestartWhenFlagsDiffer(t *testing.T) {
	desired := config.AutoUpdate{Configured: true, Enable: true, Upgrade: true, Cleanup: true}
	current := discovery.AutoupdateState{
		Installed: true,
		Running:   true,
		Interval:  config.AutoUpdateInterval{Seconds: config.DefaultAutoUpdateSeconds},
	}
	p := BuildAutoUpdatePlan(desired, current)
	types := ActionTypes(p)
	if len(types) != 2 || types[0] != ActionBrewAutoupdateDelete || types[1] != ActionBrewAutoupdateStart {
		t.Fatalf("types = %v, want delete then start", types)
	}
}

func TestBuildAutoUpdatePlan_StartWhenStoppedMatching(t *testing.T) {
	desired := config.AutoUpdate{Configured: true, Enable: true}
	current := discovery.AutoupdateState{
		Installed: true,
		Running:   false,
		Interval:  config.AutoUpdateInterval{Seconds: config.DefaultAutoUpdateSeconds},
	}
	p := BuildAutoUpdatePlan(desired, current)
	if len(p.Actions) != 1 || p.Actions[0].Type != ActionBrewAutoupdateStart {
		t.Fatalf("actions = %#v, want start only", p.Actions)
	}
}

func TestBuildAutoUpdatePlan_DeleteWhenDisabled(t *testing.T) {
	p := BuildAutoUpdatePlan(
		config.AutoUpdate{Configured: true, Enable: false},
		discovery.AutoupdateState{Installed: true, Running: true},
	)
	if len(p.Actions) != 1 || p.Actions[0].Type != ActionBrewAutoupdateDelete {
		t.Fatalf("actions = %#v", p.Actions)
	}
}

func TestBuildAutoUpdatePlan_UnconfiguredEmpty(t *testing.T) {
	p := BuildAutoUpdatePlan(config.AutoUpdate{}, discovery.AutoupdateState{Installed: true, Running: true})
	if len(p.Actions) != 0 {
		t.Fatalf("unmanaged should not touch running autoupdate: %#v", p.Actions)
	}
}
