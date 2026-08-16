package paths

import "testing"

func TestSkipFileName(t *testing.T) {
	t.Parallel()
	skip := []string{".DS_Store", ".ds_store", "Thumbs.db", "thumbs.db", "desktop.ini", "Desktop.ini", "._foo", "._Icon", "settings.cpython-313.pyc", "foo.pyo"}
	keep := []string{"nvim", "init.lua", ".zshrc", ".gitconfig", "ghostty", "foo.DS_Store", "settings.py"}
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
}
