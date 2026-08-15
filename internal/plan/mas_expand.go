package plan

import "github.com/chasebank87/PourOver/internal/config"

const masFormula = "mas"

// ExpandMasFormulae appends the mas formula to desired formulae when
// packages.mas is managed (MasConfigured) and mas is not already listed.
// Call before BuildBrewPlan so the implied package participates in brew reconcile.
// Empty managed sets still expand so discovery/apply can run uninstalls.
func ExpandMasFormulae(pkgs config.Packages) config.Packages {
	if !pkgs.MasConfigured {
		return pkgs
	}
	out := pkgs
	out.Formulae = append([]string(nil), pkgs.Formulae...)
	have := sliceSet(out.Formulae)
	if _, ok := have[brewToken(masFormula)]; !ok {
		out.Formulae = append(out.Formulae, masFormula)
	}
	return out
}
