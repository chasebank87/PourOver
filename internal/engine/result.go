package engine

import "github.com/chasebank87/PourOver/internal/plan"

// ApplyResult summarizes the outcome of a reconcile apply run.
type ApplyResult struct {
	Plan                                                                                             plan.Plan
	Taps, Formulae, Casks, Mas, PAM, Removed, Defaults, Linked, Managed, Templates, Unlinked, Pruned int
	PrunedPaths                                                                                      []string // absolute paths successfully pruned
	LinkedPaths                                                                                      []string // absolute paths successfully link-activated
	ManagedPaths                                                                                     []string // absolute paths successfully managed-copied
	TemplatePaths                                                                                    []string // absolute paths successfully template-written
	UnlinkedPaths                                                                                    []string // absolute paths successfully unlinked
	Renames, Skipped, Failures                                                                       int
}

// SucceededFileTargets returns absolute paths written by link/managed/template phases.
func (r ApplyResult) SucceededFileTargets() []string {
	out := make([]string, 0, len(r.LinkedPaths)+len(r.ManagedPaths)+len(r.TemplatePaths))
	out = append(out, r.LinkedPaths...)
	out = append(out, r.ManagedPaths...)
	out = append(out, r.TemplatePaths...)
	return out
}
