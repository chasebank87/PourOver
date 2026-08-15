package engine

import (
	"fmt"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/state"
)

// FinalizeOptions configures post-apply history and state persistence.
type FinalizeOptions struct {
	StateDir     string
	ConfigDir    string // used to resolve owned file paths; may be empty
	Manifest     config.Manifest
	GenerationID string
	PrunedPaths  []string        // absolute paths successfully pruned this apply
	Now          func() time.Time // optional; defaults to time.Now
}

// FinalizeApply records apply history and, on success, persists lock/last-plan
// and takes a snapshot/mirror. Matches CLI executeApply ordering:
// history always; persist + SnapshotAndMirror only when applyErr == nil.
// Callers that want config auto-push (CLI) should do that after a nil return.
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
	if applyErr != nil {
		return applyErr
	}
	if err := persistApplyState(opts.StateDir, opts.ConfigDir, opts.Manifest, p, opts.PrunedPaths, opts.GenerationID, at); err != nil {
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

func persistApplyState(stateDir, configDir string, manifest config.Manifest, p plan.Plan, prunedPaths []string, generationID string, at time.Time) error {
	prev, err := state.LoadLock(stateDir)
	if err != nil {
		return fmt.Errorf("load lock: %w", err)
	}
	owned, err := state.ComputeOwnedFiles(prev.OwnedFiles, p, configDir)
	if err != nil {
		return fmt.Errorf("compute owned files: %w", err)
	}
	owned = state.RemoveOwnedPaths(owned, prunedPaths)
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
