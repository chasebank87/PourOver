package paths

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigFile_FlagOverride(t *testing.T) {
	got, err := ResolveConfigFile("/tmp/custom.lua")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/tmp/custom.lua" {
		t.Errorf("got %q, want /tmp/custom.lua", got)
	}
}

func TestDefaultConfigDir_UnderHome(t *testing.T) {
	got, err := DefaultConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != ConfigDirName {
		t.Errorf("base = %q, want %q", filepath.Base(got), ConfigDirName)
	}
}

func TestDefaultConfigFile_UnderHome(t *testing.T) {
	got, err := DefaultConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != ConfigFileName {
		t.Errorf("base = %q, want %q", filepath.Base(got), ConfigFileName)
	}
	if filepath.Base(filepath.Dir(got)) != ConfigDirName {
		t.Errorf("dir = %q, want %q", filepath.Dir(got), ConfigDirName)
	}
}

func TestDisplayHome(t *testing.T) {
	home, err := DefaultConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	home = filepath.Dir(home) // $HOME
	got := DisplayHome(filepath.Join(home, ".config", "nvim", ".DS_Store"))
	if got != "~/.config/nvim/.DS_Store" {
		t.Fatalf("DisplayHome = %q, want ~/.config/nvim/.DS_Store", got)
	}
	if DisplayHome(home) != "~" {
		t.Fatalf("DisplayHome(home) = %q, want ~", DisplayHome(home))
	}
	if DisplayHome("/tmp/x") != "/tmp/x" {
		t.Fatalf("DisplayHome abs = %q", DisplayHome("/tmp/x"))
	}
}
