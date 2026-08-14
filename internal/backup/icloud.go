package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

// DefaultICloudDir returns ~/Library/Mobile Documents/com~apple~CloudDocs/PourOver.
// Decision D5 (2026-08-14): v1 default iCloud mirror root.
func DefaultICloudDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs", "PourOver"), nil
}

// ResolveICloudDir returns the mirror destination when iCloud backup is enabled
// and available. If disabled or the parent path is missing (iCloud Drive offline /
// not signed in), enabled is false and err is nil.
func ResolveICloudDir(m config.Manifest) (path string, enabled bool, err error) {
	if !m.Backup.ICloud.Enabled {
		return "", false, nil
	}

	dest := strings.TrimSpace(m.Backup.ICloud.Path)
	if dest == "" {
		dest, err = DefaultICloudDir()
		if err != nil {
			return "", false, err
		}
	} else {
		dest, err = expandHome(dest)
		if err != nil {
			return "", false, err
		}
	}

	parent := filepath.Dir(dest)
	if _, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("icloud path parent: %w", err)
	}
	return dest, true, nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
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
