package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestUpgradePackages_Formula(t *testing.T) {
	runner := &recordingUpgradeRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaUpgrade, Name: "git"},
	}}
	result, err := UpgradePackages(context.Background(), runner, p, UpgradeOptions{Quiet: true})
	if err != nil {
		t.Fatalf("UpgradePackages: %v", err)
	}
	if result.Upgraded != 1 {
		t.Fatalf("Upgraded = %d, want 1", result.Upgraded)
	}
	if len(runner.upgrades) != 1 || runner.upgrades[0] != "git" {
		t.Fatalf("upgrades = %v, want [git]", runner.upgrades)
	}
}

func TestBuildUpgradePlan_FromFixture(t *testing.T) {
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
	runner := &stubBrewRunner{
		formulae:         "git\n",
		casks:            "raycast\n",
		outdatedFormulae: "git\n",
		outdatedCasks:    "raycast\n",
		outdatedSet:      true,
	}
	p, err := BuildUpgradePlan(context.Background(), configPath, runner, nil)
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

func TestDoctor_AllOK(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	iCloudParent := filepath.Join(root, "Mobile Documents", "com~apple~CloudDocs")
	if err := os.MkdirAll(iCloudParent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	iCloudPath := filepath.Join(iCloudParent, "PourOver")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git" } },
  policy = { uninstall_mode = "safe" },
  backup = { icloud = { enabled = true, path = "`+iCloudPath+`" } },
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Doctor(DoctorInputs{
		ConfigPath: configPath,
		StateDir:   stateDir,
		Manifest:   m,
		BrewOK:     true,
		PouroverOK: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("report not OK: %+v", report.Checks)
	}
}

func TestDoctor_BrewMissing(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { packages = { formulae = {} }, policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Doctor(DoctorInputs{
		ConfigPath: configPath,
		StateDir:   filepath.Join(root, "state"),
		Manifest:   mustLoadManifest(t, configPath),
		BrewOK:     false,
		BrewErr:    "brew not found",
		PouroverOK: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected failure when brew missing")
	}
}

func TestBackupRestore_RoundTrip(t *testing.T) {
	stateDir := t.TempDir()
	iCloud := t.TempDir()
	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte(`{"manifest_hash":"a"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "last-plan.json"), []byte(`{"actions":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{Enabled: true, Path: iCloud}}}
	at := time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC)
	bak, err := Backup(context.Background(), BackupOptions{StateDir: stateDir, Manifest: m, Now: func() time.Time { return at }})
	if err != nil {
		t.Fatal(err)
	}
	if bak.LocalSnapshot == "" {
		t.Fatal("empty LocalSnapshot")
	}
	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte(`{"manifest_hash":"corrupt"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(context.Background(), RestoreOptions{
		StateDir: stateDir,
		Manifest: m,
		Snapshot: bak.LocalSnapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.SnapshotPath != bak.LocalSnapshot {
		t.Fatalf("SnapshotPath = %q", restored.SnapshotPath)
	}
	data, err := os.ReadFile(filepath.Join(stateDir, "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"a"`) {
		t.Fatalf("lock after restore = %s", data)
	}
}

func TestImport_PackagesMerge(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return {
  taps = {},
  formulae = { "git", "neofetch" },
  casks = {},
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`local packages = require("packages")
return {
  packages = packages,
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &stubBrewRunner{formulae: "git\nfzf\n", casks: ""}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		Force:      false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.PackagesDone {
		t.Fatal("PackagesDone = false")
	}
	got := strings.Join(result.Formulae, ",")
	if !strings.Contains(got, "neofetch") || !strings.Contains(got, "fzf") || !strings.Contains(got, "git") {
		t.Fatalf("formulae = %v", result.Formulae)
	}
	if len(result.AddedFormulae) != 1 || result.AddedFormulae[0] != "fzf" {
		t.Fatalf("AddedFormulae = %v", result.AddedFormulae)
	}
}

func TestImport_SkippedLinks(t *testing.T) {
	cfgDir := t.TempDir()
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("alias x=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return { taps = {}, formulae = {}, casks = {} }`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(`local packages = require("packages")
return {
  packages = packages,
  files = { links = { { source = "config/home/zshrc", target = "~/.zshrc" } } },
  policy = { uninstall_mode = "safe" },
}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Import(context.Background(), nil, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   false,
		Files:      true,
		DryRun:     true,
		Force:      false,
		Home:       home,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.SkippedLinks) != 1 || result.SkippedLinks[0] != "~/.zshrc" {
		t.Fatalf("SkippedLinks = %v, want [~/.zshrc]", result.SkippedLinks)
	}
	foundGit := false
	for _, line := range result.FileLines {
		if line.TargetDecl == "~/.gitconfig" {
			foundGit = true
		}
		if line.TargetDecl == "~/.zshrc" {
			t.Fatal("declared ~/.zshrc should not appear in FileLines")
		}
	}
	if !foundGit {
		t.Fatalf("FileLines = %+v, want ~/.gitconfig", result.FileLines)
	}
}

type recordingUpgradeRunner struct {
	upgrades []string
}

func (r *recordingUpgradeRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "upgrade" {
		r.upgrades = append(r.upgrades, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "upgrade" && args[1] == "--cask" {
		r.upgrades = append(r.upgrades, args[2])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func mustLoadManifest(t *testing.T, path string) config.Manifest {
	t.Helper()
	m, err := config.LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
