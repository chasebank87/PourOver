package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	mas := &stubMasRunner{err: &exec.Error{Name: "mas", Err: exec.ErrNotFound}}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		Force:      false,
		MasRunner:  mas,
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
	if len(result.Mas) != 0 {
		t.Fatalf("Mas = %#v, want empty when mas missing and unmanaged", result.Mas)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "packages.lua"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "mas =") {
		t.Fatalf("packages.lua should omit mas when soft-skipped unmanaged:\n%s", data)
	}
}

func TestImport_PackagesMasMerge(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return {
  taps = {},
  formulae = { "git" },
  casks = {},
  mas = { ["Xcode (kept)"] = 497799835 },
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
	runner := &stubBrewRunner{formulae: "git\n", casks: ""}
	mas := &stubMasRunner{list: "497799835 Xcode\n1569813296 1Password for Safari\n"}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		Force:      false,
		MasRunner:  mas,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mas) != 2 {
		t.Fatalf("Mas = %#v, want 2", result.Mas)
	}
	if result.Mas[0].Name != "Xcode (kept)" || result.Mas[0].ID != 497799835 {
		t.Fatalf("Mas[0] = %#v, want kept config name", result.Mas[0])
	}
	if len(result.AddedMas) != 1 || result.AddedMas[0].ID != 1569813296 {
		t.Fatalf("AddedMas = %#v", result.AddedMas)
	}
	data, err := os.ReadFile(filepath.Join(cfgDir, "packages.lua"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, `["Xcode (kept)"] = 497799835`) {
		t.Fatalf("kept name missing:\n%s", body)
	}
	if !strings.Contains(body, `["1Password for Safari"] = 1569813296`) {
		t.Fatalf("added app missing:\n%s", body)
	}
}

func TestImport_PackagesMasUnmanagedDiscovers(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return {
  taps = {},
  formulae = { "git" },
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
	runner := &stubBrewRunner{formulae: "git\n", casks: ""}
	mas := &stubMasRunner{list: "497799835 Xcode\n"}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		MasRunner:  mas,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mas) != 1 || result.Mas[0].ID != 497799835 {
		t.Fatalf("Mas = %#v", result.Mas)
	}
	m := mustLoadManifest(t, configPath)
	if !m.Packages.MasConfigured {
		t.Fatal("MasConfigured = false after import discovered apps")
	}
}

func TestImport_PackagesMasMissingSoftSkip(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return {
  taps = {},
  formulae = { "git" },
  casks = {},
  mas = { Xcode = 497799835 },
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
	mas := &stubMasRunner{err: &exec.Error{Name: "mas", Err: exec.ErrNotFound}}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		MasRunner:  mas,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AddedFormulae) != 1 || result.AddedFormulae[0] != "fzf" {
		t.Fatalf("AddedFormulae = %v (brew import should still succeed)", result.AddedFormulae)
	}
	if len(result.Mas) != 1 || result.Mas[0].ID != 497799835 {
		t.Fatalf("Mas = %#v, want existing preserved when mas missing", result.Mas)
	}
}

func TestImport_PackagesMasForce(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte(`return {
  taps = {},
  formulae = { "neofetch" },
  casks = {},
  mas = { Xcode = 497799835, WireGuard = 1451685025 },
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
	runner := &stubBrewRunner{formulae: "git\n", casks: ""}
	mas := &stubMasRunner{list: "1569813296 1Password for Safari\n"}
	result, err := Import(context.Background(), runner, ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   true,
		Files:      false,
		Force:      true,
		MasRunner:  mas,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mas) != 1 || result.Mas[0].ID != 1569813296 {
		t.Fatalf("Mas = %#v, want force-replaced discovered set", result.Mas)
	}
	if len(result.Formulae) != 1 || result.Formulae[0] != "git" {
		t.Fatalf("Formulae = %v, want force-replaced", result.Formulae)
	}
}

func TestBuildUpgradePlan_MasMissingSoftContinue(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = {
    formulae = { "git" },
    casks = {},
    mas = { Xcode = 497799835 },
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &stubBrewRunner{
		formulae:         "git\n",
		outdatedFormulae: "git\n",
		outdatedSet:      true,
	}
	mas := &stubMasRunner{err: &exec.Error{Name: "mas", Err: exec.ErrNotFound}}
	p, err := BuildUpgradePlan(context.Background(), configPath, runner, mas)
	if err != nil {
		t.Fatalf("BuildUpgradePlan: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaUpgrade); len(names) != 1 || names[0] != "git" {
		t.Fatalf("formula upgrades = %v", names)
	}
	if names := plan.ActionNames(p, plan.ActionMasUpgrade); len(names) != 0 {
		t.Fatalf("mas upgrades = %v, want none when mas missing", names)
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
		FilesAll:   true,
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
