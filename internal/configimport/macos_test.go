package configimport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestFormatMacOSLua_RoundTrip(t *testing.T) {
	d := config.MacOSDefaults{
		Sections: map[string]map[string]config.SettingValue{
			"loginwindow": {
				"LoginwindowText": {Kind: config.SettingString, String: "Found? Call 513-968-9283"},
			},
			"NSGlobalDomain": {
				"KeyRepeat": {Kind: config.SettingInt, Int: 2},
			},
			"dock": {
				"autohide":       {Kind: config.SettingBool, Bool: true},
				"show-recents":   {Kind: config.SettingBool, Bool: false},
				"persistent-apps": {Kind: config.SettingArray, Array: []string{"/Applications/Safari.app"}},
			},
		},
		Custom: map[string]map[string]config.SettingValue{
			"com.apple.Safari": {
				"ShowFullURLInSmartSearchField": {Kind: config.SettingBool, Bool: true},
			},
		},
	}

	body := FormatMacOSLua(d)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "macos.lua"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	root := `
return {
  packages = { formulae = {}, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
  macos = require("macos"),
}
`
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := config.LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v\nmacos.lua:\n%s", err, body)
	}
	got := m.MacOS.Defaults.Sections["loginwindow"]["LoginwindowText"]
	if got.Kind != config.SettingString || got.String != "Found? Call 513-968-9283" {
		t.Fatalf("LoginwindowText = %#v", got)
	}
	if m.MacOS.Defaults.Sections["NSGlobalDomain"]["KeyRepeat"].Int != 2 {
		t.Fatalf("KeyRepeat = %#v", m.MacOS.Defaults.Sections["NSGlobalDomain"]["KeyRepeat"])
	}
	if !m.MacOS.Defaults.Sections["dock"]["autohide"].Bool {
		t.Fatal("autohide missing")
	}
	if m.MacOS.Defaults.Sections["dock"]["show-recents"].Bool {
		t.Fatalf("show-recents = %#v", m.MacOS.Defaults.Sections["dock"]["show-recents"])
	}
	apps := m.MacOS.Defaults.Sections["dock"]["persistent-apps"]
	if apps.Kind != config.SettingArray || len(apps.Array) != 1 || apps.Array[0] != "/Applications/Safari.app" {
		t.Fatalf("persistent-apps = %#v", apps)
	}
	if !m.MacOS.Defaults.Custom["com.apple.Safari"]["ShowFullURLInSmartSearchField"].Bool {
		t.Fatalf("custom = %#v", m.MacOS.Defaults.Custom)
	}
}

func TestMergeMacOSDefaults_AddOnly(t *testing.T) {
	existing := config.MacOSDefaults{
		Sections: map[string]map[string]config.SettingValue{
			"dock": {
				"autohide": {Kind: config.SettingBool, Bool: true},
			},
		},
	}
	discovered := config.MacOSDefaults{
		Sections: map[string]map[string]config.SettingValue{
			"dock": {
				"autohide": {Kind: config.SettingBool, Bool: false}, // existing wins
			},
			"trackpad": {
				"Clicking": {Kind: config.SettingBool, Bool: true},
			},
		},
	}

	merged, added := MergeMacOSDefaults(existing, discovered, false)
	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}
	if !merged.Sections["dock"]["autohide"].Bool {
		t.Fatalf("existing dock.autohide should win: %#v", merged.Sections["dock"]["autohide"])
	}
	if !merged.Sections["trackpad"]["Clicking"].Bool {
		t.Fatalf("trackpad.Clicking missing: %#v", merged.Sections["trackpad"])
	}
}

func TestMergeMacOSDefaults_ForceReplacesSectionsPreservesCustom(t *testing.T) {
	existing := config.MacOSDefaults{
		Sections: map[string]map[string]config.SettingValue{
			"dock": {
				"autohide": {Kind: config.SettingBool, Bool: true},
				"orientation": {Kind: config.SettingString, String: "left"},
			},
			"finder": {
				"ShowPathbar": {Kind: config.SettingBool, Bool: true},
			},
		},
		Custom: map[string]map[string]config.SettingValue{
			"com.apple.Safari": {
				"ShowFullURLInSmartSearchField": {Kind: config.SettingBool, Bool: true},
			},
		},
	}
	discovered := config.MacOSDefaults{
		Sections: map[string]map[string]config.SettingValue{
			"dock": {
				"autohide": {Kind: config.SettingBool, Bool: false},
			},
			"trackpad": {
				"Clicking": {Kind: config.SettingBool, Bool: true},
			},
		},
	}

	merged, _ := MergeMacOSDefaults(existing, discovered, true)
	if _, ok := merged.Sections["finder"]; ok {
		t.Fatalf("force should drop finder section: %#v", merged.Sections)
	}
	if _, ok := merged.Sections["dock"]["orientation"]; ok {
		t.Fatalf("force should drop dock.orientation: %#v", merged.Sections["dock"])
	}
	if merged.Sections["dock"]["autohide"].Bool {
		t.Fatalf("force should use snapshot autohide=false: %#v", merged.Sections["dock"]["autohide"])
	}
	if !merged.Sections["trackpad"]["Clicking"].Bool {
		t.Fatalf("trackpad missing: %#v", merged.Sections)
	}
	if !merged.Custom["com.apple.Safari"]["ShowFullURLInSmartSearchField"].Bool {
		t.Fatalf("custom should be preserved: %#v", merged.Custom)
	}
}
