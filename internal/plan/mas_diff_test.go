package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildMasPlan_InstallMissing(t *testing.T) {
	plan := BuildMasPlan(
		config.Packages{
			MasConfigured: true,
			Mas: []config.MasApp{
				{Name: "Xcode", ID: 497799835},
				{Name: "WhatsApp", ID: 310633997},
			},
		},
		discovery.MasState{
			Apps: []discovery.MasInstalled{
				{ID: 310633997, Name: "WhatsApp Messenger"},
			},
		},
	)

	if got := ActionTypes(plan); len(got) != 1 || got[0] != ActionMasInstall {
		t.Fatalf("action types = %v, want [mas_install]", got)
	}
	a := plan.Actions[0]
	if a.Name != "Xcode" || a.Value != "497799835" {
		t.Fatalf("install action = %+v, want Name=Xcode Value=497799835", a)
	}
}

func TestBuildMasPlan_RemoveUndeclared(t *testing.T) {
	plan := BuildMasPlan(
		config.Packages{
			MasConfigured: true,
			Mas: []config.MasApp{
				{Name: "Xcode", ID: 497799835},
			},
		},
		discovery.MasState{
			Apps: []discovery.MasInstalled{
				{ID: 497799835, Name: "Xcode"},
				{ID: 310633997, Name: "WhatsApp Messenger"},
			},
		},
	)

	if got := ActionTypes(plan); len(got) != 1 || got[0] != ActionMasRemove {
		t.Fatalf("action types = %v, want [mas_remove]", got)
	}
	a := plan.Actions[0]
	if a.Name != "WhatsApp Messenger" || a.Value != "310633997" {
		t.Fatalf("remove action = %+v, want Name from list + Value=id", a)
	}
}

func TestBuildMasPlan_UnmanagedEmpty(t *testing.T) {
	plan := BuildMasPlan(
		config.Packages{
			MasConfigured: false,
			Mas: []config.MasApp{
				{Name: "Xcode", ID: 497799835},
			},
		},
		discovery.MasState{
			Apps: []discovery.MasInstalled{
				{ID: 310633997, Name: "WhatsApp Messenger"},
				{ID: 409183694, Name: "Keynote"},
			},
		},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("unmanaged plan = %v, want empty", plan.Actions)
	}
}

func TestBuildMasPlan_SortedByID(t *testing.T) {
	plan := BuildMasPlan(
		config.Packages{
			MasConfigured: true,
			Mas: []config.MasApp{
				{Name: "Xcode", ID: 497799835},
				{Name: "Keynote", ID: 409183694},
				{Name: "WhatsApp", ID: 310633997},
			},
		},
		discovery.MasState{
			Apps: []discovery.MasInstalled{
				{ID: 409183694, Name: "Keynote"},
				{ID: 100000001, Name: "Old App Z"},
				{ID: 100000000, Name: "Old App A"},
			},
		},
	)

	got := ActionTypes(plan)
	want := []ActionType{
		ActionMasInstall, ActionMasInstall,
		ActionMasRemove, ActionMasRemove,
	}
	if len(got) != len(want) {
		t.Fatalf("types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("types[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	installs := ActionNames(plan, ActionMasInstall)
	if len(installs) != 2 || installs[0] != "WhatsApp" || installs[1] != "Xcode" {
		t.Fatalf("installs by ID = %v, want WhatsApp then Xcode", installs)
	}
	if plan.Actions[0].Value != "310633997" || plan.Actions[1].Value != "497799835" {
		t.Fatalf("install values = %s, %s", plan.Actions[0].Value, plan.Actions[1].Value)
	}

	removes := ActionNames(plan, ActionMasRemove)
	if len(removes) != 2 || removes[0] != "Old App A" || removes[1] != "Old App Z" {
		t.Fatalf("removes by ID = %v, want Old App A then Old App Z", removes)
	}
	if plan.Actions[2].Value != "100000000" || plan.Actions[3].Value != "100000001" {
		t.Fatalf("remove values = %s, %s", plan.Actions[2].Value, plan.Actions[3].Value)
	}
}

func TestBuildMasPlan_NoopWhenInSync(t *testing.T) {
	plan := BuildMasPlan(
		config.Packages{
			MasConfigured: true,
			Mas: []config.MasApp{
				{Name: "Xcode", ID: 497799835},
			},
		},
		discovery.MasState{
			Apps: []discovery.MasInstalled{
				{ID: 497799835, Name: "Xcode"},
			},
		},
	)
	if len(plan.Actions) != 0 {
		t.Fatalf("expected no actions, got %v", plan.Actions)
	}
}

func TestRenderText_MasActions(t *testing.T) {
	p := Plan{Actions: []Action{
		{Type: ActionMasInstall, Name: "Xcode", Value: "497799835"},
		{Type: ActionMasRemove, Name: "WhatsApp Messenger", Value: "310633997"},
		{Type: ActionMasUpgrade, Name: "Keynote", Value: "409183694"},
	}}
	got := RenderText(p)
	want := "install mas Xcode (497799835)\n" +
		"remove mas WhatsApp Messenger (310633997)\n" +
		"upgrade mas Keynote (409183694)\n"
	if got != want {
		t.Fatalf("RenderText() =\n%q\nwant\n%q", got, want)
	}
}
