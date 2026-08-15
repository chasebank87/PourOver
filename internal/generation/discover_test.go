package generation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/generation"
)

func TestDiscoverFiles_DirectorySymlinkAncestorIsDiffer(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	srcDir := filepath.Join(configDir, "nvim")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "init.lua"), []byte("-- init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	liveRoot := filepath.Join(root, "home", ".config")
	if err := os.MkdirAll(liveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	liveNvim := filepath.Join(liveRoot, "nvim")
	if err := os.Symlink(srcDir, liveNvim); err != nil {
		t.Fatal(err)
	}
	// Nested file is a regular file through the symlink; content matches source.
	liveFile := filepath.Join(liveNvim, "init.lua")
	hash, err := generation.HashFile(filepath.Join(srcDir, "init.lua"))
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := generation.DiscoverFiles([]generation.FileEntry{{
		Target: liveFile,
		Hash:   hash,
		Kind:   generation.FileKindLink,
		Source: "nvim/init.lua",
		Mode:   "0644",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Kind != generation.FileStatusDiffer {
		t.Fatalf("status = %+v, want Differ (symlink ancestor)", statuses)
	}
}
