package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchICloud_ToggleExisting(t *testing.T) {
	src := `-- Root
local packages = require("packages")

return {
  packages = packages,
  policy = {
    uninstall_mode = "safe",
  },
  backup = {
    icloud = {
      enabled = false,
    },
  },
  -- keep me
  macos = {
    defaults = {
      dock = { autohide = true },
    },
  },
}
`
	out, err := PatchICloud(src, true, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "enabled = true") {
		t.Fatalf("expected enabled=true, got:\n%s", out)
	}
	if !strings.Contains(out, "-- keep me") || !strings.Contains(out, "autohide = true") {
		t.Fatalf("lost unrelated content:\n%s", out)
	}
	out, err = PatchICloud(out, false, "/tmp/icloud", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "enabled = false") {
		t.Fatalf("expected enabled=false:\n%s", out)
	}
	if !strings.Contains(out, `path = "/tmp/icloud"`) {
		t.Fatalf("expected path:\n%s", out)
	}
}

func TestPatchICloud_InsertMissingBackup(t *testing.T) {
	src := `return {
  policy = { uninstall_mode = "safe" },
}
`
	out, err := PatchICloud(src, true, "", false)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load: %v\n%s", err, out)
	}
	if !m.Backup.ICloud.Enabled {
		t.Fatalf("icloud not enabled: %#v", m.Backup)
	}
}

func TestPatchGit_InsertAndRoundTrip(t *testing.T) {
	src := `return {
  backup = {
    icloud = {
      enabled = false,
    },
  },
}
`
	out, err := PatchGit(src, GitBackup{
		Enabled:  true,
		Remote:   "git@github.com:me/pourover-config.git",
		AutoPush: true,
		Branch:   "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load: %v\n%s", err, out)
	}
	if !m.Backup.Git.Enabled || m.Backup.Git.Remote == "" || !m.Backup.Git.AutoPush {
		t.Fatalf("git: %#v", m.Backup.Git)
	}
	if m.Backup.ICloud.Enabled {
		t.Fatal("icloud should stay disabled")
	}
}

func TestPatchICloudFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = { icloud = { enabled = false } },
}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := PatchICloudFile(path, true, "", false); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Backup.ICloud.Enabled {
		t.Fatal("expected enabled")
	}
}
