package engine

import "github.com/chasebank87/PourOver/internal/plan"

// ApplyResult summarizes the outcome of a reconcile apply run.
type ApplyResult struct {
	Plan                                                                                             plan.Plan
	Taps, Formulae, Casks, Mas, PAM, Removed, Defaults, Linked, Managed, Templates, Unlinked, Pruned int
	PrunedPaths                                                                                      []string // absolute paths successfully pruned
	Renames, Skipped, Failures                                                                       int
}
