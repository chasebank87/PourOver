package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/state"
)

func TestSnapshotAndMirror_WhenEnabled(t *testing.T) {
	stateDir := t.TempDir()
	iCloud := t.TempDir()
	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte(`{"manifest_hash":"x"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "last-plan.json"), []byte(`{"actions":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{
		Enabled: true,
		Path:    iCloud,
	}}}
	at := time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC)

	result, err := SnapshotAndMirror(stateDir, m, at)
	if err != nil {
		t.Fatalf("SnapshotAndMirror: %v", err)
	}
	if result.LocalSnapshot == "" || result.MirroredTo == "" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(result.LocalSnapshot, "lock.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.MirroredTo, "lock.json")); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotAndMirror_DisabledSkipsMirror(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{Enabled: false}}}
	at := time.Date(2026, 5, 18, 15, 0, 0, 0, time.UTC)
	result, err := SnapshotAndMirror(stateDir, m, at)
	if err != nil {
		t.Fatal(err)
	}
	if result.LocalSnapshot == "" {
		t.Fatal("expected local snapshot")
	}
	if result.MirroredTo != "" {
		t.Fatalf("unexpected mirror: %q", result.MirroredTo)
	}
}

func TestRestoreSnapshot_CopiesIntoStateDir(t *testing.T) {
	stateDir := t.TempDir()
	snap := filepath.Join(t.TempDir(), "snapshots", "2026-05-18T15-00-00Z")
	if err := os.MkdirAll(snap, 0o755); err != nil {
		t.Fatal(err)
	}
	lock := state.Lock{ManifestHash: "restored", AppliedAt: "2026-05-18T15:00:00Z"}
	data, _ := json.Marshal(lock)
	if err := os.WriteFile(filepath.Join(snap, "lock.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap, "last-plan.json"), []byte(`{"actions":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RestoreSnapshot(snap, stateDir); err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(stateDir, "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	var unlocked state.Lock
	if err := json.Unmarshal(got, &unlocked); err != nil {
		t.Fatal(err)
	}
	if unlocked.ManifestHash != "restored" {
		t.Fatalf("lock = %#v", unlocked)
	}
}

func TestLatestSnapshot_PicksNewest(t *testing.T) {
	stateDir := t.TempDir()
	snaps := filepath.Join(stateDir, "snapshots")
	for _, name := range []string{"2026-05-18T10-00-00Z", "2026-05-18T12-00-00Z", "2026-05-18T11-00-00Z"} {
		if err := os.MkdirAll(filepath.Join(snaps, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := LatestSnapshot(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "2026-05-18T12-00-00Z" {
		t.Fatalf("got %q", got)
	}
}
