package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chasebank87/PourOver/internal/paths"
)

// WriteLocalSnapshot copies current lock.json and last-plan.json into
// stateDir/snapshots/<timestamp>/ and returns that directory.
func WriteLocalSnapshot(stateDir string, at time.Time) (string, error) {
	name := at.UTC().Format("2006-01-02T15-04-05Z")
	dest := filepath.Join(paths.SnapshotsDir(stateDir), name)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}

	for _, base := range []string{"lock.json", "last-plan.json"} {
		src := filepath.Join(stateDir, base)
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("read %s: %w", base, err)
		}
		if err := os.WriteFile(filepath.Join(dest, base), data, 0o644); err != nil {
			return "", fmt.Errorf("write snapshot %s: %w", base, err)
		}
	}
	return dest, nil
}
