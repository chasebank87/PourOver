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

// ResolveConfigFile returns flagPath when set, otherwise the default config path.
func ResolveConfigFile(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}
	return DefaultConfigFile()
}
