package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// InstallMas runs `mas install <id>`.
func InstallMas(ctx context.Context, runner discovery.MasRunner, id string) error {
	if _, err := runner.Run(ctx, "install", id); err != nil {
		return fmt.Errorf("mas install %s: %w", id, err)
	}
	return nil
}

// RemoveMas runs `mas uninstall <id>`.
func RemoveMas(ctx context.Context, runner discovery.MasRunner, id string) error {
	if _, err := runner.Run(ctx, "uninstall", id); err != nil {
		return fmt.Errorf("mas uninstall %s: %w", id, err)
	}
	return nil
}

// ApplyMasInstalls runs mas install for each mas_install action in plan order.
// Failures are collected so later installs still run. Returns successes and joined errors.
func ApplyMasInstalls(ctx context.Context, runner discovery.MasRunner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionMasInstall {
			continue
		}
		report(progress, a)
		if err := InstallMas(ctx, runner, a.Value); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}
