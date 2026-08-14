package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestManifestHash_ChangesWhenConfigChanges(t *testing.T) {
	a := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe},
	}
	b := a
	b.Packages.Formulae = []string{"git", "fzf"}

	ha, err := ManifestHash(a)
	if err != nil {
		t.Fatal(err)
	}
	hb, err := ManifestHash(b)
	if err != nil {
		t.Fatal(err)
	}
	if ha == hb {
		t.Fatal("expected different hashes for different manifests")
	}
	if ha == "" || len(ha) != 64 {
		t.Fatalf("hash = %q, want 64 hex chars", ha)
	}
}

func TestPersistApplyState_WritesLockAndLastPlan(t *testing.T) {
	dir := t.TempDir()
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe},
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
	}}
	appliedAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

	if err := PersistApplyState(dir, manifest, p, appliedAt); err != nil {
		t.Fatalf("PersistApplyState: %v", err)
	}

	lockPath := paths.LockFile(dir)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatal(err)
	}
	wantHash, err := ManifestHash(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if lock.ManifestHash != wantHash {
		t.Fatalf("manifest_hash = %q, want %q", lock.ManifestHash, wantHash)
	}
	if lock.AppliedAt != appliedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("applied_at = %q", lock.AppliedAt)
	}

	planPath := paths.LastPlanFile(dir)
	planData, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	var got plan.Plan
	if err := json.Unmarshal(planData, &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Actions) != 1 || got.Actions[0].Name != "fzf" {
		t.Fatalf("last-plan = %#v", got)
	}

	if filepath.Base(lockPath) != "lock.json" || filepath.Base(planPath) != "last-plan.json" {
		t.Fatal("unexpected artifact basenames")
	}
}
