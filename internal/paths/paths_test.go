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
