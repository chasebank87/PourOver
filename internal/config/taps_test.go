package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodePackages_TapStringTrustedDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
    taps = { "oven-sh/bun" },
    formulae = {},
    casks = {},
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(manifest.Packages.Taps) != 1 {
		t.Fatalf("taps = %#v, want 1", manifest.Packages.Taps)
	}
	tap := manifest.Packages.Taps[0]
	if tap.Name != "oven-sh/bun" {
		t.Errorf("Name = %q, want oven-sh/bun", tap.Name)
	}
	if !tap.Trusted {
		t.Errorf("Trusted = false, want true (string tap default)")
	}
}

func TestDecodePackages_TapTableTrustedFalse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
    taps = {
      { name = "heroku/brew", trusted = false },
    },
    formulae = {},
    casks = {},
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(manifest.Packages.Taps) != 1 {
		t.Fatalf("taps = %#v, want 1", manifest.Packages.Taps)
	}
	tap := manifest.Packages.Taps[0]
	if tap.Name != "heroku/brew" {
		t.Errorf("Name = %q, want heroku/brew", tap.Name)
	}
	if tap.Trusted {
		t.Errorf("Trusted = true, want false")
	}
}

func TestDecodePackages_TapTableWithURL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
    taps = {
      { name = "jundot/omlx", url = "https://github.com/jundot/omlx", trusted = true },
    },
    formulae = {},
    casks = {},
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(manifest.Packages.Taps) != 1 {
		t.Fatalf("taps = %#v, want 1", manifest.Packages.Taps)
	}
	tap := manifest.Packages.Taps[0]
	if tap.Name != "jundot/omlx" {
		t.Errorf("Name = %q", tap.Name)
	}
	if tap.URL != "https://github.com/jundot/omlx" {
		t.Errorf("URL = %q", tap.URL)
	}
	if !tap.Trusted {
		t.Errorf("Trusted = false, want true")
	}
}

func TestDecodePackages_TapTableMissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
    taps = {
      { trusted = true },
    },
    formulae = {},
    casks = {},
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for missing tap name, got nil")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("error = %v, want mention of name", err)
	}
}
