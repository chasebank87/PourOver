package config

import (
	"fmt"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// LoadMacOSModule reads a macos.lua module that returns { defaults = { … } }.
func LoadMacOSModule(path string) (MacOSDefaults, error) {
	if _, err := os.Stat(path); err != nil {
		return MacOSDefaults{}, fmt.Errorf("macos module: %w", err)
	}

	L := lua.NewState()
	defer L.Close()
	L.OpenLibs()

	if err := L.DoFile(path); err != nil {
		return MacOSDefaults{}, fmt.Errorf("execute %s: %w", path, err)
	}

	lv := L.Get(-1)
	if lv == lua.LNil {
		return MacOSDefaults{}, fmt.Errorf("%s: module must return a table", path)
	}
	if lv.Type() != lua.LTTable {
		return MacOSDefaults{}, fmt.Errorf("%s: module must return a table, got %s", path, lv.Type())
	}

	macos, err := decodeMacOSTable(L, lv.(*lua.LTable))
	if err != nil {
		return MacOSDefaults{}, err
	}
	if errs := validateMacOS(macos); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return MacOSDefaults{}, fmt.Errorf("invalid macos module: %s", strings.Join(msgs, "; "))
	}
	return macos.Defaults, nil
}
