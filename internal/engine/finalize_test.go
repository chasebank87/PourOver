package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/state"
)

func TestFinalizeApply_SuccessWritesHistoryAndLock(t *testing.T) {
	dir := t.TempDir()
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe},
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "git"},
	}}
	at := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if err := FinalizeApply(FinalizeOptions{
		StateDir: dir,
		Manifest: manifest,
		Now:      func() time.Time { return at },
	}, p, nil); err != nil {
		t.Fatalf("FinalizeApply: %v", err)
	}

	histDir := paths.HistoryDir(dir)
	entries, err := os.ReadDir(histDir)
	if err != nil {
		t.Fatalf("history dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(histDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var entry state.HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if !entry.Success || entry.ActionCount != 1 {
		t.Fatalf("history = %#v", entry)
	}

	if _, err := os.Stat(paths.LockFile(dir)); err != nil {
		t.Fatalf("lock.json missing: %v", err)
	}
	if _, err := os.Stat(paths.LastPlanFile(dir)); err != nil {
		t.Fatalf("last-plan.json missing: %v", err)
	}
}

func TestFinalizeApply_ApplyErrorStillWritesHistorySkipsLock(t *testing.T) {
	dir := t.TempDir()
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "git"},
	}}
	at := time.Date(2026, 8, 15, 13, 0, 0, 0, time.UTC)
	applyErr := errors.New("brew failed")

	err := FinalizeApply(FinalizeOptions{
		StateDir: dir,
		Manifest: manifest,
		Now:      func() time.Time { return at },
	}, p, applyErr)
	if !errors.Is(err, applyErr) {
		t.Fatalf("err = %v, want applyErr", err)
	}

	histDir := paths.HistoryDir(dir)
	entries, err := os.ReadDir(histDir)
	if err != nil {
		t.Fatalf("history dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("history entries = %d, want 1", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(histDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var entry state.HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Success || entry.Error == "" {
		t.Fatalf("history = %#v, want failure entry", entry)
	}

	if _, err := os.Stat(paths.LockFile(dir)); !os.IsNotExist(err) {
		t.Fatalf("lock.json should not exist on apply failure, err=%v", err)
	}
}

func TestFinalizeApply_EmptyStateDirNoop(t *testing.T) {
	err := FinalizeApply(FinalizeOptions{}, plan.Plan{}, nil)
	if err != nil {
		t.Fatalf("empty StateDir: %v", err)
	}
	err = FinalizeApply(FinalizeOptions{}, plan.Plan{}, errors.New("x"))
	if err == nil || err.Error() != "x" {
		t.Fatalf("want apply error returned, got %v", err)
	}
}
