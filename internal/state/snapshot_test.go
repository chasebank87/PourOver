package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteLocalSnapshot_CopiesLockAndPlan(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "last-plan.json"), []byte(`{"actions":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	snap, err := WriteLocalSnapshot(dir, at)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "snapshots", "2026-05-18T12-00-00Z")
	if snap != want {
		t.Fatalf("snap = %q, want %q", snap, want)
	}
	for _, base := range []string{"lock.json", "last-plan.json"} {
		if _, err := os.Stat(filepath.Join(snap, base)); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
	}
}
