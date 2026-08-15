package engine

import (
	"context"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
)

// BackupOptions configures a state snapshot (and optional iCloud mirror).
type BackupOptions struct {
	StateDir string
	Manifest config.Manifest
	Now      func() time.Time // optional; defaults to time.Now
}

// BackupResult describes where the snapshot was written.
type BackupResult struct {
	LocalSnapshot string
	MirroredTo    string
	ICloudEnabled bool
}

// Backup writes a local state snapshot and mirrors to iCloud when enabled/available.
func Backup(_ context.Context, opts BackupOptions) (BackupResult, error) {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	snap, err := backup.SnapshotAndMirror(opts.StateDir, opts.Manifest, now())
	if err != nil {
		return BackupResult{}, err
	}
	return BackupResult{
		LocalSnapshot: snap.LocalSnapshot,
		MirroredTo:    snap.MirroredTo,
		ICloudEnabled: opts.Manifest.Backup.ICloud.Enabled,
	}, nil
}

// RestoreOptions configures restoring lock/last-plan from a snapshot.
type RestoreOptions struct {
	StateDir   string
	Manifest   config.Manifest
	Snapshot   string // explicit snapshot directory; empty → latest
	FromICloud bool
}

// RestoreResult names the snapshot used and the state directory restored into.
type RestoreResult struct {
	SnapshotPath string
	StateDir     string
}

// Restore copies lock.json and last-plan.json from a snapshot into the state dir.
func Restore(_ context.Context, opts RestoreOptions) (RestoreResult, error) {
	snapPath, err := backup.ResolveSnapshotPath(opts.StateDir, opts.Manifest, opts.Snapshot, opts.FromICloud)
	if err != nil {
		return RestoreResult{}, err
	}
	if err := backup.RestoreSnapshot(snapPath, opts.StateDir); err != nil {
		return RestoreResult{}, err
	}
	return RestoreResult{SnapshotPath: snapPath, StateDir: opts.StateDir}, nil
}
