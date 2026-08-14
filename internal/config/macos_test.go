package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_MacOSDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = { formulae = {}, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
  macos = {
    defaults = {
      dock = {
        autohide = true,
        orientation = "left",
        ["show-recents"] = false,
        tilesize = 48,
      },
      finder = {
        ShowPathbar = true,
      },
      NSGlobalDomain = {
        AppleShowAllExtensions = true,
        ["com.apple.swipescrolldirection"] = false,
      },
      screencapture = {
        type = "png",
      },
      trackpad = {
        Clicking = true,
      },
      custom = {
        ["com.apple.Safari"] = {
          ShowFullURLInSmartSearchField = true,
        },
      },
    },
  },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if !m.MacOS.Defaults.Sections["dock"]["autohide"].Bool {
		t.Fatalf("autohide = %#v", m.MacOS.Defaults.Sections["dock"]["autohide"])
	}
	if m.MacOS.Defaults.Sections["dock"]["orientation"].String != "left" {
		t.Fatalf("orientation = %#v", m.MacOS.Defaults.Sections["dock"]["orientation"])
	}
	if m.MacOS.Defaults.Sections["dock"]["show-recents"].Bool {
		t.Fatalf("show-recents = %#v", m.MacOS.Defaults.Sections["dock"]["show-recents"])
	}
	if m.MacOS.Defaults.Sections["dock"]["tilesize"].Int != 48 {
		t.Fatalf("tilesize = %#v", m.MacOS.Defaults.Sections["dock"]["tilesize"])
	}
	if !m.MacOS.Defaults.Sections["finder"]["ShowPathbar"].Bool {
		t.Fatal("ShowPathbar missing")
	}
	if m.MacOS.Defaults.Sections["screencapture"]["type"].String != "png" {
		t.Fatalf("screencapture = %#v", m.MacOS.Defaults.Sections["screencapture"])
	}
	if m.MacOS.Defaults.Sections["NSGlobalDomain"]["com.apple.swipescrolldirection"].Bool {
		t.Fatalf("swipescrolldirection = %#v", m.MacOS.Defaults.Sections["NSGlobalDomain"]["com.apple.swipescrolldirection"])
	}
	if !m.MacOS.Defaults.Sections["trackpad"]["Clicking"].Bool {
		t.Fatal("Clicking missing")
	}
	if !m.MacOS.Defaults.Custom["com.apple.Safari"]["ShowFullURLInSmartSearchField"].Bool {
		t.Fatalf("custom = %#v", m.MacOS.Defaults.Custom)
	}
}

func TestValidate_UnknownCuratedDockKey(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe},
		MacOS: MacOS{Defaults: MacOSDefaults{
			Sections: map[string]map[string]SettingValue{
				"dock": {
					"not-a-real-key": {Kind: SettingBool, Bool: true},
				},
			},
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown curated key") || !strings.Contains(err.Error(), "docs/macos-defaults.md") {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestFlattenDefaults_Order(t *testing.T) {
	got := FlattenDefaults(MacOSDefaults{
		Sections: map[string]map[string]SettingValue{
			"dock": {
				"tilesize": {Kind: SettingInt, Int: 48},
				"autohide": {Kind: SettingBool, Bool: true},
			},
		},
		Custom: map[string]map[string]SettingValue{
			"com.apple.screencapture": {
				"type": {Kind: SettingString, String: "png"},
			},
		},
	})
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Key != "autohide" || got[0].Domain != DomainDock {
		t.Fatalf("first = %#v", got[0])
	}
	if got[1].Key != "tilesize" {
		t.Fatalf("second = %#v", got[1])
	}
	if got[2].Domain != "com.apple.screencapture" {
		t.Fatalf("third = %#v", got[2])
	}
}

func TestCatalog_LoadsAndHasNixDarwinSections(t *testing.T) {
	if err := CatalogLoadError(); err != nil {
		t.Fatal(err)
	}
	if CatalogKeyCount() < 180 {
		t.Fatalf("key count = %d, want >= 180 nix-darwin scalars", CatalogKeyCount())
	}
	need := []string{
		".GlobalPreferences", "ActivityMonitor", "LaunchServices", "NSGlobalDomain",
		"SoftwareUpdate", "WindowManager", "controlcenter", "dock", "finder",
		"hitoolbox", "iCal", "loginwindow", "magicmouse", "menuExtraClock",
		"screencapture", "screensaver", "smb", "spaces", "trackpad", "universalaccess",
	}
	if len(Catalog()) != len(need) {
		t.Fatalf("sections = %d, want %d", len(Catalog()), len(need))
	}
	for _, name := range need {
		if !IsCatalogSection(name) {
			t.Errorf("missing section %s", name)
			continue
		}
		n := 0
		for _, s := range Catalog() {
			if s.Name == name {
				n = len(s.Keys)
			}
		}
		if n == 0 {
			t.Errorf("section %s has no keys", name)
		}
	}
	if !IsCuratedKey("dock", "autohide") {
		t.Fatal("dock.autohide missing")
	}
	if IsCuratedKey("dock", "persistent-apps") {
		t.Fatal("persistent-apps should not be in scalar catalog")
	}
}

func TestFlattenDefaults_SystemDomainAndAlsoDomains(t *testing.T) {
	got := FlattenDefaults(MacOSDefaults{
		Sections: map[string]map[string]SettingValue{
			"loginwindow": {
				"GuestEnabled": {Kind: SettingBool, Bool: false},
			},
			"trackpad": {
				"Clicking": {Kind: SettingBool, Bool: true},
			},
		},
	})
	var login, track []DesiredSetting
	for _, s := range got {
		switch s.Section {
		case "loginwindow":
			login = append(login, s)
		case "trackpad":
			track = append(track, s)
		}
	}
	if len(login) != 1 || login[0].Domain != "/Library/Preferences/com.apple.loginwindow" || login[0].Scope != "system" {
		t.Fatalf("loginwindow = %#v", login)
	}
	if len(track) != 2 {
		t.Fatalf("trackpad writes = %#v", track)
	}
	if track[0].Domain != "com.apple.AppleMultitouchTrackpad" || track[1].Domain != "com.apple.driver.AppleBluetoothMultitouch.trackpad" {
		t.Fatalf("trackpad domains = %s, %s", track[0].Domain, track[1].Domain)
	}
}

func TestLoadManifest_UnknownSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `return { macos = { defaults = { notASection = { foo = true } } } }`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "unknown section") || !strings.Contains(err.Error(), "docs/macos-defaults.md") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMacOSDefaultsMarkdown_IncludesKeys(t *testing.T) {
	md, err := RenderMacOSDefaultsMarkdown()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"macos.defaults.dock.autohide", "screencapture", "trackpad", "com.apple.swipescrolldirection", "Indexed"}
	for _, w := range want {
		if !strings.Contains(md, w) {
			t.Errorf("markdown missing %q", w)
		}
	}
}

func TestLuaPathFor(t *testing.T) {
	if got := luaPathFor("dock", "show-recents"); got != `macos.defaults.dock["show-recents"]` {
		t.Fatalf("got %s", got)
	}
	if got := luaPathFor(".GlobalPreferences", "com.apple.mouse.scaling"); got != `macos.defaults[".GlobalPreferences"]["com.apple.mouse.scaling"]` {
		t.Fatalf("got %s", got)
	}
}

func TestNixDarwinOptionsDoc_ListsAllMyNixOSRoots(t *testing.T) {
	md, err := os.ReadFile(filepath.Join("..", "..", "docs", "nix-darwin-options.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(md)
	roots := []string{
		"_module", "documentation", "environment", "fonts", "homebrew",
		"launchd", "networking", "nix", "nixpkgs", "power", "programs",
		"security", "services", "system", "time", "users", "lib",
	}
	for _, r := range roots {
		if !strings.Contains(body, r) {
			t.Errorf("docs/nix-darwin-options.md missing MyNixOS root %q", r)
		}
	}
}

func TestMacOSDefaultsDoc_MatchesGenerator(t *testing.T) {
	generated, err := RenderMacOSDefaultsMarkdown()
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(filepath.Join("..", "..", "docs", "macos-defaults.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(onDisk) != generated {
		t.Fatal("docs/macos-defaults.md is stale; run make macos-docs")
	}
	want := fmt.Sprintf("_Indexed %d keys in %d sections", CatalogKeyCount(), len(Catalog()))
	if !strings.Contains(generated, want) {
		t.Fatalf("markdown missing %q", want)
	}
}

func TestLoadManifest_RejectsCapitalizedBrewNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = { formulae = { "git" }, casks = { "Raycast", "warp" } },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "Raycast") || !strings.Contains(err.Error(), "raycast") {
		t.Fatalf("err=%v, want lowercase Homebrew token error for Raycast", err)
	}
}
