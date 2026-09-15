package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// ApplyAutoUpdate runs brew_autoupdate_delete then brew_autoupdate_start in plan order.
func ApplyAutoUpdate(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionBrewAutoupdateDelete:
			report(progress, a)
			if err := DeleteBrewAutoupdate(ctx, runner); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
		case plan.ActionBrewAutoupdateStart:
			report(progress, a)
			if err := StartBrewAutoupdate(ctx, runner, a); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
		}
	}
	return n, errors.Join(errs...)
}

// DeleteBrewAutoupdate runs `brew autoupdate delete`.
func DeleteBrewAutoupdate(ctx context.Context, runner discovery.Runner) error {
	if _, err := runner.Run(ctx, "autoupdate", "delete"); err != nil {
		return fmt.Errorf("brew autoupdate delete: %w", err)
	}
	return nil
}

// StartBrewAutoupdate runs `brew autoupdate start` with the planned interval and flags.
func StartBrewAutoupdate(ctx context.Context, runner discovery.Runner, a plan.Action) error {
	args := []string{"autoupdate", "start"}
	if name := strings.TrimSpace(a.Name); name != "" {
		args = append(args, name)
	}
	if flags := strings.TrimSpace(a.Value); flags != "" {
		args = append(args, strings.Fields(flags)...)
	}
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("brew %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
