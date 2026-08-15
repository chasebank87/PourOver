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

func TestFinalizeApply_PersistsOwnedFilesFromPlan(t *testing.T) {
	dir := t.TempDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe},
	}
	prevOwned := []string{filepath.Join(home, ".old"), filepath.Join(home, ".gone")}
	if err := state.PersistApplyState(dir, manifest, plan.Plan{}, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), prevOwned, ""); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: "~/.new", Source: "config/new"},
		{Type: plan.ActionFileUnlink, Name: "~/.gone"},
	}}
	at := time.Date(2026, 8, 15, 14, 0, 0, 0, time.UTC)

	if err := FinalizeApply(FinalizeOptions{
		StateDir:  dir,
		ConfigDir: "/cfg",
		Manifest:  manifest,
		Now:       func() time.Time { return at },
	}, p, nil); err != nil {
		t.Fatalf("FinalizeApply: %v", err)
	}

	lock, err := state.LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(home, ".new"), filepath.Join(home, ".old")}
	if len(lock.OwnedFiles) != len(want) {
		t.Fatalf("OwnedFiles = %#v, want %#v", lock.OwnedFiles, want)
	}
	for i := range want {
		if lock.OwnedFiles[i] != want[i] {
			t.Fatalf("OwnedFiles = %#v, want %#v", lock.OwnedFiles, want)
		}
	}
}

func TestFinalizeApply_KeepsOwnedWhenPruneDeclined(t *testing.T) {
	dir := t.TempDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(home, ".prune-me")
	keep := filepath.Join(home, ".keep")
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe, FilesMode: config.FilesModeSafe},
	}
	prevOwned := []string{keep, candidate}
	if err := state.PersistApplyState(dir, manifest, plan.Plan{}, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), prevOwned, ""); err != nil {
		t.Fatal(err)
	}

	// Plan still lists file_prune, but PrunedPaths is empty (safe confirm declined).
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFilePrune, Name: candidate},
	}}
	at := time.Date(2026, 8, 15, 15, 0, 0, 0, time.UTC)

	if err := FinalizeApply(FinalizeOptions{
		StateDir:    dir,
		ConfigDir:   "/cfg",
		Manifest:    manifest,
		PrunedPaths: nil,
		Now:         func() time.Time { return at },
	}, p, nil); err != nil {
		t.Fatalf("FinalizeApply: %v", err)
	}

	lock, err := state.LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{keep, candidate}
	if len(lock.OwnedFiles) != len(want) {
		t.Fatalf("OwnedFiles = %#v, want %#v (declined prune must keep ownership)", lock.OwnedFiles, want)
	}
	for i := range want {
		if lock.OwnedFiles[i] != want[i] {
			t.Fatalf("OwnedFiles = %#v, want %#v", lock.OwnedFiles, want)
		}
	}
}

func TestFinalizeApply_RemovesOnlyActuallyPrunedPaths(t *testing.T) {
	dir := t.TempDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	pruned := filepath.Join(home, ".pruned")
	kept := filepath.Join(home, ".still-owned")
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe, FilesMode: config.FilesModeSafe},
	}
	prevOwned := []string{kept, pruned}
	if err := state.PersistApplyState(dir, manifest, plan.Plan{}, time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), prevOwned, ""); err != nil {
		t.Fatal(err)
	}

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFilePrune, Name: pruned},
		{Type: plan.ActionFilePrune, Name: kept}, // in plan but not actually pruned (soft-fail / partial)
	}}
	at := time.Date(2026, 8, 15, 16, 0, 0, 0, time.UTC)

	if err := FinalizeApply(FinalizeOptions{
		StateDir:    dir,
		ConfigDir:   "/cfg",
		Manifest:    manifest,
		PrunedPaths: []string{pruned},
		Now:         func() time.Time { return at },
	}, p, nil); err != nil {
		t.Fatalf("FinalizeApply: %v", err)
	}

	lock, err := state.LoadLock(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{kept}
	if len(lock.OwnedFiles) != 1 || lock.OwnedFiles[0] != kept {
		t.Fatalf("OwnedFiles = %#v, want %#v", lock.OwnedFiles, want)
	}
}
