package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestInitConfigDir_CreatesScaffold(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".pourover")

	if err := InitConfigDir(cfgDir, false); err != nil {
		t.Fatalf("InitConfigDir: %v", err)
	}

	root := filepath.Join(cfgDir, "pourover.lua")
	packages := filepath.Join(cfgDir, "packages.lua")
	configExample := filepath.Join(cfgDir, "config", "nvim", ".keep")

	for _, path := range []string{root, packages, configExample} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}

	manifest, err := config.LoadManifest(root)
	if err != nil {
		t.Fatalf("LoadManifest scaffold: %v", err)
	}
	if manifest.Policy.UninstallMode != config.UninstallModeSafe {
		t.Fatalf("uninstall_mode = %q, want safe", manifest.Policy.UninstallMode)
	}
	if len(manifest.Packages.Formulae) == 0 {
		t.Fatal("expected example formulae in packages.lua")
	}
}

func TestInitConfigDir_RefusesExistingWithoutForce(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".pourover")
	if err := InitConfigDir(cfgDir, false); err != nil {
		t.Fatal(err)
	}

	err := InitConfigDir(cfgDir, false)
	if err == nil {
		t.Fatal("expected error when config exists")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %q, want mention of --force", err)
	}
}

func TestInitConfigDir_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".pourover")
	if err := InitConfigDir(cfgDir, false); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(root, []byte("-- stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InitConfigDir(cfgDir, true); err != nil {
		t.Fatalf("InitConfigDir --force: %v", err)
	}
	data, err := os.ReadFile(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "stale") {
		t.Fatal("force did not overwrite pourover.lua")
	}
	if _, err := config.LoadManifest(root); err != nil {
		t.Fatalf("LoadManifest after force: %v", err)
	}
}

func TestNewInitCmd_HasForceFlag(t *testing.T) {
	cmd := NewInitCmd()
	if cmd.Flags().Lookup("force") == nil {
		t.Fatal("missing --force flag")
	}
}
