package configimport

import (
	"reflect"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestMergePackageLists(t *testing.T) {
	merged, added := MergePackageLists([]string{"git", "fzf"}, []string{"fzf", "wget", "curl"})
	wantMerged := []string{"curl", "fzf", "git", "wget"}
	wantAdded := []string{"curl", "wget"}
	if !reflect.DeepEqual(merged, wantMerged) {
		t.Fatalf("merged = %v, want %v", merged, wantMerged)
	}
	if !reflect.DeepEqual(added, wantAdded) {
		t.Fatalf("added = %v, want %v", added, wantAdded)
	}
}

func TestMergePackageLists_EmptyExisting(t *testing.T) {
	merged, added := MergePackageLists(nil, []string{"git"})
	if !reflect.DeepEqual(merged, []string{"git"}) || !reflect.DeepEqual(added, []string{"git"}) {
		t.Fatalf("merged=%v added=%v", merged, added)
	}
}

func TestMergeFileLinks(t *testing.T) {
	existing := []config.FileLink{
		{Source: "config/home/zshrc", Target: "~/.zshrc"},
		{Source: "config/nvim", Target: "~/.config/nvim"},
	}
	imported := []config.FileLink{
		{Source: "config/nvim", Target: "~/.config/nvim"}, // duplicate target
		{Source: "config/ghostty", Target: "~/.config/ghostty"},
		{Source: "config/home/gitconfig", Target: "~/.gitconfig"},
	}
	merged, added := MergeFileLinks(existing, imported)
	if len(merged) != 4 {
		t.Fatalf("merged len = %d, want 4: %+v", len(merged), merged)
	}
	if merged[0].Target != "~/.zshrc" || merged[1].Target != "~/.config/nvim" {
		t.Fatalf("existing order broken: %+v", merged)
	}
	// New targets appended sorted by Target: ~/.config/ghostty then ~/.gitconfig
	if merged[2].Target != "~/.config/ghostty" || merged[3].Target != "~/.gitconfig" {
		t.Fatalf("added order = %+v", merged[2:])
	}
	if len(added) != 2 || added[0].Target != "~/.config/ghostty" || added[1].Target != "~/.gitconfig" {
		t.Fatalf("added = %+v", added)
	}
}

func TestLinkTargets(t *testing.T) {
	got := LinkTargets([]config.FileLink{
		{Target: "~/.zshrc"},
		{Target: "~/.config/nvim"},
		{Target: ""},
	})
	if _, ok := got["~/.zshrc"]; !ok {
		t.Fatal("missing zshrc")
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
}
