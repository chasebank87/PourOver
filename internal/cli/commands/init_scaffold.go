package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/paths"
)

//go:embed templates/pourover.lua templates/packages.lua
var scaffoldFS embed.FS

// InitConfigDir writes the default PourOver scaffold into cfgDir.
// If pourover.lua already exists and force is false, it returns an error.
func InitConfigDir(cfgDir string, force bool) error {
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	rootPath := filepath.Join(cfgDir, paths.ConfigFileName)
	if !force {
		if _, err := os.Stat(rootPath); err == nil {
			return fmt.Errorf("config already exists at %s (use --force to overwrite)", rootPath)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat config: %w", err)
		}
	}

	if err := writeEmbedded(cfgDir, "templates/pourover.lua", paths.ConfigFileName); err != nil {
		return err
	}
	if err := writeEmbedded(cfgDir, "templates/packages.lua", "packages.lua"); err != nil {
		return err
	}

	exampleDir := filepath.Join(cfgDir, "config", "nvim")
	if err := os.MkdirAll(exampleDir, 0o755); err != nil {
		return fmt.Errorf("create example config dir: %w", err)
	}
	keep := filepath.Join(exampleDir, ".keep")
	if force || !fileExists(keep) {
		if err := os.WriteFile(keep, []byte("Place nvim config files here and uncomment the link in pourover.lua.\n"), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", keep, err)
		}
	}
	return nil
}

func writeEmbedded(cfgDir, embedPath, destName string) error {
	data, err := scaffoldFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("read embed %s: %w", embedPath, err)
	}
	dest := filepath.Join(cfgDir, destName)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
