package exec

import (
	"context"
	"fmt"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// InstallFormula runs `brew install <name>`.
func InstallFormula(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "install", name); err != nil {
		return fmt.Errorf("brew install %s: %w", name, err)
	}
	return nil
}

// ApplyFormulaInstalls runs brew install for each formula_install action in plan order.
// Other action types are left to later milestones. Returns how many installs ran.
func ApplyFormulaInstalls(ctx context.Context, runner discovery.Runner, p plan.Plan) (int, error) {
	n := 0
	for _, a := range p.Actions {
		if a.Type != plan.ActionFormulaInstall {
			continue
		}
		if err := InstallFormula(ctx, runner, a.Name); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// UnsupportedApplyActions returns plan actions that ApplyFormulaInstalls does not run.
func UnsupportedApplyActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		if a.Type != plan.ActionFormulaInstall {
			out = append(out, a)
		}
	}
	return out
}
