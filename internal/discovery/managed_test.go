package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestDiscoverManagedFiles_MissingTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "foo.conf")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "home", "foo.conf")

	statuses, err := DiscoverManagedFiles([]config.ManagedFile{
		{Source: "foo.conf", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverManagedFiles: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Kind != ManagedStatusMissing {
		t.Fatalf("status = %#v, want missing", statuses)
	}
}

func TestDiscoverManagedFiles_SameContent(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "foo.conf")
	target := filepath.Join(root, "tgt.conf")
	content := []byte("same-bytes\n")
	if err := os.WriteFile(source, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverManagedFiles([]config.ManagedFile{
		{Source: "foo.conf", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverManagedFiles: %v", err)
	}
	if statuses[0].Kind != ManagedStatusSame {
		t.Fatalf("status = %#v, want same", statuses[0])
	}
}

func TestDiscoverManagedFiles_ContentDiffers(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "foo.conf")
	target := filepath.Join(root, "tgt.conf")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	statuses, err := DiscoverManagedFiles([]config.ManagedFile{
		{Source: "foo.conf", Target: target},
	}, root)
	if err != nil {
		t.Fatalf("DiscoverManagedFiles: %v", err)
	}
	if statuses[0].Kind != ManagedStatusDiffer {
		t.Fatalf("status = %#v, want differ", statuses[0])
	}
}

func TestDiscoverManagedFiles_TargetDirectory(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "foo.conf")
	target := filepath.Join(root, "tgt-dir")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverManagedFiles([]config.ManagedFile{
		{Source: "foo.conf", Target: target},
	}, root)
	if err == nil {
		t.Fatal("expected error when target is a directory")
	}
}

func TestDiscoverManagedFiles_MissingSource(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "tgt.conf")

	_, err := DiscoverManagedFiles([]config.ManagedFile{
		{Source: "missing.conf", Target: target},
	}, root)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

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
