package exec

import (
	"context"
	"errors"
	"fmt"

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
	if _, err := runner.Run(ctx, "uninstall", name); err != nil {
		return fmt.Errorf("brew uninstall %s: %w", name, err)
	}
	return nil
}

// RemoveCask runs `brew uninstall --cask <name>`.
func RemoveCask(ctx context.Context, runner discovery.Runner, name string) error {
	if _, err := runner.Run(ctx, "uninstall", "--cask", name); err != nil {
		return fmt.Errorf("brew uninstall --cask %s: %w", name, err)
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
// confirm may be nil when mode is not safe (or when there are no removes).
// masRunner may be nil when there are no mas_remove actions; otherwise nil
// resolves to discovery.NewExecMasRunner().
func ApplyRemoves(ctx context.Context, runner discovery.Runner, masRunner discovery.MasRunner, p plan.Plan, mode config.UninstallMode, confirm ConfirmRemoves, progress Progress) (int, error) {
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

	needMas := false
	for _, a := range removes {
		if a.Type == plan.ActionMasRemove {
			needMas = true
			break
		}
	}
	if needMas && masRunner == nil {
		masRunner = discovery.NewExecMasRunner()
	}

	n := 0
	var errs []error
	for _, a := range removes {
		report(progress, a)
		var err error
		switch a.Type {
		case plan.ActionFormulaRemove:
			err = RemoveFormula(ctx, runner, a.Name)
		case plan.ActionCaskRemove:
			err = RemoveCask(ctx, runner, a.Name)
		case plan.ActionTapRemove:
			err = RemoveTap(ctx, runner, a.Name)
		case plan.ActionMasRemove:
			err = RemoveMas(ctx, masRunner, a.Value)
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
