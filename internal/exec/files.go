package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ApplyFileLinks runs link_create and link_update actions.
// Source paths are resolved relative to configDir; target paths expand ~.
// Per-link failures are collected so later links still run (same soft-fail model as brew).
func ApplyFileLinks(p plan.Plan, configDir string, progress Progress) (int, error) {
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return 0, fmt.Errorf("config directory: %w", err)
	}

	n := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionLinkCreate, plan.ActionLinkUpdate:
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
// Per-file failures are collected so later copies still run.
func ApplyManagedCopies(p plan.Plan, configDir string, progress Progress) (int, error) {
	configDir, err := filepath.Abs(configDir)
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

		if err := ManagedCopy(sourcePath, targetPath); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// ManagedCopy copies sourcePath to targetPath atomically (temp file + rename).
// Creates parent directories when missing. If target is a symlink, removes it
// first so the result is a regular file. Refuses directory targets.
func ManagedCopy(sourcePath, targetPath string) error {
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
			return fmt.Errorf("managed copy %s: target is a directory", targetPath)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(targetPath); err != nil {
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
