package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestDefaultICloudDir_UnderCloudDocs(t *testing.T) {
	got, err := DefaultICloudDir()
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := filepath.Join("Library", "Mobile Documents", "com~apple~CloudDocs", "PourOver")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("DefaultICloudDir = %q, want suffix %q", got, wantSuffix)
	}
}

func TestResolveICloudDir_Disabled(t *testing.T) {
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{Enabled: false}}}
	got, enabled, err := ResolveICloudDir(m)
	if err != nil {
		t.Fatal(err)
	}
	if enabled || got != "" {
		t.Fatalf("got path=%q enabled=%v, want disabled", got, enabled)
	}
}

func TestResolveICloudDir_UsesOverride(t *testing.T) {
	override := t.TempDir()
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{
		Enabled: true,
		Path:    override,
	}}}
	got, enabled, err := ResolveICloudDir(m)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || got != override {
		t.Fatalf("got path=%q enabled=%v", got, enabled)
	}
}

func TestResolveICloudDir_UnavailableWhenMissing(t *testing.T) {
	// Override to a path whose parent does not exist → unavailable, not hard error.
	missingParent := filepath.Join(t.TempDir(), "nope", "PourOver")
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{
		Enabled: true,
		Path:    missingParent,
	}}}
	got, enabled, err := ResolveICloudDir(m)
	if err != nil {
		t.Fatalf("expected soft unavailable, got err %v", err)
	}
	if enabled {
		t.Fatal("expected enabled=false when iCloud path unavailable")
	}
	if got != "" {
		t.Fatalf("path = %q, want empty when unavailable", got)
	}
}

func TestResolveICloudDir_AvailableWhenParentExists(t *testing.T) {
	parent := t.TempDir()
	dest := filepath.Join(parent, "PourOver")
	m := config.Manifest{Backup: config.Backup{ICloud: config.ICloudBackup{
		Enabled: true,
		Path:    dest,
	}}}
	got, enabled, err := ResolveICloudDir(m)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || got != dest {
		t.Fatalf("got path=%q enabled=%v", got, enabled)
	}
	// Destination itself need not exist yet; parent must.
	if _, err := os.Stat(dest); !os.IsNotExist(err) && err != nil {
		t.Fatal(err)
	}
}
