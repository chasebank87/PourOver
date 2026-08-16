package configimport

import (
	"path/filepath"
	"testing"
)

func TestDefaultFileCandidateSelected(t *testing.T) {
	home := FileCandidate{TargetDecl: "~/.zshrc"}
	cfg := FileCandidate{TargetDecl: "~/.config/cursor"}
	if !DefaultFileCandidateSelected(home, false) {
		t.Fatal("home dotfile should default on")
	}
	if DefaultFileCandidateSelected(cfg, false) {
		t.Fatal("~/.config should default off")
	}
	if !DefaultFileCandidateSelected(cfg, true) {
		t.Fatal("managed ~/.config should default on")
	}
}

func TestFilterCandidatesByTargets(t *testing.T) {
	cands := []FileCandidate{
		{TargetDecl: "~/.zshrc", TargetPath: "/tmp/home/.zshrc"},
		{TargetDecl: "~/.config/nvim", TargetPath: "/tmp/home/.config/nvim"},
	}
	got := FilterCandidatesByTargets(cands, []string{"~/.config/nvim"})
	if len(got) != 1 || got[0].TargetDecl != "~/.config/nvim" {
		t.Fatalf("got %#v", got)
	}
	_ = filepath.Separator
}
