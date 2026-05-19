package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestDiscoverFileLinks_MissingTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "tgt")

	statuses, err := DiscoverFileLinks([]config.FileLink{
		{Source: "src", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverFileLinks: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Kind != LinkStatusMissing {
		t.Fatalf("status = %#v, want missing", statuses[0])
	}
}

func TestDiscoverFileLinks_CorrectSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	target := filepath.Join(root, "tgt")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(source, target); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverFileLinks([]config.FileLink{
		{Source: "src", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverFileLinks: %v", err)
	}
	if statuses[0].Kind != LinkStatusCorrect {
		t.Fatalf("status = %#v, want correct", statuses[0])
	}
}

func TestDiscoverFileLinks_WrongSymlink(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	other := filepath.Join(root, "other")
	target := filepath.Join(root, "tgt")
	for _, dir := range []string{source, other} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(other, target); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverFileLinks([]config.FileLink{
		{Source: "src", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverFileLinks: %v", err)
	}
	if statuses[0].Kind != LinkStatusWrong {
		t.Fatalf("status = %#v, want wrong", statuses[0])
	}
	if statuses[0].ActualTarget == "" {
		t.Error("expected ActualTarget set for wrong link")
	}
}

func TestDiscoverFileLinks_BlockedByRegularFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	target := filepath.Join(root, "tgt")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("not a symlink"), 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverFileLinks([]config.FileLink{
		{Source: "src", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverFileLinks: %v", err)
	}
	if statuses[0].Kind != LinkStatusBlocked {
		t.Fatalf("status = %#v, want blocked", statuses[0])
	}
}

func TestDiscoverFileLinks_MissingSource(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "tgt")

	_, err := DiscoverFileLinks([]config.FileLink{
		{Source: "missing", Target: target},
	}, root)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestDiscoverFileLinks_HomeExpansion(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, ".pourover-test-link")
	t.Cleanup(func() { _ = os.Remove(target) })
	_ = os.Remove(target)

	statuses, err := DiscoverFileLinks([]config.FileLink{
		{Source: "src", Target: "~/.pourover-test-link"},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverFileLinks: %v", err)
	}
	if statuses[0].Kind != LinkStatusMissing {
		t.Fatalf("status = %#v, want missing before link created", statuses[0])
	}
}
