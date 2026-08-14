package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_InvalidPolicy(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "invalid_policy.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "policy.uninstall_mode")
}

func TestLoadManifest_EmptyFormula(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_formula.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "packages.formulae[2]")
}

func TestLoadManifest_EmptyLinkTarget(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_link_target.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "files.links[1].target")
}

func TestValidate_UnknownUninstallMode(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallMode("bogus")},
	})
	if err == nil || !strings.Contains(err.Error(), "policy.uninstall_mode") {
		t.Fatalf("Validate() = %v, want uninstall_mode error", err)
	}
}

func TestValidate_CapitalizedPackageName(t *testing.T) {
	err := Validate(&Manifest{
		Policy:   Policy{UninstallMode: UninstallModeSafe},
		Packages: Packages{Casks: []string{"Raycast"}},
	})
	if err == nil || !strings.Contains(err.Error(), "lowercase Homebrew token") || !strings.Contains(err.Error(), "raycast") {
		t.Fatalf("Validate() = %v", err)
	}
}

func assertLoadErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want substring %q", err.Error(), substr)
	}
}
