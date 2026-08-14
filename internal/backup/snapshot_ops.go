package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/state"
)

// SnapshotResult describes a local snapshot and optional iCloud mirror destination.
type SnapshotResult struct {
	LocalSnapshot string
	MirroredTo    string
}

// SnapshotAndMirror writes a local snapshot and mirrors it to iCloud when enabled
// and available. Disabled or unavailable iCloud leaves MirroredTo empty (not an error).
func SnapshotAndMirror(stateDir string, m config.Manifest, at time.Time) (SnapshotResult, error) {
	local, err := state.WriteLocalSnapshot(stateDir, at)
	if err != nil {
		return SnapshotResult{}, err
	}
	result := SnapshotResult{LocalSnapshot: local}

	icloud, enabled, err := ResolveICloudDir(m)
	if err != nil {
		return result, err
	}
	if !enabled {
		return result, nil
	}
	mirrored, err := MirrorSnapshot(local, icloud)
	if err != nil {
		return result, err
	}
	result.MirroredTo = mirrored
	return result, nil
}

// RestoreSnapshot copies lock.json and last-plan.json from snapshotDir into stateDir.
func RestoreSnapshot(snapshotDir, stateDir string) error {
	info, err := os.Stat(snapshotDir)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("snapshot %s is not a directory", snapshotDir)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("state dir: %w", err)
	}
	for _, base := range []string{"lock.json", "last-plan.json"} {
		src := filepath.Join(snapshotDir, base)
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", base, err)
		}
		if err := writeBytesAtomic(filepath.Join(stateDir, base), data); err != nil {
			return fmt.Errorf("restore %s: %w", base, err)
		}
	}
	return nil
}

func writeBytesAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// LatestSnapshot returns the newest snapshot directory under stateDir/snapshots.
func LatestSnapshot(stateDir string) (string, error) {
	dir := paths.SnapshotsDir(stateDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("list snapshots: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no snapshots in %s", dir)
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1]), nil
}

// ResolveSnapshotPath returns an explicit snapshot path, or the latest under stateDir
// or under iCloud when fromICloud is true.
func ResolveSnapshotPath(stateDir string, m config.Manifest, explicit string, fromICloud bool) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if fromICloud {
		icloud, enabled, err := ResolveICloudDir(m)
		if err != nil {
			return "", err
		}
		if !enabled {
			return "", fmt.Errorf("iCloud backup is not enabled or unavailable")
		}
		// LatestSnapshot expects a state-like root containing snapshots/
		return LatestSnapshot(icloud)
	}
	return LatestSnapshot(stateDir)
}
