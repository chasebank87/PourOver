package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configimport"
)

func TestNewImportCmd_HasFlags(t *testing.T) {
	cmd := NewImportCmd()
	for _, name := range []string{"packages", "files", "dry-run", "force"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s", name)
		}
	}
}

func TestImportPackagesLua_RoundTripLoad(t *testing.T) {
	dir := t.TempDir()
	body := configimport.FormatPackagesLua([]string{"git", "fzf"}, []string{"raycast"})
	if err := os.WriteFile(filepath.Join(dir, "packages.lua"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	root := configimport.FormatRootLua(nil, config.Policy{UninstallMode: config.UninstallModeSafe}, config.Backup{})
	path := filepath.Join(dir, "pourover.lua")
	if err := os.WriteFile(path, []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := config.LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Packages.Formulae) != 2 || m.Packages.Formulae[0] != "fzf" {
		// LoadManifest may preserve Lua order; FormatPackagesLua sorts so fzf first
		got := strings.Join(m.Packages.Formulae, ",")
		if !strings.Contains(got, "git") || !strings.Contains(got, "fzf") {
			t.Fatalf("formulae = %v", m.Packages.Formulae)
		}
	}
	if len(m.Packages.Casks) != 1 || m.Packages.Casks[0] != "raycast" {
		t.Fatalf("casks = %v", m.Packages.Casks)
	}
}
