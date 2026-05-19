package config

import (
	"fmt"
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

// LoadManifest reads and decodes a pourover.lua file at path.
func LoadManifest(path string) (Manifest, error) {
	if _, err := os.Stat(path); err != nil {
		return Manifest{}, fmt.Errorf("config file: %w", err)
	}

	L := lua.NewState()
	defer L.Close()

	L.OpenLibs()
	if err := prependPackagePath(L, filepath.Dir(path)); err != nil {
		return Manifest{}, err
	}

	if err := L.DoFile(path); err != nil {
		return Manifest{}, fmt.Errorf("execute %s: %w", path, err)
	}

	manifest, err := decodeManifest(L, L.Get(-1))
	if err != nil {
		return Manifest{}, err
	}
	if err := Validate(&manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}
