package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
)

// Lock is the fingerprint written after a successful apply.
type Lock struct {
	ManifestHash string   `json:"manifest_hash"`
	AppliedAt    string   `json:"applied_at"`
	OwnedFiles   []string `json:"owned_files,omitempty"` // absolute target paths PourOver manages
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

// LoadLock reads stateDir/lock.json. A missing file returns an empty Lock with
// nil OwnedFiles (not an error). Corrupt JSON returns an error. Old locks
// without owned_files unmarshal with nil OwnedFiles.
func LoadLock(stateDir string) (Lock, error) {
	path := paths.LockFile(stateDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Lock{}, nil
		}
		return Lock{}, fmt.Errorf("read lock.json: %w", err)
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return Lock{}, fmt.Errorf("parse lock.json: %w", err)
	}
	return lock, nil
}

// PersistApplyState writes lock.json and last-plan.json under stateDir.
// owned is the updated absolute path set callers computed (may be nil/empty).
func PersistApplyState(stateDir string, manifest config.Manifest, p plan.Plan, appliedAt time.Time, owned []string) error {
	hash, err := ManifestHash(manifest)
	if err != nil {
		return err
	}
	lock := Lock{
		ManifestHash: hash,
		AppliedAt:    appliedAt.UTC().Format(time.RFC3339),
		OwnedFiles:   owned,
	}
	if err := WriteJSONAtomic(paths.LockFile(stateDir), lock); err != nil {
		return fmt.Errorf("write lock.json: %w", err)
	}
	if err := WriteJSONAtomic(paths.LastPlanFile(stateDir), p); err != nil {
		return fmt.Errorf("write last-plan.json: %w", err)
	}
	return nil
}

// ComputeOwnedFiles updates the previous owned set from plan file actions.
// link_create / link_update / link_replace / managed_copy / template_write add
// absolute targets; file_unlink removes them. file_prune is not applied here —
// callers must pass actually pruned absolute paths to RemoveOwnedPaths after a
// successful apply. configDir is reserved for resolving relative paths in later
// phases; targets must already be absolute or ~/….
func ComputeOwnedFiles(prev []string, p plan.Plan, configDir string) ([]string, error) {
	_ = configDir
	set := make(map[string]struct{}, len(prev)+len(p.Actions))
	for _, path := range prev {
		if path == "" {
			continue
		}
		set[path] = struct{}{}
	}
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionLinkCreate, plan.ActionLinkUpdate, plan.ActionLinkReplace, plan.ActionManagedCopy, plan.ActionTemplateWrite:
			abs, err := absOwnedTarget(a.Name)
			if err != nil {
				return nil, err
			}
			set[abs] = struct{}{}
		case plan.ActionFileUnlink:
			abs, err := absOwnedTarget(a.Name)
			if err != nil {
				return nil, err
			}
			delete(set, abs)
		}
	}
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

// RemoveOwnedPaths drops absolute paths from the owned set (e.g. successfully pruned).
// Unknown paths are ignored. Returns a sorted copy.
func RemoveOwnedPaths(owned []string, remove []string) []string {
	if len(owned) == 0 || len(remove) == 0 {
		out := append([]string(nil), owned...)
		sort.Strings(out)
		return out
	}
	drop := make(map[string]struct{}, len(remove))
	for _, path := range remove {
		if path == "" {
			continue
		}
		drop[filepath.Clean(path)] = struct{}{}
	}
	out := make([]string, 0, len(owned))
	for _, path := range owned {
		if _, ok := drop[filepath.Clean(path)]; ok {
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func absOwnedTarget(target string) (string, error) {
	expanded, err := expandHomePath(target)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("owned target %q must be absolute or start with ~", target)
	}
	return filepath.Clean(expanded), nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
