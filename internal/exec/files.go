package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
)

// CreateLink creates a symlink at targetPath pointing to sourcePath.
// Parent directories of targetPath are created when missing.
func CreateLink(targetPath, sourcePath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create link %s: parent dir: %w", targetPath, err)
	}
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		return fmt.Errorf("create link %s -> %s: %w", targetPath, sourcePath, err)
	}
	return nil
}

// UpdateLink replaces an existing symlink at targetPath to point to sourcePath.
// Refuses to replace a non-symlink target.
func UpdateLink(targetPath, sourcePath string) error {
	info, err := os.Lstat(targetPath)
	if err != nil {
		return fmt.Errorf("update link %s: %w", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("update link %s: target exists and is not a symlink", targetPath)
	}
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("update link %s: remove old: %w", targetPath, err)
	}
	return CreateLink(targetPath, sourcePath)
}

// FileApplyOptions configures file link/copy apply with optional backup-on-replace.
type FileApplyOptions struct {
	ConfigDir   string
	StateDir    string
	FileReplace config.FileReplaceMode
	Now         func() time.Time
}

func (o FileApplyOptions) clock() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// ApplyFileLinks runs link_create, link_update, and link_replace actions.
// Source paths are resolved relative to configDir; target paths expand ~.
// link_replace backs up an existing target under StateDir/backups/files/ then creates the link.
// Per-link failures are collected so later links still run (same soft-fail model as brew).
func ApplyFileLinks(p plan.Plan, opts FileApplyOptions, progress Progress) (int, error) {
	configDir, err := filepath.Abs(opts.ConfigDir)
	if err != nil {
		return 0, fmt.Errorf("config directory: %w", err)
	}

	n := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionLinkCreate, plan.ActionLinkUpdate, plan.ActionLinkReplace:
		default:
			continue
		}

		report(progress, a)

		sourcePath, err := resolveLinkSource(a.Source, configDir)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		switch a.Type {
		case plan.ActionLinkCreate:
			err = CreateLink(targetPath, sourcePath)
		case plan.ActionLinkUpdate:
			err = UpdateLink(targetPath, sourcePath)
		case plan.ActionLinkReplace:
			err = replaceLinkWithBackup(targetPath, sourcePath, opts.StateDir, opts.clock())
		}
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

func replaceLinkWithBackup(targetPath, sourcePath, stateDir string, at time.Time) error {
	if _, err := os.Lstat(targetPath); err == nil {
		if _, err := BackupAside(stateDir, targetPath, at); err != nil {
			return fmt.Errorf("link replace %s: %w", targetPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("link replace %s: %w", targetPath, err)
	}
	return CreateLink(targetPath, sourcePath)
}

// BackupAside moves path aside under stateDir/backups/files/<timestamp>/<escaped-path>.
// Returns the destination backup path.
func BackupAside(stateDir, path string, at time.Time) (string, error) {
	if strings.TrimSpace(stateDir) == "" {
		return "", fmt.Errorf("backup: state directory required")
	}
	stamp := at.UTC().Format("20060102T150405Z")
	destDir := filepath.Join(stateDir, "backups", "files", stamp)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("backup: create dir: %w", err)
	}
	dest := filepath.Join(destDir, EscapeBackupPath(path))
	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("backup move %s -> %s: %w", path, dest, err)
	}
	return dest, nil
}

// EscapeBackupPath turns an absolute path into a single path segment for backup storage.
func EscapeBackupPath(path string) string {
	cleaned := filepath.Clean(path)
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		switch r {
		case '/', '\\', ':':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" || out == "." {
		return "_root"
	}
	return out
}

func resolveLinkSource(source, configDir string) (string, error) {
	expanded, err := expandHomePath(source)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(configDir, expanded)), nil
}

func resolveLinkTarget(target string) (string, error) {
	expanded, err := expandHomePath(target)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("target %q must be absolute or start with ~", target)
	}
	return filepath.Clean(expanded), nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// ApplyManagedCopies runs managed_copy actions with atomic writes.
// Source paths are resolved relative to configDir; target paths expand ~.
// When FileReplace is backup, unexpected target types (e.g. directories) are moved aside first.
// Per-file failures are collected so later copies still run.
func ApplyManagedCopies(p plan.Plan, opts FileApplyOptions, progress Progress) (int, error) {
	configDir, err := filepath.Abs(opts.ConfigDir)
	if err != nil {
		return 0, fmt.Errorf("config directory: %w", err)
	}

	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionManagedCopy {
			continue
		}
		report(progress, a)

		sourcePath, err := resolveLinkSource(a.Source, configDir)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		backupFirst := opts.FileReplace == config.FileReplaceBackup || a.Kind == "backup"
		if err := ManagedCopy(sourcePath, targetPath, ManagedCopyOptions{
			StateDir:    opts.StateDir,
			BackupFirst: backupFirst,
			Now:         opts.clock(),
		}); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// ManagedCopyOptions controls backup-before-write for unexpected targets.
type ManagedCopyOptions struct {
	StateDir    string
	BackupFirst bool
	Now         time.Time
}

// ManagedCopy copies sourcePath to targetPath atomically (temp file + rename).
// Creates parent directories when missing. If target is a symlink, removes it
// first so the result is a regular file. Refuses directory targets unless
// BackupFirst is set (moves the directory aside under StateDir, then writes).
func ManagedCopy(sourcePath, targetPath string, opts ...ManagedCopyOptions) error {
	var o ManagedCopyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("managed copy %s: read source: %w", targetPath, err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(sourcePath); err == nil {
		mode = info.Mode().Perm()
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("managed copy %s: parent dir: %w", targetPath, err)
	}

	if info, err := os.Lstat(targetPath); err == nil {
		if info.IsDir() {
			if !o.BackupFirst {
				return fmt.Errorf("managed copy %s: target is a directory", targetPath)
			}
			if _, err := BackupAside(o.StateDir, targetPath, o.Now); err != nil {
				return fmt.Errorf("managed copy %s: %w", targetPath, err)
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			if o.BackupFirst {
				if _, err := BackupAside(o.StateDir, targetPath, o.Now); err != nil {
					return fmt.Errorf("managed copy %s: %w", targetPath, err)
				}
			} else if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("managed copy %s: remove symlink: %w", targetPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("managed copy %s: %w", targetPath, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".pourover-managed-*")
	if err != nil {
		return fmt.Errorf("managed copy %s: create temp: %w", targetPath, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("managed copy %s: chmod temp: %w", targetPath, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("managed copy %s: write temp: %w", targetPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("managed copy %s: close temp: %w", targetPath, err)
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("managed copy %s: rename: %w", targetPath, err)
	}
	cleanup = false
	return nil
}

// ApplyFileUnlinks runs file_unlink actions with apply-time safety checks.
// Paths expand ~. Symlinks and regular files are removed; directories are refused.
// Per-path failures are collected so later unlinks still run.
func ApplyFileUnlinks(p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionFileUnlink {
			continue
		}
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		if err := SafeUnlink(targetPath); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// SafeUnlink removes a symlink or regular file at path. Refuses directories.
// Missing paths are a no-op success (already gone).
func SafeUnlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unlink %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("unlink %s: path is a directory (refusing unlink)", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("unlink %s: %w", path, err)
	}
	return nil
}

// ConfirmPrunes asks whether to proceed with removing PourOver-owned undeclared files.
// paths are target paths only (no action type prefix).
type ConfirmPrunes func(paths []string) bool

// ApplyFilePrunes runs file_prune actions according to files_mode.
//
//   - non_destructive: skip all prunes (no prompt); plan should normally be empty
//   - strict: prune without prompting
//   - safe: prompt once via confirm; if declined, skip all prunes
//
// confirm may be nil when mode is not safe (or when there are no prunes).
// Per-path failures are collected so later prunes still run.
func ApplyFilePrunes(p plan.Plan, mode config.FilesMode, confirm ConfirmPrunes, progress Progress) (int, error) {
	var prunes []plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionFilePrune {
			prunes = append(prunes, a)
		}
	}
	if len(prunes) == 0 {
		return 0, nil
	}

	switch mode {
	case config.FilesModeNonDestructive:
		return 0, nil
	case config.FilesModeSafe, "":
		paths := make([]string, len(prunes))
		for i, a := range prunes {
			paths[i] = a.Name
		}
		if confirm == nil || !confirm(paths) {
			return 0, nil
		}
	case config.FilesModeStrict:
		// no prompt
	default:
		paths := make([]string, len(prunes))
		for i, a := range prunes {
			paths[i] = a.Name
		}
		if confirm == nil || !confirm(paths) {
			return 0, nil
		}
	}

	n := 0
	var errs []error
	for _, a := range prunes {
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		if err := SafeUnlink(targetPath); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}
