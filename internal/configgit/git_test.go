package configgit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitStatusCommit(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if !IsRepo(dir) {
		t.Fatal("expected repo")
	}
	if err := run(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureGitignore(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pourover.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err := StatusDirty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !dirty {
		t.Fatal("expected dirty")
	}
	if err := AddAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, SyncCommitMessage(time.Now())); err != nil {
		t.Fatal(err)
	}
	dirty, err = StatusDirty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirty {
		t.Fatal("expected clean after commit")
	}
}

func TestDirEmpty(t *testing.T) {
	dir := t.TempDir()
	empty, err := DirEmpty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("temp dir should be empty")
	}
	if err := os.WriteFile(filepath.Join(dir, "x"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = DirEmpty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected non-empty")
	}
	missing := filepath.Join(dir, "nope")
	empty, err = DirEmpty(missing)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("missing dir is empty")
	}
}

func TestEnsureRemote(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	url := "git@github.com:example/pourover-config.git"
	if err := EnsureRemote(dir, url); err != nil {
		t.Fatal(err)
	}
	got, err := RemoteURL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != url {
		t.Fatalf("remote=%q", got)
	}
	url2 := "https://github.com/example/pourover-config.git"
	if err := EnsureRemote(dir, url2); err != nil {
		t.Fatal(err)
	}
	got, err = RemoteURL(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != url2 {
		t.Fatalf("remote=%q", got)
	}
}
