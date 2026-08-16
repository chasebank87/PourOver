package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

func TestNewUpgradeCmd_HasFlags(t *testing.T) {
	cmd := NewUpgradeCmd()
	if cmd.Flags().Lookup("dry-run") == nil || cmd.Flags().Lookup("yes") == nil || cmd.Flags().Lookup("skip-self-update") == nil || cmd.Flags().Lookup("quiet") == nil {
		t.Fatal("upgrade missing flags")
	}
}

func TestBuildUpgradePlanForTest_FromFixture(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = { "raycast" } },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &stubBrewRunner{formulae: "git\n", casks: "raycast\n"}
	p, err := buildUpgradePlanForTest(context.Background(), configPath, runner)
	if err != nil {
		t.Fatal(err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaUpgrade); len(names) != 1 || names[0] != "git" {
		t.Fatalf("formula upgrades = %v", names)
	}
	if names := plan.ActionNames(p, plan.ActionCaskUpgrade); len(names) != 1 || names[0] != "raycast" {
		t.Fatalf("cask upgrades = %v", names)
	}
}

func TestUpgradeDryRun_MergesPlans(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &stubBrewRunner{formulae: "git\n", casks: ""}
	up, err := buildUpgradePlanForTest(context.Background(), configPath, runner)
	if err != nil {
		t.Fatal(err)
	}
	ap, _ := isolatedPlan(t, configPath, runner)
	combined := plan.MergePlans(up, ap)
	types := plan.ActionTypes(combined)
	if len(types) < 2 || types[0] != plan.ActionFormulaUpgrade || types[1] != plan.ActionFormulaInstall {
		t.Fatalf("combined types = %v, want upgrade then install", types)
	}
	_ = cobra.Command{}
}
