package config

import (
	"path/filepath"
	"testing"
)

func TestLoadManifest_ValidFixture(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "valid", "pourover.lua")

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if manifest.Policy.UninstallMode != UninstallModeSafe {
		t.Errorf("policy.uninstall_mode = %q, want safe", manifest.Policy.UninstallMode)
	}
	if len(manifest.Packages.Formulae) != 2 || manifest.Packages.Formulae[0] != "git" {
		t.Errorf("packages.formulae = %#v", manifest.Packages.Formulae)
	}
	if len(manifest.Packages.Casks) != 1 || manifest.Packages.Casks[0] != "raycast" {
		t.Errorf("packages.casks = %#v", manifest.Packages.Casks)
	}
	if len(manifest.Files.Links) != 1 || manifest.Files.Links[0].Target != "~/.config/nvim" {
		t.Errorf("files.links = %#v", manifest.Files.Links)
	}
	if !manifest.Backup.ICloud.Enabled {
		t.Errorf("backup.icloud.enabled = false, want true")
	}
}
