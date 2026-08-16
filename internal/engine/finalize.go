package engine

import (
	"errors"
	"fmt"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/state"
)

// FinalizeOptions configures post-apply history and state persistence.
type FinalizeOptions struct {
	StateDir     string
	ConfigDir    string // used to resolve owned file paths; may be empty
	Manifest     config.Manifest
	GenerationID string
	PrunedPaths  []string // absolute paths successfully pruned this apply
	// SucceededFileTargets are absolute paths written by link/managed/template.
	SucceededFileTargets []string
	// UnlinkedPaths are absolute paths successfully removed via file_unlink.
	UnlinkedPaths []string
	Now           func() time.Time // optional; defaults to time.Now
}

// FinalizeApply records apply history and merges succeeded file ownership into
// lock.json even when applyErr != nil (partial apply). Snapshot/mirror and
// generation.SetCurrent run only on full success so a failed apply does not
// look fully activated.
func FinalizeApply(opts FinalizeOptions, p plan.Plan, applyErr error) error {
	if opts.StateDir == "" {
		return applyErr
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	at := now()

	histErr := appendApplyHistory(opts.StateDir, opts.Manifest, p, at, applyErr)
	if histErr != nil && applyErr == nil {
		return histErr
	}

	ownedErr := mergeOwnedFiles(opts)
	if applyErr != nil {
		if ownedErr != nil {
			return errors.Join(applyErr, ownedErr)
		}
		return applyErr
	}
	if ownedErr != nil {
		return ownedErr
	}
	if err := persistApplyState(opts.StateDir, opts.ConfigDir, opts.Manifest, p, opts.PrunedPaths, opts.SucceededFileTargets, opts.UnlinkedPaths, opts.GenerationID, at); err != nil {
		return err
	}
	return nil
}

func appendApplyHistory(stateDir string, manifest config.Manifest, p plan.Plan, at time.Time, applyErr error) error {
	hash, err := state.ManifestHash(manifest)
	if err != nil {
		return fmt.Errorf("history manifest hash: %w", err)
	}
	entry := state.NewHistoryEntry(p, hash, at, applyErr)
	if _, err := state.AppendHistory(stateDir, entry, at); err != nil {
		return fmt.Errorf("persist history: %w", err)
	}
	return nil
}

// mergeOwnedFiles updates lock.json owned_files from succeeded file ops without
// claiming a successful apply (preserves prior ManifestHash / GenerationID).
func mergeOwnedFiles(opts FinalizeOptions) error {
	prev, err := state.LoadLock(opts.StateDir)
	if err != nil {
		return fmt.Errorf("load lock: %w", err)
	}
	owned := state.AddOwnedPaths(prev.OwnedFiles, opts.SucceededFileTargets)
	owned = state.RemoveOwnedPaths(owned, opts.UnlinkedPaths)
	owned = state.RemoveOwnedPaths(owned, opts.PrunedPaths)
	if sameStringSlice(prev.OwnedFiles, owned) {
		return nil
	}
	prev.OwnedFiles = owned
	if err := state.WriteJSONAtomic(paths.LockFile(opts.StateDir), prev); err != nil {
		return fmt.Errorf("write lock owned_files: %w", err)
	}
	return nil
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func persistApplyState(stateDir, configDir string, manifest config.Manifest, p plan.Plan, prunedPaths, succeededFiles, unlinkedPaths []string, generationID string, at time.Time) error {
	prev, err := state.LoadLock(stateDir)
	if err != nil {
		return fmt.Errorf("load lock: %w", err)
	}
	// Prefer succeeded paths; fall back to plan-derived ownership for older callers.
	var owned []string
	if len(succeededFiles) > 0 || len(unlinkedPaths) > 0 || len(prunedPaths) > 0 {
		owned = state.AddOwnedPaths(prev.OwnedFiles, succeededFiles)
		owned = state.RemoveOwnedPaths(owned, unlinkedPaths)
		owned = state.RemoveOwnedPaths(owned, prunedPaths)
	} else {
		owned, err = state.ComputeOwnedFiles(prev.OwnedFiles, p, configDir)
		if err != nil {
			return fmt.Errorf("compute owned files: %w", err)
		}
		owned = state.RemoveOwnedPaths(owned, prunedPaths)
	}
	if err := state.PersistApplyState(stateDir, manifest, p, at, owned, generationID); err != nil {
		return fmt.Errorf("persist state: %w", err)
	}
	if generationID != "" {
		if err := generation.SetCurrent(stateDir, generationID); err != nil {
			return fmt.Errorf("set current generation: %w", err)
		}
		_ = generation.Prune(stateDir, generation.DefaultKeep)
	}
	if _, err := backup.SnapshotAndMirror(stateDir, manifest, at); err != nil {
		return fmt.Errorf("snapshot/mirror: %w", err)
	}
	return nil
}
