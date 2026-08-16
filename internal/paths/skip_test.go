package paths

import "testing"

func TestSkipFileName(t *testing.T) {
	t.Parallel()
	skip := []string{".DS_Store", ".ds_store", "Thumbs.db", "thumbs.db", "desktop.ini", "Desktop.ini", "._foo", "._Icon", "settings.cpython-313.pyc", "foo.pyo", "statsig-cache.json", "app.log", "Debug.LOG"}
	keep := []string{"nvim", "init.lua", ".zshrc", ".gitconfig", "ghostty", "foo.DS_Store", "settings.py", "cli-config.json"}
	for _, name := range skip {
		if !SkipFileName(name) {
			t.Errorf("SkipFileName(%q) = false, want true", name)
		}
	}
	for _, name := range keep {
		if SkipFileName(name) {
			t.Errorf("SkipFileName(%q) = true, want false", name)
		}
	}
}

func TestSkipWalkDir(t *testing.T) {
	t.Parallel()
	if !SkipWalkDir(".git") {
		t.Fatal("SkipWalkDir(.git) = false, want true")
	}
	if !SkipWalkDir(".GIT") {
		t.Fatal("SkipWalkDir(.GIT) = false, want true")
	}
	if SkipWalkDir("nvim") {
		t.Fatal("SkipWalkDir(nvim) = true, want false")
	}
	if !SkipWalkDir(".DS_Store") {
		t.Fatal("SkipWalkDir(.DS_Store) = false, want true")
	}
	if !SkipWalkDir("__pycache__") {
		t.Fatal("SkipWalkDir(__pycache__) = false, want true")
	}
	for _, name := range []string{"Cache", "CachedData", "Code Cache", "GPUCache", "logs", "Crashpad"} {
		if !SkipWalkDir(name) {
			t.Errorf("SkipWalkDir(%q) = false, want true", name)
		}
	}
}

func TestSkipOwnedPath(t *testing.T) {
	t.Parallel()
	if !SkipOwnedPath("/Users/x/.config/cursor/statsig-cache.json") {
		t.Fatal("want skip statsig-cache.json")
	}
	if !SkipOwnedPath("/Users/x/.config/app/Cache/foo") {
		t.Fatal("want skip under Cache/")
	}
	if SkipOwnedPath("/Users/x/.config/cursor/cli-config.json") {
		t.Fatal("want keep cli-config.json")
	}
}

func TestFilterOwnedPaths(t *testing.T) {
	t.Parallel()
	got := FilterOwnedPaths([]string{
		"/tmp/keep.txt",
		"/tmp/statsig-cache.json",
		"/tmp/Cache/x",
	})
	if len(got) != 1 || got[0] != "/tmp/keep.txt" {
		t.Fatalf("got %#v", got)
	}
}
