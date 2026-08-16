package config

import (
	"strings"
	"testing"
)

func TestPatchFilesLinks_PreservesOtherSections(t *testing.T) {
	src := `local packages = require("packages")
local macos = require("macos")

return {
  packages = packages,
  macos = macos,
  files = {
    links = {
      { source = "config/old", target = "~/.config/old" },
    },
    managed = {
      { source = "config/foo.conf", target = "~/.config/foo.conf" },
    },
    templates = {
      { source = "config/gitconfig.tmpl", target = "~/.gitconfig" },
    },
  },
  policy = {
    uninstall_mode = "safe",
    files_mode = "safe",
  },
}
`
	out, err := PatchFilesLinks(src, []FileLink{
		{Source: "config/nvim", Target: "~/.config/nvim"},
		{Source: "config/home/zshrc", Target: "~/.zshrc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, frag := range []string{
		`macos = macos`,
		`source = "config/nvim"`,
		`target = "~/.zshrc"`,
		`managed = {`,
		`templates = {`,
		`files_mode = "safe"`,
		`config/foo.conf`,
	} {
		if !strings.Contains(out, frag) {
			t.Fatalf("missing %q in:\n%s", frag, out)
		}
	}
	if strings.Contains(out, "config/old") {
		t.Fatalf("old link should be replaced:\n%s", out)
	}
}

func TestPatchFilesLinks_InsertsMissing(t *testing.T) {
	src := `local packages = require("packages")
return {
  packages = packages,
  policy = { uninstall_mode = "safe" },
}
`
	out, err := PatchFilesLinks(src, []FileLink{
		{Source: "config/nvim", Target: "~/.config/nvim"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `source = "config/nvim"`) {
		t.Fatalf("missing link:\n%s", out)
	}
	if !strings.Contains(out, "files = {") || !strings.Contains(out, "links = {") {
		t.Fatalf("missing files.links:\n%s", out)
	}
}

func TestPatchFilesLinks_EmptyClears(t *testing.T) {
	src := `return {
  files = {
    links = {
      { source = "config/nvim", target = "~/.config/nvim" },
    },
  },
}
`
	out, err := PatchFilesLinks(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "config/nvim") {
		t.Fatalf("expected empty links:\n%s", out)
	}
}
