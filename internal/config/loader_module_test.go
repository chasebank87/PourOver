package config

import (
	"path/filepath"
	"testing"
)

func TestLoadManifest_WithRequireModule(t *testing.T) {
	dir := filepath.Join("..", "..", "test", "fixtures", "config", "valid_with_module")
	path := filepath.Join(dir, "pourover.lua")

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if len(manifest.Packages.Taps) != 1 || manifest.Packages.Taps[0] != "homebrew/cask-fonts" {
		t.Errorf("packages.taps = %#v", manifest.Packages.Taps)
	}
	if len(manifest.Packages.Formulae) != 2 || manifest.Packages.Formulae[0] != "jq" {
		t.Errorf("packages.formulae = %#v", manifest.Packages.Formulae)
	}
	if len(manifest.Packages.Casks) != 1 || manifest.Packages.Casks[0] != "1password" {
		t.Errorf("packages.casks = %#v", manifest.Packages.Casks)
	}
}
