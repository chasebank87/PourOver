package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// UpgradeFormula runs `brew upgrade <name>`.
func UpgradeFormula(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "upgrade", name); err != nil {
		return fmt.Errorf("brew upgrade %s: %w", name, err)
	}
	return nil
}

// UpgradeCask runs `brew upgrade --cask <name>`.
func UpgradeCask(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "upgrade", "--cask", name); err != nil {
		return fmt.Errorf("brew upgrade --cask %s: %w", name, err)
	}
	return nil
}

// ApplyUpgrades runs formula_upgrade and cask_upgrade actions in plan order.
// Per-package failures are collected; remaining upgrades still run.
func ApplyUpgrades(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		var err error
		switch a.Type {
		case plan.ActionFormulaUpgrade:
			report(progress, a)
			err = UpgradeFormula(ctx, runner, a.Name)
		case plan.ActionCaskUpgrade:
			report(progress, a)
			err = UpgradeCask(ctx, runner, a.Name)
		default:
			continue
		}
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}
