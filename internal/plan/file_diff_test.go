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

func TestBuildManagedPlan_MissingAndDiffer(t *testing.T) {
	p, err := BuildManagedPlan([]discovery.ManagedStatus{
		{
			File:       config.ManagedFile{Source: "config/a", Target: "~/.config/a"},
			Kind:       discovery.ManagedStatusMissing,
			SourcePath: "/cfg/config/a",
			TargetPath: "/home/user/.config/a",
		},
		{
			File:       config.ManagedFile{Source: "config/b", Target: "~/.config/b"},
			Kind:       discovery.ManagedStatusDiffer,
			SourcePath: "/cfg/config/b",
			TargetPath: "/home/user/.config/b",
		},
		{
			File: config.ManagedFile{Source: "config/c", Target: "~/.config/c"},
			Kind: discovery.ManagedStatusSame,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	types := ActionTypes(p)
	want := []ActionType{ActionManagedCopy, ActionManagedCopy}
	if len(types) != len(want) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("types[%d] = %v, want %v", i, types[i], want[i])
		}
	}
	if p.Actions[0].Name != "~/.config/a" || p.Actions[0].Source != "config/a" {
		t.Fatalf("action[0] = %#v", p.Actions[0])
	}
	if p.Actions[1].Name != "~/.config/b" || p.Actions[1].Source != "config/b" {
		t.Fatalf("action[1] = %#v", p.Actions[1])
	}
}

func TestBuildUnlinkPlan_RemoveOnly(t *testing.T) {
	p, err := BuildUnlinkPlan([]discovery.UnlinkStatus{
		{Path: "~/.gone", TargetPath: "/home/user/.gone", Kind: discovery.UnlinkStatusMissing},
		{Path: "~/.old", TargetPath: "/home/user/.old", Kind: discovery.UnlinkStatusRemove},
	})
	if err != nil {
		t.Fatal(err)
	}
	types := ActionTypes(p)
	if len(types) != 1 || types[0] != ActionFileUnlink {
		t.Fatalf("types = %v, want [file_unlink]", types)
	}
	if p.Actions[0].Name != "~/.old" {
		t.Fatalf("name = %q", p.Actions[0].Name)
	}
}

func TestRenderText_ManagedAndUnlink(t *testing.T) {
	got := RenderText(Plan{Actions: []Action{
		{Type: ActionManagedCopy, Name: "~/.config/foo", Source: "config/foo"},
		{Type: ActionFileUnlink, Name: "~/.old-dotfile"},
	}})
	want := "managed copy ~/.config/foo <- config/foo\nunlink ~/.old-dotfile\n"
	if got != want {
		t.Fatalf("RenderText() = %q, want %q", got, want)
	}
}
