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
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("live target must be a regular file, not a symlink")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "export HI=1\n" {
		t.Fatalf("live = %q err=%v", got, err)
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
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("live nvim must be a directory tree, not a symlink")
	}
	data, err := os.ReadFile(filepath.Join(target, "init.lua"))
	if err != nil || string(data) != "-- nvim\n" {
		t.Fatalf("live init.lua = %q err=%v", data, err)
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

func TestImportFile_ReimportWhenAlreadySymlinkedIntoConfig(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	src := filepath.Join(cfgDir, "config", "home", "zshrc")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export HI=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".zshrc")
	if err := os.Symlink(src, target); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: target,
		TargetDecl: "~/.zshrc",
		RelSource:  "config/home/zshrc",
	}
	if _, err := ImportFile(cfgDir, c, true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(src)
	if err != nil || string(data) != "export HI=1\n" {
		t.Fatalf("content = %q err=%v", data, err)
	}
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("reimport must replace legacy symlink with a regular file")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "export HI=1\n" {
		t.Fatalf("live = %q err=%v", got, err)
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

func TestDefaultHomeCandidates_SkipsJunkNames(t *testing.T) {
	home := t.TempDir()
	configRoot := filepath.Join(home, ".config")
	if err := os.MkdirAll(configRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nvim", ".DS_Store", "~", "ghostty"} {
		path := filepath.Join(configRoot, name)
		if name == "nvim" || name == "ghostty" {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cands := DefaultHomeCandidates(home)
	for _, c := range cands {
		base := filepath.Base(c.TargetPath)
		if base == ".DS_Store" || base == "~" {
			t.Fatalf("unexpected junk candidate: %+v", c)
		}
	}
	var sawNvim, sawGhostty bool
	for _, c := range cands {
		if c.TargetDecl == "~/.config/nvim" {
			sawNvim = true
		}
		if c.TargetDecl == "~/.config/ghostty" {
			sawGhostty = true
		}
	}
	if !sawNvim || !sawGhostty {
		t.Fatalf("expected real config apps, got %+v", cands)
	}
}

func TestImportFile_SkipsNestedJunk(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".pourover")
	configRoot := filepath.Join(home, ".config")
	app := filepath.Join(configRoot, "blackbird")
	if err := os.MkdirAll(filepath.Join(app, "epm-dashboard"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "readme.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "epm-dashboard", ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := FileCandidate{
		TargetPath: app,
		TargetDecl: "~/.config/blackbird",
		RelSource:  "config/blackbird",
	}
	if _, err := ImportFile(cfgDir, c, false); err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(cfgDir, "config", "blackbird")
	if _, err := os.Stat(filepath.Join(copied, "readme.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(copied, ".DS_Store")); !os.IsNotExist(err) {
		t.Fatalf("copied top-level .DS_Store, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(copied, "epm-dashboard", ".DS_Store")); !os.IsNotExist(err) {
		t.Fatalf("copied nested .DS_Store, err=%v", err)
	}
}
