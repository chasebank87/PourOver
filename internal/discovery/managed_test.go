package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverUnlinkPaths_MissingIsNoop(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "gone")

	statuses, err := DiscoverUnlinkPaths([]string{path}, root)
	if err != nil {
		t.Fatalf("DiscoverUnlinkPaths: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Kind != UnlinkStatusMissing {
		t.Fatalf("status = %#v, want missing", statuses)
	}
}

func TestDiscoverUnlinkPaths_RegularFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "old")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverUnlinkPaths([]string{path}, root)
	if err != nil {
		t.Fatalf("DiscoverUnlinkPaths: %v", err)
	}
	if statuses[0].Kind != UnlinkStatusRemove {
		t.Fatalf("status = %#v, want remove", statuses[0])
	}
}

func TestDiscoverUnlinkPaths_AnySymlink(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverUnlinkPaths([]string{link}, root)
	if err != nil {
		t.Fatalf("DiscoverUnlinkPaths: %v", err)
	}
	if statuses[0].Kind != UnlinkStatusRemove {
		t.Fatalf("status = %#v, want remove for explicit symlink", statuses[0])
	}
}

func TestDiscoverUnlinkPaths_DirectoryError(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverUnlinkPaths([]string{dir}, root)
	if err == nil {
		t.Fatal("expected error for directory unlink")
	}
}
