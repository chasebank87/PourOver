package state

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
)

// HistoryEntry records one apply attempt (success or failure).
type HistoryEntry struct {
	Timestamp    string        `json:"timestamp"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	ActionCount  int           `json:"action_count"`
	Actions      []plan.Action `json:"actions,omitempty"`
	ManifestHash string        `json:"manifest_hash,omitempty"`
}

// AppendHistory writes history/<timestamp>.json under stateDir and returns the path.
func AppendHistory(stateDir string, entry HistoryEntry, at time.Time) (string, error) {
	dir := paths.HistoryDir(stateDir)
	name := historyFileName(at.UTC())
	path := filepath.Join(dir, name)
	if err := WriteJSONAtomic(path, entry); err != nil {
		return "", fmt.Errorf("write history: %w", err)
	}
	return path, nil
}

func historyFileName(at time.Time) string {
	// RFC3339 uses colons; replace for filesystem-safe names.
	stamp := at.UTC().Format("2006-01-02T15-04-05Z")
	return stamp + ".json"
}

// NewHistoryEntry builds a history record from a plan and optional error.
func NewHistoryEntry(p plan.Plan, manifestHash string, at time.Time, applyErr error) HistoryEntry {
	entry := HistoryEntry{
		Timestamp:    at.UTC().Format(time.RFC3339),
		Success:      applyErr == nil,
		ActionCount:  len(p.Actions),
		Actions:      append([]plan.Action(nil), p.Actions...),
		ManifestHash: manifestHash,
	}
	if applyErr != nil {
		entry.Error = applyErr.Error()
	}
	return entry
}
