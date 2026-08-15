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
	if strings.Contains(got, "mas =") {
		t.Fatalf("nil mas should omit mas section, got:\n%s", got)
	}
}

func TestFormatPackagesLuaFull_MasSection(t *testing.T) {
	got := FormatPackagesLuaFull(
		nil,
		[]string{"git"},
		[]string{"raycast"},
		[]config.MasApp{
			{Name: "1Password for Safari", ID: 1569813296},
			{Name: "Xcode", ID: 497799835},
		},
	)
	if !strings.Contains(got, "mas = {") {
		t.Fatalf("missing mas section:\n%s", got)
	}
	if !strings.Contains(got, `["1Password for Safari"] = 1569813296`) {
		t.Fatalf("missing quoted mas key:\n%s", got)
	}
	if !strings.Contains(got, "Xcode = 497799835") {
		t.Fatalf("missing bare mas key:\n%s", got)
	}
	// Sorted by ID: Xcode (497799835) before 1Password (1569813296).
	xcodeIdx := strings.Index(got, "Xcode = 497799835")
	onePwIdx := strings.Index(got, `["1Password for Safari"] = 1569813296`)
	if xcodeIdx < 0 || onePwIdx < 0 || xcodeIdx > onePwIdx {
		t.Fatalf("mas apps not sorted by ID:\n%s", got)
	}
	casksIdx := strings.Index(got, "casks = {")
	masIdx := strings.Index(got, "mas = {")
	if casksIdx < 0 || masIdx < 0 || casksIdx > masIdx {
		t.Fatalf("mas should follow casks:\n%s", got)
	}
}

func TestFormatPackagesLuaFull_MasEmptyConfigured(t *testing.T) {
	got := FormatPackagesLuaFull(nil, nil, nil, []config.MasApp{})
	if !strings.Contains(got, "mas = {") {
		t.Fatalf("empty non-nil mas should emit mas = {}, got:\n%s", got)
	}
}

func TestFormatPackagesLuaFull_RoundTripLoadManifest(t *testing.T) {
	dir := t.TempDir()
	body := FormatPackagesLuaFull(
		[]string{"oven-sh/bun", "homebrew/cask-fonts"},
		[]string{"git"},
		[]string{"raycast"},
		[]config.MasApp{
			{Name: "1Password for Safari", ID: 1569813296},
			{Name: "Xcode", ID: 497799835},
		},
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
	if !manifest.Packages.MasConfigured {
		t.Fatal("MasConfigured = false, want true")
	}
	if len(manifest.Packages.Mas) != 2 {
		t.Fatalf("Mas = %#v, want 2", manifest.Packages.Mas)
	}
	if manifest.Packages.Mas[0].Name != "Xcode" || manifest.Packages.Mas[0].ID != 497799835 {
		t.Fatalf("Mas[0] = %#v", manifest.Packages.Mas[0])
	}
	if manifest.Packages.Mas[1].Name != "1Password for Safari" || manifest.Packages.Mas[1].ID != 1569813296 {
		t.Fatalf("Mas[1] = %#v", manifest.Packages.Mas[1])
	}
}
