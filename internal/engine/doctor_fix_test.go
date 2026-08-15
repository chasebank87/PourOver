package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestEnsureStateDir_CreatesDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	stateDir := filepath.Join(root, "nested", "state")
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Fatalf("state dir should not exist yet: %v", err)
	}

	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatalf("EnsureStateDir: %v", err)
	}

	info, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("stat state dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected state path to be a directory")
	}
}

func TestEnsureStateDir_Idempotent(t *testing.T) {
	t.Parallel()

	stateDir := filepath.Join(t.TempDir(), "state")
	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatal(err)
	}
	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatalf("second EnsureStateDir: %v", err)
	}
}

func TestInitConfig_CreatesScaffold(t *testing.T) {
	t.Parallel()

	cfgDir := filepath.Join(t.TempDir(), ".pourover")
	if err := InitConfig(cfgDir, false); err != nil {
		t.Fatalf("InitConfig: %v", err)
	}

	root := filepath.Join(cfgDir, "pourover.lua")
	packages := filepath.Join(cfgDir, "packages.lua")
	for _, path := range []string{root, packages} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	if _, err := config.LoadManifest(root); err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
}

func TestInitConfig_RefusesExistingWithoutForce(t *testing.T) {
	t.Parallel()

	cfgDir := filepath.Join(t.TempDir(), ".pourover")
	if err := InitConfig(cfgDir, false); err != nil {
		t.Fatal(err)
	}
	err := InitConfig(cfgDir, false)
	if err == nil {
		t.Fatal("expected error when config exists")
	}
	if !strings.Contains(err.Error(), "--force") && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %q, want refuse-existing message", err)
	}
}
