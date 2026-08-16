package discovery

import (
	"fmt"
	"os"
	"path/filepath"
)

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
