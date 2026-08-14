package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
)

// Lock is the fingerprint written after a successful apply.
type Lock struct {
	ManifestHash string `json:"manifest_hash"`
	AppliedAt    string `json:"applied_at"`
}

// ManifestHash returns a stable SHA-256 hex digest of the manifest JSON.
func ManifestHash(m config.Manifest) (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal manifest: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// PersistApplyState writes lock.json and last-plan.json under stateDir.
func PersistApplyState(stateDir string, manifest config.Manifest, p plan.Plan, appliedAt time.Time) error {
	hash, err := ManifestHash(manifest)
	if err != nil {
		return err
	}
	lock := Lock{
		ManifestHash: hash,
		AppliedAt:    appliedAt.UTC().Format(time.RFC3339),
	}
	if err := WriteJSONAtomic(paths.LockFile(stateDir), lock); err != nil {
		return fmt.Errorf("write lock.json: %w", err)
	}
	if err := WriteJSONAtomic(paths.LastPlanFile(stateDir), p); err != nil {
		return fmt.Errorf("write last-plan.json: %w", err)
	}
	return nil
}
