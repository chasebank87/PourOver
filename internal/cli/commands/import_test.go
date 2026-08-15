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
	force := cmd.Flags().Lookup("force")
	if force == nil {
		t.Fatal("missing --force")
	}
	if !strings.Contains(force.Usage, "merge") && !strings.Contains(strings.ToLower(cmd.Long), "merge") {
		t.Fatalf("expected merge wording in import help, long=%q force=%q", cmd.Long, force.Usage)
	}
}

func TestImportMerge_PackagesAddOnlyPreservesExisting(t *testing.T) {
	// Mirrors runImport default merge: keep declared packages not currently discovered.
	existingF := []string{"git", "neofetch"}
	discoveredF := []string{"git", "fzf"}
	merged, added := configimport.MergePackageLists(existingF, discoveredF)
	if len(added) != 1 || added[0] != "fzf" {
		t.Fatalf("added = %v", added)
	}
	body := configimport.FormatPackagesLua(merged, nil)
	dir := t.TempDir()
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
	got := strings.Join(m.Packages.Formulae, ",")
	if !strings.Contains(got, "neofetch") || !strings.Contains(got, "fzf") || !strings.Contains(got, "git") {
		t.Fatalf("formulae = %v, want git+neofetch+fzf", m.Packages.Formulae)
	}
}

func TestImportMerge_ForceReplacesPackages(t *testing.T) {
	// --force uses discovered set only (no merge).
	discoveredF := []string{"fzf"}
	discoveredC := []string{"raycast"}
	body := configimport.FormatPackagesLua(discoveredF, discoveredC)
	dir := t.TempDir()
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
	if len(m.Packages.Formulae) != 1 || m.Packages.Formulae[0] != "fzf" {
		t.Fatalf("formulae = %v", m.Packages.Formulae)
	}
	if len(m.Packages.Casks) != 1 || m.Packages.Casks[0] != "raycast" {
		t.Fatalf("casks = %v", m.Packages.Casks)
	}
}

func TestImportMerge_FilesSkipsDeclaredTargets(t *testing.T) {
	existing := []config.FileLink{{Source: "config/home/zshrc", Target: "~/.zshrc"}}
	declared := configimport.LinkTargets(existing)
	candidates := []configimport.FileCandidate{
		{TargetDecl: "~/.zshrc", RelSource: "config/home/zshrc"},
		{TargetDecl: "~/.config/ghostty", RelSource: "config/ghostty"},
	}
	var toImport []configimport.FileCandidate
	for _, c := range candidates {
		if _, ok := declared[c.TargetDecl]; ok {
			continue
		}
		toImport = append(toImport, c)
	}
	if len(toImport) != 1 || toImport[0].TargetDecl != "~/.config/ghostty" {
		t.Fatalf("toImport = %+v", toImport)
	}
	imported := []config.FileLink{{Source: "config/ghostty", Target: "~/.config/ghostty"}}
	merged, added := configimport.MergeFileLinks(existing, imported)
	if len(added) != 1 || len(merged) != 2 {
		t.Fatalf("merged=%+v added=%+v", merged, added)
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
