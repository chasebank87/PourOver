package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/state"
)

func TestImport_FileTargetsSubset(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.MkdirAll(filepath.Join(cfgDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte("return { formulae = {}, casks = {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := `local packages = require("packages")
local macos = require("macos")
return {
  packages = packages,
  macos = macos,
  files = {
    links = {},
    managed = { { source = "config/foo.conf", target = "~/.config/foo.conf" } },
  },
  policy = { uninstall_mode = "safe", files_mode = "safe" },
}
`
	if err := os.WriteFile(configPath, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "macos.lua"), []byte("return { defaults = {} }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export HI=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "nvim", "init.lua"), []byte("-- nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "cursor", "argv.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Import(context.Background(), nil, ImportOptions{
		ConfigDir:   cfgDir,
		ConfigPath:  configPath,
		Packages:    false,
		Files:       true,
		Home:        home,
		FileTargets: []string{"~/.zshrc", "~/.config/nvim"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AddedLinks) != 2 {
		t.Fatalf("AddedLinks = %#v", result.AddedLinks)
	}
	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "~/.zshrc") || !strings.Contains(got, "~/.config/nvim") {
		t.Fatalf("missing selected links:\n%s", got)
	}
	if strings.Contains(got, "cursor") {
		t.Fatalf("cursor must not be imported:\n%s", got)
	}
	if !strings.Contains(got, "macos = macos") || !strings.Contains(got, "managed") {
		t.Fatalf("surgical patch lost other sections:\n%s", got)
	}
}

func TestUnmanageFiles_RemovesLinkAndOwned(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configPath := filepath.Join(cfgDir, "pourover.lua")
	stateDir := filepath.Join(home, "state")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "packages.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := `local packages = require("packages")
return {
  packages = packages,
  files = {
    links = {
      { source = "config/cursor", target = "~/.config/cursor" },
      { source = "config/nvim", target = "~/.config/nvim" },
    },
  },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(configPath, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(home, ".config", "cursor", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, []byte("alive\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	ownedCursor := filepath.Join(home, ".config", "cursor", "keep.txt")
	ownedNvim := filepath.Join(home, ".config", "nvim", "init.lua")
	if err := state.WriteJSONAtomic(filepath.Join(stateDir, "lock.json"), state.Lock{
		OwnedFiles: []string{ownedCursor, ownedNvim},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := UnmanageFiles(UnmanageFilesOptions{
		ConfigPath: configPath,
		StateDir:   stateDir,
		Targets:    []string{"~/.config/cursor"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RemovedLinks) != 1 || result.RemovedLinks[0].Target != "~/.config/cursor" {
		t.Fatalf("RemovedLinks = %#v", result.RemovedLinks)
	}
	man, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(man.Files.Links) != 1 || man.Files.Links[0].Target != "~/.config/nvim" {
		t.Fatalf("links = %#v", man.Files.Links)
	}
	if _, err := os.Stat(live); err != nil {
		t.Fatalf("live file should remain: %v", err)
	}
	lock, err := state.LoadLock(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range lock.OwnedFiles {
		if strings.Contains(p, "cursor") {
			t.Fatalf("owned still has cursor path: %#v", lock.OwnedFiles)
		}
	}
	if len(lock.OwnedFiles) != 1 || lock.OwnedFiles[0] != ownedNvim {
		t.Fatalf("OwnedFiles = %#v", lock.OwnedFiles)
	}
}
