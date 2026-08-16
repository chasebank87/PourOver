package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/state"
)

func clearOwnedForRemovedLinks(stateDir string, removed []config.FileLink) ([]string, error) {
	lock, err := state.LoadLock(stateDir)
	if err != nil {
		return nil, err
	}
	if len(lock.OwnedFiles) == 0 {
		return nil, nil
	}

	var drop []string
	for _, link := range removed {
		abs, err := expandUnmanageTarget(link.Target)
		if err != nil {
			return nil, err
		}
		abs = filepath.Clean(abs)
		for _, owned := range lock.OwnedFiles {
			clean := filepath.Clean(owned)
			if clean == abs || strings.HasPrefix(clean, abs+string(os.PathSeparator)) {
				drop = append(drop, clean)
			}
		}
	}
	if len(drop) == 0 {
		return nil, nil
	}
	updated := state.RemoveOwnedPaths(lock.OwnedFiles, drop)
	lock.OwnedFiles = updated
	if err := state.WriteJSONAtomic(paths.LockFile(stateDir), lock); err != nil {
		return nil, fmt.Errorf("write lock.json: %w", err)
	}
	return drop, nil
}

func expandUnmanageTarget(target string) (string, error) {
	if target == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, target[2:]), nil
	}
	if !filepath.IsAbs(target) {
		return "", fmt.Errorf("target %q must be absolute or start with ~", target)
	}
	return target, nil
}
