package generation_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/generation"
)

func TestBuild_StoresLinkBlobAndManifest(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	src := filepath.Join(configDir, "config", "gitconfig")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("content-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "gitconfig")

	m := config.Manifest{
		Files: config.Files{
			Links: []config.FileLink{{Source: "config/gitconfig", Target: target}},
		},
	}
	at := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	res, err := generation.Build(stateDir, configDir, m, at)
	if err != nil {
		t.Fatal(err)
	}
	if res.Manifest.ID == "" {
		t.Fatal("empty generation id")
	}
	if len(res.Manifest.Files) != 1 {
		t.Fatalf("files = %d, want 1", len(res.Manifest.Files))
	}
	e := res.Manifest.Files[0]
	if e.Kind != generation.FileKindLink || e.Target != target {
		t.Fatalf("entry = %+v", e)
	}
	data, err := generation.ReadBlob(stateDir, res.Manifest.ID, e.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content-a\n" {
		t.Fatalf("blob = %q", data)
	}
	loaded, err := generation.Load(stateDir, res.Manifest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != res.Manifest.ID {
		t.Fatalf("load id = %s", loaded.ID)
	}
}

func TestBuild_ExpandsDirectoryLink(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	srcDir := filepath.Join(configDir, "config", "nvim")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "init.lua"), []byte("-- init\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "lua"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "lua", "x.lua"), []byte("-- x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "nvim")

	m := config.Manifest{
		Files: config.Files{
			Links: []config.FileLink{{Source: "config/nvim", Target: targetRoot}},
		},
	}
	res, err := generation.Build(stateDir, configDir, m, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Manifest.Files) != 2 {
		t.Fatalf("files = %d, want 2: %+v", len(res.Manifest.Files), res.Manifest.Files)
	}
}

func TestCurrentAndPrune(t *testing.T) {
	stateDir := t.TempDir()
	configDir := t.TempDir()
	src := filepath.Join(configDir, "a")
	if err := os.WriteFile(src, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "a")
	m := config.Manifest{Files: config.Files{Links: []config.FileLink{{Source: "a", Target: target}}}}

	var ids []string
	for i := 0; i < 7; i++ {
		at := time.Date(2026, 8, 15, 12, i, 0, 0, time.UTC)
		// change content so hash/id differs when needed; timestamps already uniquify
		_ = os.WriteFile(src, []byte{byte('a' + i)}, 0o644)
		res, err := generation.Build(stateDir, configDir, m, at)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, res.Manifest.ID)
	}
	if err := generation.SetCurrent(stateDir, ids[len(ids)-1]); err != nil {
		t.Fatal(err)
	}
	cur, err := generation.Current(stateDir)
	if err != nil || cur != ids[len(ids)-1] {
		t.Fatalf("current = %q, err=%v", cur, err)
	}
	if err := generation.Prune(stateDir, 3); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(generation.GenerationsDir(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("after prune got %d gens, want 3", len(entries))
	}
}
