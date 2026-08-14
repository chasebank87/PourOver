package exec

import (
	"context"
	"errors"
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

// InstallCask runs `brew install --cask <name>`.
func InstallCask(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "install", "--cask", name); err != nil {
		return fmt.Errorf("brew install --cask %s: %w", name, err)
	}
	return nil
}

// ApplyFormulaInstalls runs brew install for each formula_install action in plan order.
// A failure for one formula is recorded and the remaining installs still run (nix-darwin-style).
// Returns how many installs succeeded; error is errors.Join of any failures.
func ApplyFormulaInstalls(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionFormulaInstall {
			continue
		}
		report(progress, a)
		if err := InstallFormula(ctx, runner, a.Name); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// ApplyCaskInstalls runs brew install --cask for each cask_install action in plan order.
// Failures are collected so later casks still install. Returns successes and joined errors.
func ApplyCaskInstalls(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionCaskInstall {
			continue
		}
		report(progress, a)
		if err := InstallCask(ctx, runner, a.Name); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// UnsupportedApplyActions returns plan actions that apply does not run yet.
// v1 apply supports all current brew and file-link action types.
func UnsupportedApplyActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionFormulaInstall, plan.ActionCaskInstall,
			plan.ActionFormulaRemove, plan.ActionCaskRemove,
			plan.ActionLinkCreate, plan.ActionLinkUpdate,
			plan.ActionDefaultsWrite:
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}
