package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMirrorSnapshot_CopiesTree(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()
	ts := "2026-05-18T12-00-00Z"
	srcSnap := filepath.Join(srcRoot, "snapshots", ts)
	if err := os.MkdirAll(srcSnap, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcSnap, "lock.json"), []byte(`{"ok":true}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(srcSnap, "meta")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "note.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest, err := MirrorSnapshot(srcSnap, dstRoot)
	if err != nil {
		t.Fatalf("MirrorSnapshot: %v", err)
	}
	want := filepath.Join(dstRoot, "snapshots", ts)
	if dest != want {
		t.Fatalf("dest = %q, want %q", dest, want)
	}

	data, err := os.ReadFile(filepath.Join(want, "lock.json"))
	if err != nil || string(data) != `{"ok":true}`+"\n" {
		t.Fatalf("lock.json = %q err=%v", data, err)
	}
	note, err := os.ReadFile(filepath.Join(want, "meta", "note.txt"))
	if err != nil || string(note) != "hi\n" {
		t.Fatalf("note = %q err=%v", note, err)
	}
}
