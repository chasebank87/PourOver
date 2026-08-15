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

func TestLoadManifest_ManagedAndUnlink(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "valid", "files_managed_unlink.lua")

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if len(manifest.Files.Managed) != 1 ||
		manifest.Files.Managed[0].Source != "config/foo.conf" ||
		manifest.Files.Managed[0].Target != "~/.config/foo.conf" {
		t.Errorf("files.managed = %#v", manifest.Files.Managed)
	}
	if len(manifest.Files.Unlink) != 1 || manifest.Files.Unlink[0] != "~/.old-dotfile" {
		t.Errorf("files.unlink = %#v", manifest.Files.Unlink)
	}
}

func TestLoadManifest_Templates(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "valid", "files_templates.lua")

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	if len(manifest.Files.Templates) != 1 ||
		manifest.Files.Templates[0].Source != "config/gitconfig.tmpl" ||
		manifest.Files.Templates[0].Target != "~/.gitconfig" {
		t.Errorf("files.templates = %#v", manifest.Files.Templates)
	}
}
