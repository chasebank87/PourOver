package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

type stubBrewRunner struct {
	formulae         string
	casks            string
	outdatedFormulae string
	outdatedCasks    string
	outdatedSet      bool
}

func (s *stubBrewRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if isBrewListArgs(args, "--formula") {
		return []byte(s.formulae), nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return []byte(s.formulae), nil
	}
	if isBrewListArgs(args, "--cask") {
		return []byte(s.casks), nil
	}
	if len(args) == 3 && args[0] == "outdated" && args[1] == "--formula" && args[2] == "-q" {
		if s.outdatedSet {
			return []byte(s.outdatedFormulae), nil
		}
		return []byte(s.formulae), nil
	}
	if len(args) == 3 && args[0] == "outdated" && args[1] == "--cask" && args[2] == "-q" {
		if s.outdatedSet {
			return []byte(s.outdatedCasks), nil
		}
		return []byte(s.casks), nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func isBrewListArgs(args []string, kind string) bool {
	if len(args) == 2 && args[0] == "list" && args[1] == kind {
		return true
	}
	return len(args) == 3 && args[0] == "list" && args[1] == kind && args[2] == "-1"
}

func TestBuildPlan_FromFixture(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "..", "test", "fixtures", "config", "valid")
	data, err := os.ReadFile(filepath.Join(src, "pourover.lua"))
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Touch link source so file discovery succeeds.
	if err := os.WriteFile(filepath.Join(configDir, "config", "nvim", "init.lua"), []byte("--"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{formulae: "git\n", casks: "raycast\n"}
	p, err := BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 1 || names[0] != "fzf" {
		t.Fatalf("formula installs = %v", names)
	}
}

func TestBuildPlan_ManagedAndUnlink(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	srcContent := []byte("managed-body\n")
	if err := os.WriteFile(filepath.Join(configDir, "config", "foo.conf"), srcContent, 0o644); err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	managedTarget := filepath.Join(home, ".config", "foo.conf")
	if err := os.MkdirAll(filepath.Dir(managedTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	unlinkPath := filepath.Join(home, ".old-dotfile")
	if err := os.WriteFile(unlinkPath, []byte("remove-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	lua := `return {
  packages = { formulae = {}, casks = {} },
  files = {
    managed = {
      { source = "config/foo.conf", target = "~/.config/foo.conf" },
    },
    unlink = { "~/.old-dotfile" },
  },
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{}
	p, err := BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionManagedCopy); len(names) != 1 || names[0] != "~/.config/foo.conf" {
		t.Fatalf("managed copies = %v", names)
	}
	if names := plan.ActionNames(p, plan.ActionFileUnlink); len(names) != 1 || names[0] != "~/.old-dotfile" {
		t.Fatalf("unlinks = %v", names)
	}
}
