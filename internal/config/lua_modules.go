package config

import (
	"fmt"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

// prependPackagePath adds <configDir>/?.lua to package.path for require().
func prependPackagePath(L *lua.LState, configDir string) error {
	pattern := filepath.ToSlash(filepath.Join(configDir, "?.lua"))
	script := fmt.Sprintf(`package.path = %q .. ";" .. package.path`, pattern)
	return L.DoString(script)
}
