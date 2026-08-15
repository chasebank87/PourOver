package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest_SudoLocalPAMFlags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = { formulae = {}, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
  macos = {
    security = {
      pam = {
        sudo_local = {
          enable = true,
          reattach = true,
          touch_id_auth = true,
          watch_id_auth = true,
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
		t.Fatalf("LoadManifest: %v", err)
	}
	sl := m.MacOS.Security.PAM.SudoLocal
	if !sl.Configured {
		t.Fatal("Configured = false, want true when sudo_local table present")
	}
	if !sl.Enable {
		t.Error("Enable = false, want true")
	}
	if !sl.Reattach {
		t.Error("Reattach = false, want true")
	}
	if !sl.TouchIDAuth {
		t.Error("TouchIDAuth = false, want true")
	}
	if !sl.WatchIDAuth {
		t.Error("WatchIDAuth = false, want true")
	}
}

func TestLoadManifest_OmittedSecurityPAMUnconfigured(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = { formulae = {}, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
  macos = {
    defaults = {
      dock = { autohide = true },
    },
  },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.MacOS.Security.PAM.SudoLocal.Configured {
		t.Fatalf("Configured = true, want false when security omitted; got %#v", m.MacOS.Security.PAM.SudoLocal)
	}
	if !m.MacOS.Defaults.Sections["dock"]["autohide"].Bool {
		t.Fatal("defaults still load when security omitted")
	}
}
