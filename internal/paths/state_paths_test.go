package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultStateDir_UnderApplicationSupport(t *testing.T) {
	got, err := DefaultStateDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, filepath.Join("Library", "Application Support", "PourOver", "state")) {
		t.Fatalf("DefaultStateDir = %q, want .../Library/Application Support/PourOver/state", got)
	}
}

func TestStateArtifactPaths(t *testing.T) {
	root := "/tmp/pourover-state"
	if got := LockFile(root); got != filepath.Join(root, "lock.json") {
		t.Errorf("LockFile = %q", got)
	}
	if got := LastPlanFile(root); got != filepath.Join(root, "last-plan.json") {
		t.Errorf("LastPlanFile = %q", got)
	}
	if got := HistoryDir(root); got != filepath.Join(root, "history") {
		t.Errorf("HistoryDir = %q", got)
	}
}
