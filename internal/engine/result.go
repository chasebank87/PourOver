package engine

import "github.com/chasebank87/PourOver/internal/plan"

// ApplyResult summarizes the outcome of a reconcile apply run.
type ApplyResult struct {
	Plan                                                                plan.Plan
	Taps, Formulae, Casks, Removed, Defaults, Linked, Managed, Unlinked int
	Renames, Skipped, Failures                                          int
}
