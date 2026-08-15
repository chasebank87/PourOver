package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	tmpl "github.com/chasebank87/PourOver/internal/template"
)

func TestDiscoverTemplateFiles_MissingSameDiffer(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(configDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	ctx := tmpl.Context{Hostname: "host", User: "alice", Home: home}
	srcBody := "user={{.User}}\n"
	rendered := "user=alice\n"

	if err := os.WriteFile(filepath.Join(configDir, "config", "a.tmpl"), []byte(srcBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config", "b.tmpl"), []byte(srcBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config", "c.tmpl"), []byte(srcBody), 0o644); err != nil {
		t.Fatal(err)
	}

	sameTarget := filepath.Join(home, "same")
	if err := os.WriteFile(sameTarget, []byte(rendered), 0o644); err != nil {
		t.Fatal(err)
	}
	differTarget := filepath.Join(home, "differ")
	if err := os.WriteFile(differTarget, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverTemplateFiles([]config.TemplateFile{
		{Source: "config/a.tmpl", Target: "~/missing"},
		{Source: "config/b.tmpl", Target: "~/same"},
		{Source: "config/c.tmpl", Target: "~/differ"},
	}, configDir, ctx)
	if err != nil {
		t.Fatalf("DiscoverTemplateFiles: %v", err)
	}
	if len(statuses) != 3 {
		t.Fatalf("len = %d", len(statuses))
	}
	if statuses[0].Kind != TemplateStatusMissing || statuses[0].Rendered != rendered {
		t.Fatalf("missing = %#v", statuses[0])
	}
	if statuses[1].Kind != TemplateStatusSame {
		t.Fatalf("same = %#v", statuses[1])
	}
	if statuses[2].Kind != TemplateStatusDiffer || statuses[2].Rendered != rendered {
		t.Fatalf("differ = %#v", statuses[2])
	}
}

func TestDiscoverTemplateFiles_BlockedDirectory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(configDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(configDir, "config", "x.tmpl"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverTemplateFiles([]config.TemplateFile{
		{Source: "config/x.tmpl", Target: "~/dir"},
	}, configDir, tmpl.Context{Hostname: "h", User: "u", Home: home})
	if err != nil {
		t.Fatalf("DiscoverTemplateFiles: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Kind != TemplateStatusBlocked {
		t.Fatalf("status = %#v", statuses)
	}
}
