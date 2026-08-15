package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

type stubBrewRunner struct {
	formulae string
	casks    string
}

func (s *stubBrewRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if len(args) == 2 && args[0] == "list" && args[1] == "--formula" {
		return []byte(s.formulae), nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return []byte(s.formulae), nil
	}
	if len(args) == 2 && args[0] == "list" && args[1] == "--cask" {
		return []byte(s.casks), nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestBuildPlan_FromFixture(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = { "raycast" } },
  files = {
    links = { { source = "config/nvim", target = "`+filepath.Join(root, "tgt")+`" } },
  },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{
		formulae: "git\n",
		casks:    "",
	}

	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 1 || names[0] != "fzf" {
		t.Errorf("formula installs = %v, want [fzf]", names)
	}
	if types := plan.ActionTypes(p); len(types) < 2 || types[len(types)-1] != plan.ActionLinkCreate {
		t.Errorf("expected link create in plan, types = %v", types)
	}
}
