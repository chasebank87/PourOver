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
