package generation_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestApplyMigratesLegacySymlinkToRegularFile(t *testing.T) {
	configDir := t.TempDir()
	stateDir := t.TempDir()
	src := filepath.Join(configDir, "config", "zshrc")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export NEW=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.Symlink(src, target); err != nil {
		t.Fatal(err)
	}

	m := config.Manifest{
		Files: config.Files{
			Links: []config.FileLink{{Source: "config/zshrc", Target: target}},
		},
	}
	res, err := generation.Build(stateDir, configDir, m, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	statuses, err := generation.DiscoverFiles(res.Manifest.Files)
	if err != nil {
		t.Fatal(err)
	}
	p, err := plan.BuildGenerationFilePlan(statuses, config.FileReplaceError)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 1 || p.Actions[0].Type != plan.ActionLinkUpdate {
		t.Fatalf("plan = %+v, want link_update for legacy symlink", p.Actions)
	}

	n, err := exec.ApplyFileLinks(p, exec.FileApplyOptions{
		StateDir:     stateDir,
		GenerationID: res.Manifest.ID,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("target still a symlink after apply")
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "export NEW=1\n" {
		t.Fatalf("got %q err=%v", got, err)
	}
}
