package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestRunDoctor_AllOK(t *testing.T) {
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

	report, err := runDoctorChecks(doctorInputs{
		configPath:  configPath,
		stateDir:    stateDir,
		manifest:    mustLoadDoctor(t, configPath),
		brewOK:      true,
		brewErr:     "",
		pouroverOK:  true,
		pouroverErr: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Fatalf("report not OK: %+v", report.Checks)
	}
}

func TestRunDoctor_BrewMissing(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { packages = { formulae = {} }, policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := runDoctorChecks(doctorInputs{
		configPath:  configPath,
		stateDir:    filepath.Join(root, "state"),
		manifest:    mustLoadDoctor(t, configPath),
		brewOK:      false,
		brewErr:     "brew not found",
		pouroverOK:  true,
		pouroverErr: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected failure when brew missing")
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "brew" && !c.OK && strings.Contains(c.Detail, "brew not found") {
			found = true
		}
	}
	if !found {
		t.Fatalf("checks = %+v", report.Checks)
	}
}

func TestRunDoctor_CapitalizedPackageName(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git" }, casks = { "Raycast" } },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := runDoctorChecks(doctorInputs{
		configPath: configPath,
		stateDir:   filepath.Join(root, "state"),
		manifest:   config.Manifest{},
		brewOK:     true,
		pouroverOK: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.OK() {
		t.Fatal("expected failure for capitalized cask name")
	}
	found := false
	for _, c := range report.Checks {
		if c.Name == "packages" && !c.OK && strings.Contains(c.Detail, "Raycast") && strings.Contains(c.Detail, "raycast") {
			found = true
		}
	}
	if !found {
		t.Fatalf("checks = %+v", report.Checks)
	}
}

func TestNewDoctorCmd(t *testing.T) {
	cmd := NewDoctorCmd()
	if cmd.Use != "doctor" {
		t.Fatalf("use = %q", cmd.Use)
	}
}

func mustLoadDoctor(t *testing.T, path string) config.Manifest {
	t.Helper()
	m, err := config.LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
