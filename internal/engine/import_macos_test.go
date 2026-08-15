package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// mapDefaults stubs discovery.DefaultsRunner like discovery tests.
type mapDefaults struct{ vals map[string]string } // key "domain|key" → raw

func (m *mapDefaults) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	k := args[1] + "|" + args[2]
	if v, ok := m.vals[k]; ok {
		return []byte(v), nil
	}
	return nil, &discovery.DefaultsExitError{Args: args, Stderr: "The domain/default pair does not exist"}
}

const loginwindowDomain = "/Library/Preferences/com.apple.loginwindow"

func scaffoldRoot(t *testing.T, dir string) string {
	t.Helper()
	root := `-- Root PourOver config.
local packages = require("packages")

return {
  packages = packages,
  files = {
    links = {},
  },
  -- macos = { defaults = { dock = { autohide = true } } },
  policy = {
    uninstall_mode = "safe",
  },
  backup = {
    icloud = { enabled = false },
    git = { enabled = false, auto_push = true, branch = "main" },
  },
}
`
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	pkgs := `return { taps = {}, formulae = {}, casks = {} }`
	if err := os.WriteFile(filepath.Join(dir, "packages.lua"), []byte(pkgs), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportMacOS_DryRun_NoWrite(t *testing.T) {
	dir := t.TempDir()
	configPath := scaffoldRoot(t, dir)
	runner := &mapDefaults{vals: map[string]string{
		loginwindowDomain + "|LoginwindowText": "Found? Call 513-968-9283",
	}}

	res, err := ImportMacOS(context.Background(), ImportMacOSOptions{
		ConfigDir:  dir,
		ConfigPath: configPath,
		DryRun:     true,
		Runner:     runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatal("DryRun flag")
	}
	if !strings.Contains(res.Lua, "LoginwindowText") {
		t.Fatalf("Lua missing LoginwindowText:\n%s", res.Lua)
	}
	if !strings.Contains(res.Lua, "Found? Call 513-968-9283") {
		t.Fatalf("Lua missing value:\n%s", res.Lua)
	}
	if res.ReadCount < 1 {
		t.Fatalf("ReadCount = %d", res.ReadCount)
	}
	macosPath := filepath.Join(dir, "macos.lua")
	if _, err := os.Stat(macosPath); !os.IsNotExist(err) {
		t.Fatalf("macos.lua should be absent on dry-run; err=%v", err)
	}
	root, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(root), `require("macos")`) {
		t.Fatal("dry-run should not patch pourover.lua")
	}
	if res.EnsuredRequire {
		t.Fatal("EnsuredRequire should be false on dry-run")
	}
	if !res.HasSystemScope {
		t.Fatal("loginwindow is system scope")
	}
	if res.AdminNote == "" {
		t.Fatal("expected AdminNote for system-scope keys")
	}
}

func TestImportMacOS_MergeWrite_CreatesMacOSAndRequire(t *testing.T) {
	dir := t.TempDir()
	configPath := scaffoldRoot(t, dir)
	runner := &mapDefaults{vals: map[string]string{
		loginwindowDomain + "|LoginwindowText": "Hello from import",
		config.DomainDock + "|autohide":        "1",
	}}

	res, err := ImportMacOS(context.Background(), ImportMacOSOptions{
		ConfigDir:  dir,
		ConfigPath: configPath,
		Runner:     runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.DryRun {
		t.Fatal("not dry-run")
	}
	if res.MacOSPath != filepath.Join(dir, "macos.lua") {
		t.Fatalf("MacOSPath = %q", res.MacOSPath)
	}
	if _, err := os.Stat(res.MacOSPath); err != nil {
		t.Fatal(err)
	}
	if !res.EnsuredRequire {
		t.Fatal("expected EnsuredRequire")
	}
	root, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(root)
	if !strings.Contains(body, `local macos = require("macos")`) {
		t.Fatalf("missing macos require:\n%s", body)
	}
	if !strings.Contains(body, "macos = macos") {
		t.Fatalf("missing macos = macos in return:\n%s", body)
	}

	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	got := m.MacOS.Defaults.Sections["loginwindow"]["LoginwindowText"]
	if got.Kind != config.SettingString || got.String != "Hello from import" {
		t.Fatalf("LoginwindowText = %#v", got)
	}
	if !m.MacOS.Defaults.Sections["dock"]["autohide"].Bool {
		t.Fatalf("dock.autohide = %#v", m.MacOS.Defaults.Sections["dock"]["autohide"])
	}
	if res.Added < 1 {
		t.Fatalf("Added = %d", res.Added)
	}
}

func TestImportMacOS_Force_ReplacesCuratedSections(t *testing.T) {
	dir := t.TempDir()
	configPath := scaffoldRoot(t, dir)

	existingMacOS := `-- generated
return {
  defaults = {
    dock = {
      autohide = false,
      tilesize = 64,
    },
    custom = {
      ["com.apple.Safari"] = {
        ShowFullURLInSmartSearchField = true,
      },
    },
  },
}
`
	if err := os.WriteFile(filepath.Join(dir, "macos.lua"), []byte(existingMacOS), 0o644); err != nil {
		t.Fatal(err)
	}
	// Point root at macos module already (pre-required).
	root := `local packages = require("packages")
local macos = require("macos")
return {
  packages = packages,
  macos = macos,
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(configPath, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &mapDefaults{vals: map[string]string{
		loginwindowDomain + "|LoginwindowText": "forced text",
		config.DomainDock + "|autohide":        "1",
	}}

	res, err := ImportMacOS(context.Background(), ImportMacOSOptions{
		ConfigDir:  dir,
		ConfigPath: configPath,
		Force:      true,
		Runner:     runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Force {
		t.Fatal("Force flag")
	}
	if res.EnsuredRequire {
		t.Fatal("require already present; should not re-ensure")
	}

	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	dock := m.MacOS.Defaults.Sections["dock"]
	if dock == nil || !dock["autohide"].Bool {
		t.Fatalf("dock after force = %#v", dock)
	}
	if _, ok := dock["tilesize"]; ok {
		t.Fatalf("force should drop tilesize not in snapshot; dock=%#v", dock)
	}
	lw := m.MacOS.Defaults.Sections["loginwindow"]["LoginwindowText"]
	if lw.String != "forced text" {
		t.Fatalf("LoginwindowText = %#v", lw)
	}
	if !m.MacOS.Defaults.Custom["com.apple.Safari"]["ShowFullURLInSmartSearchField"].Bool {
		t.Fatalf("custom must be preserved: %#v", m.MacOS.Defaults.Custom)
	}
}

func TestImportMacOS_WriteWithoutConfig_Errors(t *testing.T) {
	dir := t.TempDir()
	runner := &mapDefaults{vals: map[string]string{
		loginwindowDomain + "|LoginwindowText": "x",
	}}
	_, err := ImportMacOS(context.Background(), ImportMacOSOptions{
		ConfigDir:  dir,
		ConfigPath: filepath.Join(dir, "pourover.lua"),
		Runner:     runner,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pourover init") {
		t.Fatalf("error = %v", err)
	}
}

func TestImportMacOS_DryRunWithoutConfig_OK(t *testing.T) {
	dir := t.TempDir()
	runner := &mapDefaults{vals: map[string]string{
		loginwindowDomain + "|LoginwindowText": "dry",
	}}
	res, err := ImportMacOS(context.Background(), ImportMacOSOptions{
		ConfigDir:  dir,
		ConfigPath: filepath.Join(dir, "pourover.lua"),
		DryRun:     true,
		Runner:     runner,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Lua, "LoginwindowText") {
		t.Fatalf("Lua:\n%s", res.Lua)
	}
}

func TestEnsureMacOSRequire_AddsLocalAndTable(t *testing.T) {
	in := `local packages = require("packages")

return {
  packages = packages,
  files = { links = {} },
}
`
	out, changed := ensureMacOSRequire(in)
	if !changed {
		t.Fatal("expected change")
	}
	if !strings.Contains(out, `local macos = require("macos")`) {
		t.Fatalf("missing local require:\n%s", out)
	}
	if !strings.Contains(out, "macos = macos") {
		t.Fatalf("missing table field:\n%s", out)
	}
	_, changed2 := ensureMacOSRequire(out)
	if changed2 {
		t.Fatal("second pass should be no-op")
	}
}

func TestEnsureMacOSRequire_InlineRequireIntact(t *testing.T) {
	in := `return {
  macos = require("macos"),
  packages = { formulae = {} },
}
`
	out, changed := ensureMacOSRequire(in)
	if changed {
		t.Fatal("already requires macos")
	}
	if out != in {
		t.Fatalf("mutated:\n%s", out)
	}
}
