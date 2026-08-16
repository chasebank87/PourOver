package plan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

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

func TestBuildFilePrunePlan_NonDestructiveEmpty(t *testing.T) {
	dir := t.TempDir()
	extra := filepath.Join(dir, "extra")
	if err := os.WriteFile(extra, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	declared := filepath.Join(dir, "keep")
	p := BuildFilePrunePlan([]string{extra}, []string{declared}, config.FilesModeNonDestructive)
	if len(p.Actions) != 0 {
		t.Fatalf("expected no prune actions, got %#v", p.Actions)
	}
}

func TestBuildFilePrunePlan_SafeAndStrictPruneOwnedExtras(t *testing.T) {
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep")
	extra := filepath.Join(dir, "extra")
	missing := filepath.Join(dir, "missing")
	for _, path := range []string{keep, extra} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	owned := []string{keep, extra, missing}
	declared := []string{keep}

	for _, mode := range []config.FilesMode{config.FilesModeSafe, config.FilesModeStrict} {
		p := BuildFilePrunePlan(owned, declared, mode)
		names := ActionNames(p, ActionFilePrune)
		if len(names) != 1 || names[0] != extra {
			t.Fatalf("mode %s: prune names = %v, want [%s]", mode, names, extra)
		}
	}
}

func TestBuildFilePrunePlan_OwnedEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orphan")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := BuildFilePrunePlan(nil, nil, config.FilesModeSafe)
	if len(p.Actions) != 0 {
		t.Fatalf("expected no prune when owned empty, got %#v", p.Actions)
	}
}

func TestBuildFilePrunePlan_OldLockEmptyOwned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orphan")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Old locks load with nil/empty OwnedFiles — never invent prune candidates.
	p := BuildFilePrunePlan([]string{}, []string{}, config.FilesModeStrict)
	if len(p.Actions) != 0 {
		t.Fatalf("expected no prune for empty owned, got %#v", p.Actions)
	}
}

func TestBuildFilePrunePlan_SkipsExplicitUnlinkTargets(t *testing.T) {
	dir := t.TempDir()
	unlinked := filepath.Join(dir, "unlinked")
	if err := os.WriteFile(unlinked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Unlink paths are passed in declaredTargets so we do not also emit file_prune.
	p := BuildFilePrunePlan([]string{unlinked}, []string{unlinked}, config.FilesModeSafe)
	if len(p.Actions) != 0 {
		t.Fatalf("expected no prune for unlink target, got %#v", p.Actions)
	}
}

func TestRenderText_ManagedAndUnlink(t *testing.T) {
	got := RenderText(Plan{Actions: []Action{
		{Type: ActionManagedCopy, Name: "~/.config/foo", Source: "config/foo"},
		{Type: ActionFileUnlink, Name: "~/.old-dotfile"},
		{Type: ActionFilePrune, Name: "~/.config/old"},
		{Type: ActionLinkReplace, Name: "~/.zshrc", Source: "config/zshrc"},
	}})
	want := "managed copy ~/.config/foo <- config/foo\nunlink ~/.old-dotfile\nprune file ~/.config/old\nreplace file ~/.zshrc <- config/zshrc (backup)\n"
	if got != want {
		t.Fatalf("RenderText() = %q, want %q", got, want)
	}
}
