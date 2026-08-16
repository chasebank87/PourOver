package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

type applyOptions struct {
	mode         config.UninstallMode
	autoYes      bool
	quiet        bool
	configPath   string
	configDir    string
	stateDir     string
	manifest     config.Manifest
	generationID string
	now          func() time.Time
}

func runApply(cmd *cobra.Command, dryRun, autoYes, quiet bool) error {
	configPath, verbose, asJSON, err := planDisplayOptions(cmd)
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s (run `pourover init` to scaffold)", configPath)
		}
		return fmt.Errorf("config file: %w", err)
	}

	if verbose {
		ui.Mutedf(cmd.ErrOrStderr(), "using config %s", configPath)
	}

	runner := discovery.NewExecRunner()
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	res, err := engine.BuildPlanResult(cmd.Context(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, stateDir, time.Now())
	if err != nil {
		return err
	}

	if dryRun {
		return printPlan(res.Plan, asJSON)
	}

	opts := applyOptions{
		mode:         policy.ResolveModeFromManifest(res.Manifest),
		autoYes:      autoYes,
		quiet:        quiet,
		configPath:   configPath,
		configDir:    filepath.Dir(configPath),
		stateDir:     stateDir,
		manifest:     res.Manifest,
		generationID: res.GenerationID,
		now:          time.Now,
	}
	return executeApply(cmd, runner, res.Plan, opts)
}

func executeApply(cmd *cobra.Command, runner discovery.Runner, p plan.Plan, opts applyOptions) error {
	result, applyErr := runApplyActions(cmd, runner, p, opts)
	if err := engine.FinalizeApply(engine.FinalizeOptions{
		StateDir:             opts.stateDir,
		ConfigDir:            opts.configDir,
		Manifest:             opts.manifest,
		GenerationID:         opts.generationID,
		PrunedPaths:          result.PrunedPaths,
		SucceededFileTargets: result.SucceededFileTargets(),
		UnlinkedPaths:        result.UnlinkedPaths,
		Now:                  opts.now,
	}, p, applyErr); err != nil {
		return err
	}
	maybeAutoPushConfig(cmd, opts.configPath, opts.manifest)
	return nil
}

func runApplyActions(cmd *cobra.Command, runner discovery.Runner, p plan.Plan, opts applyOptions) (engine.ApplyResult, error) {
	out := cmd.ErrOrStderr()
	if len(p.Actions) == 0 {
		if ui.Enabled(out, opts.quiet) {
			ui.Header(out, "apply")
			ui.Mutedf(out, "☕ No changes.")
		} else {
			fmt.Fprintln(out, "No changes.")
		}
		return engine.ApplyResult{Plan: p}, nil
	}

	renames := exec.CaskRenameActions(p)
	skipped := exec.UnsupportedApplyActions(p)
	total := len(p.Actions) - len(skipped) - len(renames)

	var session *ui.Session
	var progress engine.Progress
	var brewOut io.Writer = out
	fancy := ui.Enabled(out, opts.quiet)
	if fancy {
		session = ui.NewSession(out, "apply")
		session.Start(total)
		if total > 0 {
			progress = session.ProgressAdapter()
			brewOut = session
		}
	} else {
		if p := applyProgress(out, opts.quiet); p != nil {
			progress = engine.Progress(p)
		}
	}

	engineOpts := engine.ApplyOptions{
		ConfigPath:   opts.configPath,
		ConfigDir:    opts.configDir,
		StateDir:     opts.stateDir,
		GenerationID: opts.generationID,
		Mode:         opts.mode,
		FileReplace:  policy.ResolveFileReplaceFromManifest(opts.manifest),
		FilesMode:    policy.ResolveFilesModeFromManifest(opts.manifest),
		AutoYes:      opts.autoYes,
		Quiet:        opts.quiet,
		Progress:     progress,
		Confirm:      stdinConfirmer{in: cmd.InOrStdin(), out: out},
		Stdout:       brewOut,
		Stderr:       brewOut,
		Now:          opts.now,
	}
	if session != nil && total > 0 {
		engineOpts.OnPhase = session.SetPhase
		engineOpts.BeforeAuth = session.PrepareAuth
		engineOpts.BeforePrompt = session.PreparePrompt
	}

	result, err := engine.Apply(cmd.Context(), runner, p, engineOpts)

	n := result.Taps + result.Formulae + result.Casks + result.Mas + result.Removed + result.Defaults +
		result.Linked + result.Managed + result.Templates + result.Unlinked + result.Pruned

	sum := ui.Summary{
		Taps:      result.Taps,
		Formulae:  result.Formulae,
		Casks:     result.Casks,
		Mas:       result.Mas,
		Removed:   result.Removed,
		Defaults:  result.Defaults,
		Linked:    result.Linked,
		Managed:   result.Managed,
		Templates: result.Templates,
		Unlinked:  result.Unlinked,
		Pruned:    result.Pruned,
		Renames:   result.Renames,
		Skipped:   result.Skipped,
		Failures:  result.Failures,
	}

	if session != nil {
		session.Finish(sum)
		printCaskRenameAdvice(out, renames)
		printUnsupportedActions(out, skipped)
	} else {
		if n > 0 || result.Failures > 0 || result.Skipped > 0 {
			ui.WriteSummary(out, sum, fancy)
		} else if n == 0 && len(skipped) == 0 && len(renames) == 0 && err == nil {
			ui.WriteSummary(out, ui.Summary{}, fancy)
		}
		printCaskRenameAdvice(out, renames)
		printUnsupportedActions(out, skipped)
	}
	return result, err
}

type stdinConfirmer struct {
	in  io.Reader
	out io.Writer
}

func (c stdinConfirmer) Confirm(prompt string) bool {
	return exec.ConfirmYes(c.in, c.out, prompt)
}

func printCaskRenameAdvice(out io.Writer, renames []plan.Action) {
	if len(renames) == 0 {
		return
	}
	fancy := ui.Enabled(out, false)
	for _, a := range renames {
		line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
		line = "  " + line
		if fancy {
			line = ui.Warning().Render(line)
		}
		fmt.Fprintln(out, line)
	}
	note := fmt.Sprintf("☕ Update packages.lua to use the new cask names (%d).", len(renames))
	if fancy {
		ui.Warnf(out, "%s", note)
		return
	}
	fmt.Fprintln(out, note)
}

func printUnsupportedActions(out io.Writer, skipped []plan.Action) {
	fancy := ui.Enabled(out, false)
	for _, a := range skipped {
		line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
		line = "  " + line
		if fancy {
			line = ui.Warning().Render(line)
		}
		fmt.Fprintln(out, line)
	}
}

func applyProgress(out io.Writer, quiet bool) exec.Progress {
	if quiet {
		return nil
	}
	return func(line string) {
		fmt.Fprintf(out, "☕ %s\n", line)
	}
}

func brewRunnerWithProgress(runner discovery.Runner, out io.Writer, quiet bool) discovery.Runner {
	if quiet {
		return runner
	}
	if er, ok := runner.(*discovery.ExecRunner); ok {
		styled := discovery.NewBrewStyleWriter(out)
		return er.WithOutput(styled, styled)
	}
	return runner
}
