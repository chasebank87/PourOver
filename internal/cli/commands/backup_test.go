package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
)

func TestNewRestoreCmd_HasFlags(t *testing.T) {
	cmd := NewRestoreCmd()
	if cmd.Flags().Lookup("snapshot") == nil || cmd.Flags().Lookup("icloud") == nil {
		t.Fatal("restore missing flags")
	}
	if NewBackupCmd().Use != "backup" {
		t.Fatal("backup cmd")
	}
}

func TestBackupRestore_RoundTripWithSnapshotFlag(t *testing.T) {
	stateDir := t.TempDir()
	iCloud := t.TempDir()
	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte(`{"manifest_hash":"a"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "last-plan.json"), []byte(`{"actions":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{Enabled: true, Path: iCloud}}}
	at := time.Date(2026, 5, 18, 16, 0, 0, 0, time.UTC)
	result, err := backup.SnapshotAndMirror(stateDir, m, at)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(stateDir, "lock.json"), []byte(`{"manifest_hash":"corrupt"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := backup.RestoreSnapshot(result.LocalSnapshot, stateDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(stateDir, "lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"a"`) {
		t.Fatalf("lock after restore = %s", data)
	}
}
