package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

	if err := PersistApplyState(dir, manifest, p, appliedAt, nil); err != nil {
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
	if lock.OwnedFiles != nil {
		t.Fatalf("owned_files = %#v, want nil when empty", lock.OwnedFiles)
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

func TestLoadLock_MissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	lock, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if lock.ManifestHash != "" || lock.AppliedAt != "" || lock.OwnedFiles != nil {
		t.Fatalf("lock = %#v, want empty with nil OwnedFiles", lock)
	}
}

func TestLoadLock_CorruptJSONReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(paths.LockFile(dir), []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLock(dir)
	if err == nil {
		t.Fatal("expected error for corrupt lock.json")
	}
}

func TestLoadLock_OldLockWithoutOwnedFiles(t *testing.T) {
	dir := t.TempDir()
	old := `{"manifest_hash":"abc","applied_at":"2026-05-18T12:00:00Z"}` + "\n"
	if err := os.WriteFile(paths.LockFile(dir), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if lock.ManifestHash != "abc" {
		t.Fatalf("manifest_hash = %q", lock.ManifestHash)
	}
	if lock.OwnedFiles != nil {
		t.Fatalf("OwnedFiles = %#v, want nil for old locks", lock.OwnedFiles)
	}
}

func TestPersistApplyState_OwnedFilesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	manifest := config.Manifest{
		Packages: config.Packages{Formulae: []string{"git"}},
		Policy:   config.Policy{UninstallMode: config.UninstallModeSafe},
	}
	p := plan.Plan{}
	appliedAt := time.Date(2026, 8, 15, 1, 0, 0, 0, time.UTC)
	owned := []string{"/tmp/pourover-a", "/tmp/pourover-b"}

	if err := PersistApplyState(dir, manifest, p, appliedAt, owned); err != nil {
		t.Fatalf("PersistApplyState: %v", err)
	}

	lock, err := LoadLock(dir)
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if !reflect.DeepEqual(lock.OwnedFiles, owned) {
		t.Fatalf("OwnedFiles = %#v, want %#v", lock.OwnedFiles, owned)
	}
}

func TestComputeOwnedFiles_AddsCreatesRemovesUnlinks(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	prev := []string{
		filepath.Join(home, ".keep"),
		filepath.Join(home, ".gone"),
	}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: "~/.newlink", Source: "config/new"},
		{Type: plan.ActionLinkUpdate, Name: "~/.keep", Source: "config/keep"},
		{Type: plan.ActionManagedCopy, Name: "~/.managed", Source: "config/managed"},
		{Type: plan.ActionLinkReplace, Name: "~/.replaced", Source: "config/replaced"},
		{Type: plan.ActionFileUnlink, Name: "~/.gone"},
		{Type: plan.ActionFormulaInstall, Name: "git"},
	}}

	got, err := ComputeOwnedFiles(prev, p, "/cfg")
	if err != nil {
		t.Fatalf("ComputeOwnedFiles: %v", err)
	}
	want := []string{
		filepath.Join(home, ".keep"),
		filepath.Join(home, ".managed"),
		filepath.Join(home, ".newlink"),
		filepath.Join(home, ".replaced"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("owned = %#v, want %#v", got, want)
	}
}

func TestComputeOwnedFiles_RejectsRelativeTarget(t *testing.T) {
	_, err := ComputeOwnedFiles(nil, plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkCreate, Name: "relative/path", Source: "src"},
	}}, "/cfg")
	if err == nil {
		t.Fatal("expected error for relative target")
	}
}
