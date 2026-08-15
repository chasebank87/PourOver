package exec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// ConfirmRemoves asks whether to proceed with uninstalling the named packages.
// names are package names only (no action type prefix). For mas_remove, names
// are formatted as "Name (id)".
type ConfirmRemoves func(names []string) bool

// RemoveFormula runs `brew uninstall <name>`.
func RemoveFormula(ctx context.Context, runner discovery.Runner, name string) error {
	return RemoveFormulae(ctx, runner, []string{name})
}

// RemoveFormulae runs `brew uninstall` for all names in one invocation.
func RemoveFormulae(ctx context.Context, runner discovery.Runner, names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"uninstall"}, names...)
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("brew uninstall %s: %w", strings.Join(names, " "), err)
	}
	return nil
}

// RemoveCask runs `brew uninstall --cask <name>`.
func RemoveCask(ctx context.Context, runner discovery.Runner, name string) error {
	return RemoveCasks(ctx, runner, []string{name})
}

// RemoveCasks runs `brew uninstall --cask` for all names in one invocation so
// Homebrew only elevates once for the batch.
func RemoveCasks(ctx context.Context, runner discovery.Runner, names []string) error {
	if len(names) == 0 {
		return nil
	}
	args := append([]string{"uninstall", "--cask"}, names...)
	if _, err := runner.Run(ctx, args...); err != nil {
		return fmt.Errorf("brew uninstall --cask %s: %w", strings.Join(names, " "), err)
	}
	return nil
}

// RemoveTap runs `brew untap <name>`.
func RemoveTap(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "untap", name); err != nil {
		return fmt.Errorf("brew untap %s: %w", name, err)
	}
	return nil
}

// ApplyRemoves runs formula/cask/tap/mas remove actions according to uninstall mode.
//
//   - non_destructive: skip all removes (no prompt)
//   - strict: remove without prompting
//   - safe: prompt once via confirm; if declined, skip all removes
//
// Formula/cask/mas uninstalls are batched into one brew/mas invocation each so
// admin elevation is requested once per tool, not once per package. beforeAuth
// parks fancy UI before the one-time `sudo -v` when cask or mas removes need it.
//
// confirm may be nil when mode is not safe (or when there are no removes).
// masRunner may be nil when there are no mas_remove actions; otherwise nil
// resolves to discovery.NewExecMasRunner().
func ApplyRemoves(ctx context.Context, runner discovery.Runner, masRunner discovery.MasRunner, p plan.Plan, mode config.UninstallMode, confirm ConfirmRemoves, beforeAuth func(), progress Progress) (int, error) {
	var removes []plan.Action
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionFormulaRemove, plan.ActionCaskRemove, plan.ActionTapRemove, plan.ActionMasRemove:
			removes = append(removes, a)
		}
	}
	if len(removes) == 0 {
		return 0, nil
	}

	switch mode {
	case config.UninstallModeNonDestructive:
		return 0, nil
	case config.UninstallModeSafe:
		names := removeConfirmNames(removes)
		if confirm == nil || !confirm(names) {
			return 0, nil
		}
	case config.UninstallModeStrict:
		// no prompt
	default:
		// treat unknown like safe decline if no confirm
		names := removeConfirmNames(removes)
		if confirm == nil || !confirm(names) {
			return 0, nil
		}
	}

	var formulae, casks, taps, mas []plan.Action
	for _, a := range removes {
		switch a.Type {
		case plan.ActionFormulaRemove:
			formulae = append(formulae, a)
		case plan.ActionCaskRemove:
			casks = append(casks, a)
		case plan.ActionTapRemove:
			taps = append(taps, a)
		case plan.ActionMasRemove:
			mas = append(mas, a)
		}
	}

	if len(casks) > 0 || len(mas) > 0 {
		if err := EnsureSudo(ctx, beforeAuth); err != nil {
			return 0, err
		}
	}

	if len(mas) > 0 && masRunner == nil {
		masRunner = discovery.NewExecMasRunner()
	}

	n := 0
	var errs []error

	if len(formulae) > 0 {
		for _, a := range formulae {
			report(progress, a)
		}
		names := actionNames(formulae)
		if err := RemoveFormulae(ctx, runner, names); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
		} else {
			n += len(formulae)
		}
	}
	if len(casks) > 0 {
		for _, a := range casks {
			report(progress, a)
		}
		names := actionNames(casks)
		if err := RemoveCasks(ctx, runner, names); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
		} else {
			n += len(casks)
		}
	}
	for _, a := range taps {
		report(progress, a)
		if err := RemoveTap(ctx, runner, a.Name); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	if len(mas) > 0 {
		for _, a := range mas {
			report(progress, a)
		}
		ids := make([]string, len(mas))
		for i, a := range mas {
			ids[i] = a.Value
		}
		if err := RemoveMasApps(ctx, masRunner, ids); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
		} else {
			n += len(mas)
		}
	}

	return n, errors.Join(errs...)
}

func actionNames(actions []plan.Action) []string {
	names := make([]string, len(actions))
	for i, a := range actions {
		names[i] = a.Name
	}
	return names
}

func removeConfirmNames(removes []plan.Action) []string {
	names := make([]string, len(removes))
	for i, a := range removes {
		if a.Type == plan.ActionMasRemove {
			names[i] = fmt.Sprintf("%s (%s)", a.Name, a.Value)
			continue
		}
		names[i] = a.Name
	}
	return names
}
