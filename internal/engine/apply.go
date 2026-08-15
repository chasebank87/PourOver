package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/plan"
)

// Apply runs the reconcile mutation loop for taps, formulae, casks, removes,
// defaults, file links, managed copies, and unlinks. It does not create UI
// sessions or print summaries; frontends pass Progress / Confirmer / writers via opts.
func Apply(ctx context.Context, runner discovery.Runner, p plan.Plan, opts ApplyOptions) (ApplyResult, error) {
	renames := exec.CaskRenameActions(p)
	skipped := exec.UnsupportedApplyActions(p)
	result := ApplyResult{
		Plan:    p,
		Renames: len(renames),
		Skipped: len(skipped),
	}
	if len(p.Actions) == 0 {
		return result, nil
	}

	configDir := opts.ConfigDir
	if configDir == "" && opts.ConfigPath != "" {
		configDir = filepath.Dir(opts.ConfigPath)
	}

	brewOut := opts.Stderr
	if brewOut == nil {
		brewOut = opts.Stdout
	}
	mutRunner := brewRunnerWithProgress(runner, brewOut, opts.Quiet)

	failures := 0
	var progress exec.Progress
	if opts.Progress != nil {
		progress = func(line string) {
			if strings.HasPrefix(line, "failed:") {
				failures++
			}
			opts.Progress(line)
		}
	}

	total := len(p.Actions) - len(skipped) - len(renames)
	phase := func(name string) {
		if opts.OnPhase != nil && total > 0 {
			opts.OnPhase(name)
		}
	}

	var errs []error

	phase("taps")
	taps, err := exec.ApplyTapAdds(ctx, mutRunner, p, progress)
	if err != nil {
		errs = append(errs, err)
	}
	phase("formulae")
	formulae, err := exec.ApplyFormulaInstalls(ctx, mutRunner, p, progress)
	if err != nil {
		errs = append(errs, err)
	}
	phase("casks")
	casks, err := exec.ApplyCaskInstalls(ctx, mutRunner, p, progress)
	if err != nil {
		errs = append(errs, err)
	}

	phase("removes")
	removed, err := exec.ApplyRemoves(ctx, mutRunner, p, opts.Mode, confirmRemoves(opts), progress)
	if err != nil {
		errs = append(errs, err)
	}

	phase("defaults")
	written, err := exec.ApplyDefaultsWrites(ctx, exec.NewExecDefaultsApplier(), p, progress)
	if err != nil {
		errs = append(errs, err)
	}

	phase("links")
	fileOpts := exec.FileApplyOptions{
		ConfigDir:   configDir,
		StateDir:    opts.StateDir,
		FileReplace: opts.FileReplace,
		Now:         opts.Now,
	}
	linked, err := exec.ApplyFileLinks(p, fileOpts, progress)
	if err != nil {
		errs = append(errs, err)
	}

	phase("managed")
	managed, err := exec.ApplyManagedCopies(p, fileOpts, progress)
	if err != nil {
		errs = append(errs, err)
	}

	phase("unlink")
	unlinked, err := exec.ApplyFileUnlinks(p, progress)
	if err != nil {
		errs = append(errs, err)
	}

	result.Taps = taps
	result.Formulae = formulae
	result.Casks = casks
	result.Removed = removed
	result.Defaults = written
	result.Linked = linked
	result.Managed = managed
	result.Unlinked = unlinked
	result.Failures = failures
	return result, errors.Join(errs...)
}

func confirmRemoves(opts ApplyOptions) exec.ConfirmRemoves {
	return func(names []string) bool {
		if opts.AutoYes {
			return true
		}
		if opts.Confirm == nil {
			return false
		}
		prompt := fmt.Sprintf("Uninstall undeclared packages: %s?", strings.Join(names, ", "))
		return opts.Confirm.Confirm(prompt)
	}
}

func brewRunnerWithProgress(runner discovery.Runner, out io.Writer, quiet bool) discovery.Runner {
	if quiet || out == nil {
		return runner
	}
	if er, ok := runner.(*discovery.ExecRunner); ok {
		styled := discovery.NewBrewStyleWriter(out)
		return er.WithOutput(styled, styled)
	}
	return runner
}
