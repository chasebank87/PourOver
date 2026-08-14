package configimport

import (
	"strings"
	"testing"
)

func TestFormatPackagesLua(t *testing.T) {
	got := FormatPackagesLua([]string{"fzf", "git"}, []string{"raycast"})
	for _, frag := range []string{`"fzf"`, `"git"`, `"raycast"`, "formulae = {", "casks = {"} {
		if !strings.Contains(got, frag) {
			t.Fatalf("missing %q in:\n%s", frag, got)
		}
	}
	if strings.Index(got, `"fzf"`) > strings.Index(got, `"git"`) {
		t.Fatal("formulae not sorted")
	}
}
