package configgit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const remoteName = "origin"

// DefaultGitignore is written into the config repo when missing.
const DefaultGitignore = `.DS_Store
`

// IsRepo reports whether dir is a git working tree.
func IsRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// Init runs git init in dir when not already a repo.
func Init(dir string) error {
	if IsRepo(dir) {
		return nil
	}
	return run(dir, "init")
}

// Clone clones url into dest (dest must not exist or be empty).
func Clone(url, dest, branch string) error {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	args = append(args, url, dest)
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git clone: %s", msg)
	}
	return nil
}

// EnsureRemote sets origin to url (add or set-url).
func EnsureRemote(dir, url string) error {
	if err := run(dir, "remote", "get-url", remoteName); err != nil {
		return run(dir, "remote", "add", remoteName, url)
	}
	return run(dir, "remote", "set-url", remoteName, url)
}

// RemoteURL returns origin URL when set.
func RemoteURL(dir string) (string, error) {
	out, err := output(dir, "remote", "get-url", remoteName)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// StatusDirty reports whether the working tree has uncommitted changes.
func StatusDirty(dir string) (bool, error) {
	out, err := output(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// EnsureGitignore writes DefaultGitignore when .gitignore is missing.
func EnsureGitignore(dir string) error {
	path := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.WriteFile(path, []byte(DefaultGitignore), 0o644)
}

// AddAll stages all changes.
func AddAll(dir string) error {
	return run(dir, "add", "-A")
}

// Commit creates a commit with message. No-op error if nothing to commit is not returned
// when tree is clean — callers should check StatusDirty first.
func Commit(dir, message string) error {
	return run(dir, "commit", "-m", message)
}

// SyncCommitMessage returns the default pourover sync commit message.
func SyncCommitMessage(at time.Time) string {
	return fmt.Sprintf("pourover: sync config %s", at.UTC().Format("2006-01-02T15:04:05Z"))
}

// Pull pulls from origin (ff-only).
func Pull(dir, branch string) error {
	args := []string{"pull", "--ff-only", remoteName}
	if branch != "" {
		args = append(args, branch)
	}
	return run(dir, args...)
}

// Push pushes to origin.
func Push(dir, branch string) error {
	args := []string{"push", "-u", remoteName}
	if branch != "" {
		args = append(args, branch)
	} else {
		args = append(args, "HEAD")
	}
	return run(dir, args...)
}

// CurrentBranch returns the current branch name.
func CurrentBranch(dir string) (string, error) {
	out, err := output(dir, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// EnsureBranch renames the current branch to name when different (no-op if already set).
func EnsureBranch(dir, name string) error {
	if name == "" {
		return nil
	}
	cur, err := CurrentBranch(dir)
	if err != nil {
		// empty repo with no commits yet — set symbolic-ref
		return run(dir, "symbolic-ref", "HEAD", "refs/heads/"+name)
	}
	if cur == name {
		return nil
	}
	if cur == "" {
		return run(dir, "symbolic-ref", "HEAD", "refs/heads/"+name)
	}
	return run(dir, "branch", "-M", name)
}

// CommitAndPushIfDirty stages, commits, and pushes when the tree is dirty.
func CommitAndPushIfDirty(dir, branch string, at time.Time) (pushed bool, err error) {
	dirty, err := StatusDirty(dir)
	if err != nil {
		return false, err
	}
	if !dirty {
		return false, nil
	}
	if err := AddAll(dir); err != nil {
		return false, err
	}
	if err := Commit(dir, SyncCommitMessage(at)); err != nil {
		return false, err
	}
	if err := Push(dir, branch); err != nil {
		return false, err
	}
	return true, nil
}

// DirEmpty reports whether dir does not exist or has no entries.
func DirEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(entries) == 0, nil
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("git %s: %s", args[0], msg)
	}
	return nil
}

func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}
	return stdout.String(), nil
}
