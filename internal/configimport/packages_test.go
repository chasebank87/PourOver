package configimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestFormatPackagesLua(t *testing.T) {
	got := FormatPackagesLua([]string{"fzf", "git"}, []string{"raycast"})
	for _, frag := range []string{`"fzf"`, `"git"`, `"raycast"`, "taps = {", "formulae = {", "casks = {"} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in:\n%s", frag, got)
		}
	}
	if strings.Index(got, `"fzf"`) > strings.Index(got, `"git"`) {
		t.Fatal("formulae not sorted")
	}
}

func TestFormatPackagesLuaFull_IncludesTaps(t *testing.T) {
	got := FormatPackagesLuaFull(
		[]string{"nikitabobko/tap", "homebrew/cask-fonts"},
		[]string{"git"},
		nil,
	)
	if !strings.Contains(got, `"homebrew/cask-fonts"`) || !strings.Contains(got, `"nikitabobko/tap"`) {
		t.Fatalf("missing taps in:\n%s", got)
	}
	if strings.Index(got, `"homebrew/cask-fonts"`) > strings.Index(got, `"nikitabobko/tap"`) {
		t.Fatal("taps not sorted")
	}
	// Import format emits plain quoted strings (implicit trusted), never tap tables.
	if strings.Contains(got, "trusted") || strings.Contains(got, "name =") {
		t.Fatalf("expected plain string taps, got:\n%s", got)
	}
}

func TestFormatPackagesLuaFull_RoundTripLoadManifest(t *testing.T) {
	dir := t.TempDir()
	body := FormatPackagesLuaFull(
		[]string{"oven-sh/bun", "homebrew/cask-fonts"},
		[]string{"git"},
		[]string{"raycast"},
	)
	if err := os.WriteFile(filepath.Join(dir, "packages.lua"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	root := `
local packages = require("packages")
return {
  packages = packages,
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := config.LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(manifest.Packages.Taps) != 2 {
		t.Fatalf("taps = %#v, want 2", manifest.Packages.Taps)
	}
	// Format sorts taps; both must decode as TapSpec with Trusted default true.
	for _, tap := range manifest.Packages.Taps {
		if tap.Name == "" {
			t.Fatalf("empty tap name in %#v", manifest.Packages.Taps)
		}
		if !tap.Trusted {
			t.Errorf("tap %q Trusted = false, want true (string tap default)", tap.Name)
		}
	}
	names := manifest.Packages.TapNames()
	if names[0] != "homebrew/cask-fonts" || names[1] != "oven-sh/bun" {
		t.Fatalf("TapNames = %v, want sorted names", names)
	}
	if len(manifest.Packages.Formulae) != 1 || manifest.Packages.Formulae[0] != "git" {
		t.Fatalf("formulae = %#v", manifest.Packages.Formulae)
	}
	if len(manifest.Packages.Casks) != 1 || manifest.Packages.Casks[0] != "raycast" {
		t.Fatalf("casks = %#v", manifest.Packages.Casks)
	}
}
