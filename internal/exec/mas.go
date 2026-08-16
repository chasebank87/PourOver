package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// InstallMas runs `mas install <id>`, falling back to `mas get <id>` on failure.
func InstallMas(ctx context.Context, runner discovery.MasRunner, id string) error {
	_, installErr := runner.Run(ctx, "install", id)
	if installErr == nil {
		return nil
	}
	_, getErr := runner.Run(ctx, "get", id)
	if getErr == nil {
		return nil
	}
	return fmt.Errorf("mas get %s: %w (after mas install failed: %v); sign in to the App Store (Media & Purchases) and retry. First-time apps must be gotten onto this Apple Account. Open the page with: mas open %s", id, getErr, installErr, id)
}

// RemoveMas runs `mas uninstall <id>`.
func RemoveMas(ctx context.Context, runner discovery.MasRunner, id string) error {
	return RemoveMasApps(ctx, runner, []string{id})
}

// RemoveMasApps runs `mas uninstall` for all App Store IDs in one invocation.
func RemoveMasApps(ctx context.Context, runner discovery.MasRunner, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	args := append([]string{"uninstall"}, ids...)
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("mas uninstall %s: %w", strings.Join(ids, " "), err)
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
