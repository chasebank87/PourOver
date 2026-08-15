package plan

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestExpandMasFormulae_AddsMasWhenConfigured(t *testing.T) {
	pkgs := config.Packages{
		Formulae:      []string{"git"},
		MasConfigured: true,
		Mas:           []config.MasApp{{Name: "Xcode", ID: 497799835}},
	}
	got := ExpandMasFormulae(pkgs)
	want := []string{"git", "mas"}
	if !equalStrings(got.Formulae, want) {
		t.Fatalf("Formulae = %v, want %v", got.Formulae, want)
	}
}

func TestExpandMasFormulae_AddsMasWhenManagedEmpty(t *testing.T) {
	// Empty managed set still needs mas for discovery/uninstalls.
	pkgs := config.Packages{
		Formulae:      []string{"git"},
		MasConfigured: true,
	}
	got := ExpandMasFormulae(pkgs)
	want := []string{"git", "mas"}
	if !equalStrings(got.Formulae, want) {
		t.Fatalf("Formulae = %v, want %v", got.Formulae, want)
	}
}

func TestExpandMasFormulae_UnconfiguredUnchanged(t *testing.T) {
	pkgs := config.Packages{Formulae: []string{"git"}}
	got := ExpandMasFormulae(pkgs)
	if !equalStrings(got.Formulae, pkgs.Formulae) {
		t.Fatalf("Formulae = %v, want unchanged %v", got.Formulae, pkgs.Formulae)
	}
}

func TestExpandMasFormulae_SkipsAlreadyDesired(t *testing.T) {
	pkgs := config.Packages{
		Formulae:      []string{"mas", "fzf"},
		MasConfigured: true,
	}
	got := ExpandMasFormulae(pkgs)
	if !equalStrings(got.Formulae, []string{"mas", "fzf"}) {
		t.Fatalf("Formulae = %v, want unchanged order without dupes", got.Formulae)
	}
}

func TestExpandMasFormulae_SkipsCaseInsensitiveDuplicate(t *testing.T) {
	pkgs := config.Packages{
		Formulae:      []string{"Mas", "git"},
		MasConfigured: true,
	}
	got := ExpandMasFormulae(pkgs)
	if !equalStrings(got.Formulae, []string{"Mas", "git"}) {
		t.Fatalf("Formulae = %v, want no case-insensitive duplicate", got.Formulae)
	}
}
