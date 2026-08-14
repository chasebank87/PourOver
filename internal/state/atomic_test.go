package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomic_WritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lock.json")
	payload := map[string]string{"manifest_hash": "abc"}

	if err := WriteJSONAtomic(path, payload); err != nil {
		t.Fatalf("WriteJSONAtomic: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["manifest_hash"] != "abc" {
		t.Fatalf("got %#v", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || len(e.Name()) > 4 && e.Name()[len(e.Name())-4:] == ".tmp" {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
		if filepath.Ext(e.Name()) != ".json" && !isTempSuffix(e.Name()) {
			// only lock.json expected
		}
	}
	if len(entries) != 1 || entries[0].Name() != "lock.json" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("dir entries = %v, want [lock.json]", names)
	}
}

func isTempSuffix(name string) bool {
	return len(name) >= 4 && name[len(name)-4:] == ".tmp"
}

func TestWriteJSONAtomic_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "a", "lock.json")
	if err := WriteJSONAtomic(path, map[string]int{"n": 1}); err != nil {
		t.Fatalf("WriteJSONAtomic: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
