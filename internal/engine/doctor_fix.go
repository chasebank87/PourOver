package engine

import (
	"fmt"
	"os"

	"github.com/chasebank87/PourOver/internal/scaffold"
)

// EnsureStateDir creates stateDir (and parents) with mode 0755.
func EnsureStateDir(stateDir string) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	return nil
}

// InitConfig writes the default PourOver scaffold into cfgDir.
// Doctor fixes must call with force=false only.
func InitConfig(cfgDir string, force bool) error {
	return scaffold.InitConfigDir(cfgDir, force)
}
