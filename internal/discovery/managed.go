package discovery

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
)

// DiscoverManagedFiles inspects each declared managed copy against the filesystem.
// Sources are resolved relative to configDir when not absolute; targets expand ~.
// Source must exist. A directory target is an error.
func DiscoverManagedFiles(managed []config.ManagedFile, configDir string) ([]ManagedStatus, error) {
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("config directory: %w", err)
	}

	statuses := make([]ManagedStatus, 0, len(managed))
	for i, file := range managed {
		st, err := inspectManagedFile(file, configDir)
		if err != nil {
			return nil, fmt.Errorf("files.managed[%d]: %w", i+1, err)
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func inspectManagedFile(file config.ManagedFile, configDir string) (ManagedStatus, error) {
	sourcePath, err := resolveSourcePath(file.Source, configDir)
	if err != nil {
		return ManagedStatus{}, err
	}
	targetPath, err := resolveTargetPath(file.Target)
	if err != nil {
		return ManagedStatus{}, err
	}

	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return ManagedStatus{}, fmt.Errorf("source %q does not exist", file.Source)
		}
		return ManagedStatus{}, fmt.Errorf("source %q: %w", file.Source, err)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ManagedStatus{
				File:       file,
				SourcePath: sourcePath,
				TargetPath: targetPath,
				Kind:       ManagedStatusMissing,
			}, nil
		}
		return ManagedStatus{}, err
	}

	if info.IsDir() {
		return ManagedStatus{
			File:       file,
			SourcePath: sourcePath,
			TargetPath: targetPath,
			Kind:       ManagedStatusBlocked,
		}, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(targetPath)
		if err == nil {
			if st, err := os.Stat(resolved); err == nil && st.IsDir() {
				return ManagedStatus{
					File:       file,
					SourcePath: sourcePath,
					TargetPath: targetPath,
					Kind:       ManagedStatusBlocked,
				}, nil
			}
		}
	}

	kind, err := compareManagedContent(sourcePath, targetPath)
	if err != nil {
		return ManagedStatus{}, err
	}
	return ManagedStatus{
		File:       file,
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Kind:       kind,
	}, nil
}

func compareManagedContent(sourcePath, targetPath string) (ManagedStatusKind, error) {
	want, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read source: %w", err)
	}
	got, err := os.ReadFile(targetPath)
	if err != nil {
		return "", fmt.Errorf("read target: %w", err)
	}
	if bytes.Equal(want, got) {
		return ManagedStatusSame, nil
	}
	return ManagedStatusDiffer, nil
}

// DiscoverUnlinkPaths inspects each explicit unlink path.
// Missing paths are noop. Symlinks and regular files are removable.
// Directories are refused.
func DiscoverUnlinkPaths(paths []string, configDir string) ([]UnlinkStatus, error) {
	if _, err := filepath.Abs(configDir); err != nil {
		return nil, fmt.Errorf("config directory: %w", err)
	}

	statuses := make([]UnlinkStatus, 0, len(paths))
	for i, path := range paths {
		st, err := inspectUnlinkPath(path)
		if err != nil {
			return nil, fmt.Errorf("files.unlink[%d]: %w", i+1, err)
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func inspectUnlinkPath(path string) (UnlinkStatus, error) {
	targetPath, err := resolveTargetPath(path)
	if err != nil {
		return UnlinkStatus{}, err
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return UnlinkStatus{
				Path:       path,
				TargetPath: targetPath,
				Kind:       UnlinkStatusMissing,
			}, nil
		}
		return UnlinkStatus{}, err
	}

	if info.IsDir() {
		return UnlinkStatus{}, fmt.Errorf("path %q is a directory (refusing unlink)", path)
	}

	// Explicit unlink list: allow any symlink or regular file the user declared.
	return UnlinkStatus{
		Path:       path,
		TargetPath: targetPath,
		Kind:       UnlinkStatusRemove,
	}, nil
}
