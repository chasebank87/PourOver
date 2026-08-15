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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %s", gitErrMsg(err, stdout.String(), stderr.String()))
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

// HasStagedChanges reports whether the index has staged diffs vs HEAD.
func HasStagedChanges(dir string) (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	err := cmd.Run()
	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("git diff --cached: %w", err)
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

// AddAll stages all changes. If a nested git directory without commits blocks
// `git add -A`, falls back to updating tracked paths only (`git add -u`).
func AddAll(dir string) error {
	err := run(dir, "add", "-A")
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "does not have a commit checked out") ||
		strings.Contains(msg, "unable to index file") {
		if err2 := run(dir, "add", "-u"); err2 == nil {
			return nil
		}
	}
	return err
}

// Commit creates a commit with message. A clean "nothing to commit" is treated as success.
func Commit(dir, message string) error {
	err := run(dir, "commit", "-m", message)
	if err != nil && isNothingToCommit(err) {
		return nil
	}
	return err
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

// PullRebase pulls from origin with rebase (for diverged branches before push).
func PullRebase(dir, branch string) error {
	args := []string{"pull", "--rebase", remoteName}
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

// PushReconcile pushes, and on non-fast-forward rejection pulls --rebase then pushes again.
func PushReconcile(dir, branch string) error {
	_ = run(dir, "fetch", remoteName)
	err := Push(dir, branch)
	if err == nil {
		return nil
	}
	if !isNonFastForward(err) {
		return err
	}
	if err := PullRebase(dir, branch); err != nil {
		return fmt.Errorf("push rejected (remote has new commits); rebase failed: %w\nfix: run `pourover config pull`, resolve conflicts, then `pourover config push`", err)
	}
	if err := Push(dir, branch); err != nil {
		return fmt.Errorf("push after rebase: %w", err)
	}
	return nil
}

// IsAheadOfRemote reports whether HEAD has commits not on origin/branch.
func IsAheadOfRemote(dir, branch string) (bool, error) {
	if branch == "" {
		branch = "main"
	}
	ref := remoteName + "/" + branch
	out, err := output(dir, "rev-list", "--count", ref+"..HEAD")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "0", nil
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

// CommitAndPushIfDirty stages+commits when there are stagedable changes, then
// pushes (rebasing onto remote when needed). Nested dirty git worktrees that
// do not change the parent index no longer block push.
func CommitAndPushIfDirty(dir, branch string, at time.Time) (pushed bool, err error) {
	dirty, err := StatusDirty(dir)
	if err != nil {
		return false, err
	}
	committed := false
	if dirty {
		if err := AddAll(dir); err != nil {
			return false, err
		}
		staged, err := HasStagedChanges(dir)
		if err != nil {
			return false, err
		}
		if staged {
			if err := Commit(dir, SyncCommitMessage(at)); err != nil {
				return false, err
			}
			committed = true
		}
	}

	ahead := false
	if a, aerr := IsAheadOfRemote(dir, branch); aerr == nil {
		ahead = a
	} else if committed {
		// No usable upstream yet; still attempt push after a new commit.
		ahead = true
	}

	if !committed && !ahead {
		return false, nil
	}
	if err := PushReconcile(dir, branch); err != nil {
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
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %s", args[0], gitErrMsg(err, stdout.String(), stderr.String()))
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
		return "", fmt.Errorf("git %s: %s", args[0], gitErrMsg(err, stdout.String(), stderr.String()))
	}
	return stdout.String(), nil
}

func gitErrMsg(err error, stdout, stderr string) string {
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = strings.TrimSpace(stdout)
	}
	if msg == "" && err != nil {
		msg = err.Error()
	}
	return msg
}

func isNothingToCommit(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nothing to commit") ||
		strings.Contains(msg, "no changes added to commit")
}

func isNonFastForward(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "fetch first") ||
		strings.Contains(msg, "non-fast-forward") ||
		strings.Contains(msg, "updates were rejected")
}
