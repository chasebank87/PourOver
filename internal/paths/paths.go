package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the default directory under $HOME for pourover.lua.
	ConfigDirName = ".pourover"
	// ConfigFileName is the default declarative config file name.
	ConfigFileName = "pourover.lua"
	appSupportPourOver = "PourOver"
	stateDirName       = "state"
)

// DefaultConfigDir returns ~/.pourover.
func DefaultConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ConfigDirName), nil
}

// DefaultConfigFile returns ~/.pourover/pourover.lua.
func DefaultConfigFile() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// DefaultStateDir returns ~/Library/Application Support/PourOver/state.
func DefaultStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", appSupportPourOver, stateDirName), nil
}

// ResolveConfigFile returns flagPath when set, otherwise the default config path.
func ResolveConfigFile(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	return DefaultConfigFile()
}

// LockFile returns stateDir/lock.json.
func LockFile(stateDir string) string {
	return filepath.Join(stateDir, "lock.json")
}

// LastPlanFile returns stateDir/last-plan.json.
func LastPlanFile(stateDir string) string {
	return filepath.Join(stateDir, "last-plan.json")
}

// HistoryDir returns stateDir/history.
func HistoryDir(stateDir string) string {
	return filepath.Join(stateDir, "history")
}

// SnapshotsDir returns stateDir/snapshots.
func SnapshotsDir(stateDir string) string {
	return filepath.Join(stateDir, "snapshots")
}
