package configgit

import (
	"fmt"
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

func TestHasStagedChangesAndNothingToCommit(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pourover.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatal(err)
	}
	staged, err := HasStagedChanges(dir)
	if err != nil || !staged {
		t.Fatalf("staged=%v err=%v, want true", staged, err)
	}
	if err := Commit(dir, "initial"); err != nil {
		t.Fatal(err)
	}
	staged, err = HasStagedChanges(dir)
	if err != nil || staged {
		t.Fatalf("staged=%v err=%v, want false", staged, err)
	}
	// Empty commit should soft-succeed via isNothingToCommit path after add of nothing new.
	if err := Commit(dir, "empty"); err != nil {
		t.Fatalf("Commit with nothing staged: %v", err)
	}
}

func TestIsNonFastForward(t *testing.T) {
	err := fmt.Errorf("git push: ! [rejected] main -> main (fetch first)\nerror: failed to push")
	if !isNonFastForward(err) {
		t.Fatal("expected non-fast-forward detection")
	}
	if isNonFastForward(fmt.Errorf("git push: permission denied")) {
		t.Fatal("permission denied should not match")
	}
}

func TestIsNothingToCommit(t *testing.T) {
	err := fmt.Errorf("git commit: On branch main\nnothing to commit, working tree clean")
	if !isNothingToCommit(err) {
		t.Fatal("expected nothing-to-commit detection")
	}
}

func TestCommitAndPushIfDirty_SkipsCommitWhenNothingStaged(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pourover.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Init(nested); err != nil {
		t.Fatal(err)
	}
	if err := run(nested, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := run(nested, "config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "x"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(nested); err != nil {
		t.Fatal(err)
	}
	if err := Commit(nested, "nested"); err != nil {
		t.Fatal(err)
	}
	if err := AddAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := Commit(dir, "initial"); err != nil {
		t.Fatal(err)
	}
	// Dirty nested worktree — parent porcelain shows modified submodule content.
	if err := os.WriteFile(filepath.Join(nested, "x"), []byte("2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err := StatusDirty(dir)
	if err != nil || !dirty {
		t.Fatalf("dirty=%v err=%v, want true", dirty, err)
	}
	pushed, err := CommitAndPushIfDirty(dir, "main", time.Now())
	if err != nil {
		t.Fatalf("CommitAndPushIfDirty: %v", err)
	}
	if pushed {
		t.Fatal("expected no push without remote / not ahead")
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
