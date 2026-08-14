package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestAppendHistory_WritesTimestampedFile(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 5, 18, 12, 30, 45, 0, time.UTC)
	entry := HistoryEntry{
		Timestamp:    ts.UTC().Format(time.RFC3339),
		Success:      true,
		ActionCount:  1,
		Actions:      []plan.Action{{Type: plan.ActionFormulaInstall, Name: "fzf"}},
		ManifestHash: "abc",
	}

	path, err := AppendHistory(dir, entry, ts)
	if err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}
	if !strings.HasPrefix(filepath.Base(path), "2026-05-18T12-30-45Z") {
		t.Fatalf("filename = %q, want ISO-like prefix", filepath.Base(path))
	}
	if filepath.Dir(path) != paths.HistoryDir(dir) {
		t.Fatalf("dir = %q", filepath.Dir(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got HistoryEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Success || got.ActionCount != 1 || got.Actions[0].Name != "fzf" {
		t.Fatalf("got %#v", got)
	}
}

func TestAppendHistory_FailureEntry(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 5, 18, 13, 0, 0, 0, time.UTC)
	entry := HistoryEntry{
		Timestamp:   ts.UTC().Format(time.RFC3339),
		Success:     false,
		Error:       "brew install fzf: exit 1",
		ActionCount: 2,
		Actions: []plan.Action{
			{Type: plan.ActionFormulaInstall, Name: "fzf"},
			{Type: plan.ActionCaskInstall, Name: "raycast"},
		},
	}
	path, err := AppendHistory(dir, entry, ts)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got HistoryEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Success || got.Error == "" {
		t.Fatalf("got %#v", got)
	}
}
