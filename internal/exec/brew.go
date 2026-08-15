package exec

import (
	"context"
	"errors"
	"fmt"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// AddTap runs `brew tap <name>` then `brew trust --tap <name>` for non-official taps
// (Homebrew 6+ requires explicit trust before loading third-party tap code).
func AddTap(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "tap", name); err != nil {
		return fmt.Errorf("brew tap %s: %w", name, err)
	}
	if discovery.NeedsExplicitTrust(name) {
		if err := TrustTap(ctx, runner, name); err != nil {
			return err
		}
	}
	return nil
}

// TrustTap runs `brew trust --tap <name>`.
func TrustTap(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "trust", "--tap", name); err != nil {
		return fmt.Errorf("brew trust --tap %s: %w", name, err)
	}
	return nil
}

// UpdateBrew runs `brew update` so newly tapped formulae/casks are visible to install.
func UpdateBrew(ctx context.Context, runner discovery.Runner) error {
	if _, err := runner.Run(ctx, "update"); err != nil {
		return fmt.Errorf("brew update: %w", err)
	}
	return nil
}

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

// ApplyTapAdds runs brew tap (and trust) for each tap_add action, and brew trust
// for each tap_trust action, in plan order. After any successful tap_add it runs
// `brew update` once so packages from new taps can be installed.
// Failures are collected so later taps still run. Returns successes and joined errors.
func ApplyTapAdds(ctx context.Context, runner discovery.Runner, p plan.Plan, progress Progress) (int, error) {
	n := 0
	added := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionTapAdd:
			report(progress, a)
			if err := AddTap(ctx, runner, a.Name); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
			added++
		case plan.ActionTapTrust:
			report(progress, a)
			if err := TrustTap(ctx, runner, a.Name); err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			n++
		}
	}
	if added > 0 {
		if progress != nil {
			progress("brew update")
		}
		if err := UpdateBrew(ctx, runner); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
		}
	}
	return n, errors.Join(errs...)
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
// Advisory actions like cask_rename are excluded (handled separately).
func UnsupportedApplyActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionTapAdd, plan.ActionTapTrust, plan.ActionTapRemove,
			plan.ActionFormulaInstall, plan.ActionCaskInstall,
			plan.ActionFormulaRemove, plan.ActionCaskRemove,
			plan.ActionLinkCreate, plan.ActionLinkUpdate, plan.ActionLinkReplace,
			plan.ActionManagedCopy, plan.ActionTemplateWrite, plan.ActionFileUnlink, plan.ActionFilePrune,
			plan.ActionDefaultsWrite,
			plan.ActionCaskRename:
			continue
		default:
			out = append(out, a)
		}
	}
	return out
}

// CaskRenameActions returns advisory cask_rename actions from the plan.
func CaskRenameActions(p plan.Plan) []plan.Action {
	var out []plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionCaskRename {
			out = append(out, a)
		}
	}
	return out
}
