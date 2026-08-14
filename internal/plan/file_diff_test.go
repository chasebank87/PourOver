package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildFilePlan_MissingAndWrong(t *testing.T) {
	p, err := BuildFilePlan([]discovery.FileLinkStatus{
		{
			Link:       config.FileLink{Source: "config/a", Target: "~/.config/a"},
			Kind:       discovery.LinkStatusMissing,
			SourcePath: "/cfg/config/a",
			TargetPath: "/home/user/.config/a",
		},
		{
			Link:       config.FileLink{Source: "config/b", Target: "~/.config/b"},
			Kind:       discovery.LinkStatusWrong,
			SourcePath: "/cfg/config/b",
			TargetPath: "/home/user/.config/b",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	types := ActionTypes(p)
	want := []ActionType{ActionLinkCreate, ActionLinkUpdate}
	if len(types) != len(want) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("types[%d] = %v, want %v", i, types[i], want[i])
		}
	}
}

func TestBuildFilePlan_Blocked(t *testing.T) {
	_, err := BuildFilePlan([]discovery.FileLinkStatus{{
		Link: config.FileLink{Source: "config/x", Target: "~/.config/x"},
		Kind: discovery.LinkStatusBlocked,
	}})
	if err == nil {
		t.Fatal("expected error for blocked target")
	}
}

func TestMergePlans_BrewThenFiles(t *testing.T) {
	merged := MergePlans(
		Plan{Actions: []Action{{Type: ActionFormulaInstall, Name: "git"}}},
		Plan{Actions: []Action{{Type: ActionLinkCreate, Name: "~/.config/nvim", Source: "config/nvim"}}},
	)
	got := ActionTypes(merged)
	want := []ActionType{ActionFormulaInstall, ActionLinkCreate}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
