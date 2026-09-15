package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAutoUpdateManifest(t *testing.T, packagesBody string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
` + packagesBody + `
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadManifest_AutoUpdateBoolTrue(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    casks = {},
    auto_update = true,
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	au := m.Packages.AutoUpdate
	if !au.Configured {
		t.Fatal("Configured = false, want true")
	}
	if !au.Enable {
		t.Fatal("Enable = false, want true")
	}
	if au.Interval != "" || au.Upgrade || au.Greedy || au.Cleanup || au.Immediate {
		t.Fatalf("unexpected extras: %#v", au)
	}
}

func TestLoadManifest_AutoUpdateBoolFalse(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = false,
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	au := m.Packages.AutoUpdate
	if !au.Configured {
		t.Fatal("Configured = false, want true when key present")
	}
	if au.Enable {
		t.Fatal("Enable = true, want false")
	}
}

func TestLoadManifest_AutoUpdateOmittedUnconfigured(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = { "git" },
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Packages.AutoUpdate.Configured {
		t.Fatalf("Configured = true, want false: %#v", m.Packages.AutoUpdate)
	}
}

func TestLoadManifest_AutoUpdateTable(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = {
      enable = true,
      interval = "12h",
      upgrade = true,
      greedy = true,
      cleanup = true,
      immediate = true,
    },
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	au := m.Packages.AutoUpdate
	if !au.Configured || !au.Enable || au.Interval != "12h" {
		t.Fatalf("got %#v", au)
	}
	if !au.Upgrade || !au.Greedy || !au.Cleanup || !au.Immediate {
		t.Fatalf("flags: %#v", au)
	}
}

func TestLoadManifest_AutoUpdateEmptyTableDefaultsEnable(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = {},
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	au := m.Packages.AutoUpdate
	if !au.Configured || !au.Enable {
		t.Fatalf("empty table should enable: %#v", au)
	}
}

func TestLoadManifest_AutoUpdateIntervalNumber(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = { interval = 86400 },
`)
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Packages.AutoUpdate.Interval != "86400" {
		t.Fatalf("Interval = %q, want 86400", m.Packages.AutoUpdate.Interval)
	}
}

func TestLoadManifest_AutoUpdateGreedyRequiresUpgrade(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = { greedy = true },
`)
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "packages.auto_update.greedy") {
		t.Fatalf("LoadManifest = %v, want greedy requires upgrade", err)
	}
}

func TestLoadManifest_AutoUpdateInvalidInterval(t *testing.T) {
	path := writeAutoUpdateManifest(t, `
    formulae = {},
    auto_update = { interval = "tomorrow" },
`)
	_, err := LoadManifest(path)
	if err == nil || !strings.Contains(err.Error(), "packages.auto_update.interval") {
		t.Fatalf("LoadManifest = %v, want interval error", err)
	}
}

func TestParseAutoUpdateInterval(t *testing.T) {
	t.Parallel()
	empty, err := ParseAutoUpdateInterval("")
	if err != nil || empty.Seconds != DefaultAutoUpdateSeconds || empty.Calendar {
		t.Fatalf("empty = %#v, %v", empty, err)
	}
	hour, err := ParseAutoUpdateInterval("12h")
	if err != nil || hour.Seconds != 12*3600 {
		t.Fatalf("12h = %#v, %v", hour, err)
	}
	clock, err := ParseAutoUpdateInterval("00:00")
	if err != nil || !clock.Calendar || clock.Hour != 0 || clock.Minute != 0 {
		t.Fatalf("00:00 = %#v, %v", clock, err)
	}
	if _, err := ParseAutoUpdateInterval("24:00"); err == nil {
		t.Fatal("24:00 should fail")
	}
}

func TestAutoUpdateStartArgs(t *testing.T) {
	t.Parallel()
	got := AutoUpdate{Enable: true, Interval: "12h", Upgrade: true, Cleanup: true}.StartArgs()
	want := []string{"autoupdate", "start", "12h", "--upgrade", "--cleanup"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("StartArgs = %v, want %v", got, want)
	}
	defaults := AutoUpdate{Enable: true}.StartArgs()
	if strings.Join(defaults, " ") != "autoupdate start" {
		t.Fatalf("default StartArgs = %v", defaults)
	}
}
