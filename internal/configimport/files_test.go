package configimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestImportFile_RegularFile(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	target := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(target, []byte("export HI=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: target,
		TargetDecl: "~/.zshrc",
		RelSource:  "config/home/zshrc",
	}
	link, err := ImportFile(cfgDir, c, true)
	if err != nil {
		t.Fatal(err)
	}
	if link.Source != "config/home/zshrc" || link.Target != "~/.zshrc" {
		t.Fatalf("link = %+v", link)
	}
	src := filepath.Join(cfgDir, "config", "home", "zshrc")
	data, err := os.ReadFile(src)
	if err != nil || string(data) != "export HI=1\n" {
		t.Fatalf("copied = %q err=%v", data, err)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != src {
		t.Fatalf("symlink = %q, want %q", got, src)
	}
}

func TestImportFile_ExistingSymlink(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	oldSrc := filepath.Join(home, "old-nvim")
	if err := os.MkdirAll(oldSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldSrc, "init.lua"), []byte("-- nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configRoot := filepath.Join(home, ".config")
	if err := os.MkdirAll(configRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(configRoot, "nvim")
	if err := os.Symlink(oldSrc, target); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: target,
		TargetDecl: "~/.config/nvim",
		RelSource:  "config/nvim",
	}
	if _, err := ImportFile(cfgDir, c, true); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(cfgDir, "config", "nvim", "init.lua")
	if _, err := os.Stat(copied); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cfgDir, "config", "nvim")
	if got != want {
		t.Fatalf("symlink = %q, want %q", got, want)
	}
}

func TestImportFile_DirectoryWithNestedDir(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configRoot := filepath.Join(home, ".config")
	nvim := filepath.Join(configRoot, "nvim")
	if err := os.MkdirAll(filepath.Join(nvim, "lua", "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nvim, "init.lua"), []byte("-- root\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nvim, "lua", "plugins", "init.lua"), []byte("-- plug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: nvim,
		TargetDecl: "~/.config/nvim",
		RelSource:  "config/nvim",
	}
	if _, err := ImportFile(cfgDir, c, true); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"config/nvim/init.lua",
		"config/nvim/lua/plugins/init.lua",
	} {
		if _, err := os.Stat(filepath.Join(cfgDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestImportFile_SymlinkToDirectoryInsideTree(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configRoot := filepath.Join(home, ".config")
	nvim := filepath.Join(configRoot, "nvim")
	realLua := filepath.Join(home, "real-lua")
	if err := os.MkdirAll(filepath.Join(realLua, "plugins"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realLua, "plugins", "x.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nvim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nvim, "init.lua"), []byte("-- nvim\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Symlink-to-directory: DirEntry.IsDir() is false; import must still recurse.
	if err := os.Symlink(realLua, filepath.Join(nvim, "lua")); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: nvim,
		TargetDecl: "~/.config/nvim",
		RelSource:  "config/nvim",
	}
	if _, err := ImportFile(cfgDir, c, true); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(cfgDir, "config", "nvim", "lua", "plugins", "x.lua")
	data, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "return {}\n" {
		t.Fatalf("copied = %q", data)
	}
}

func TestImportFile_ClearsLeftoverFileWhereDirectoryNeeded(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configRoot := filepath.Join(home, ".config")
	nvim := filepath.Join(configRoot, "nvim")
	if err := os.MkdirAll(filepath.Join(nvim, "lua"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nvim, "lua", "init.lua"), []byte("-- ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Simulate v0.1.3 failure residue: empty file at the path that should be a directory.
	leftover := filepath.Join(cfgDir, "config", "nvim", "lua")
	if err := os.MkdirAll(filepath.Dir(leftover), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leftover, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: nvim,
		TargetDecl: "~/.config/nvim",
		RelSource:  "config/nvim",
	}
	if _, err := ImportFile(cfgDir, c, false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(leftover)
	if err != nil || !info.IsDir() {
		t.Fatalf("lua path = %#v err=%v, want directory", info, err)
	}
	if _, err := os.Stat(filepath.Join(leftover, "init.lua")); err != nil {
		t.Fatal(err)
	}
}

func TestFormatRootLua(t *testing.T) {
	got := FormatRootLua(
		[]config.FileLink{{Source: "config/nvim", Target: "~/.config/nvim"}},
		config.Policy{UninstallMode: config.UninstallModeSafe},
		config.Backup{},
	)
	for _, frag := range []string{`require("packages")`, `source = "config/nvim"`, `uninstall_mode = "safe"`, `git = {`, `auto_push = false`} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in %s", frag, got)
		}
	}
}
