package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/state"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

type applyOptions struct {
	mode       config.UninstallMode
	autoYes    bool
	quiet      bool
	configPath string
	configDir  string
	stateDir   string
	manifest   config.Manifest
	now        func() time.Time
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
		fmt.Fprintf(cmd.ErrOrStderr(), "using config %s\n", configPath)
	}

	runner := discovery.NewExecRunner()
	p, err := buildPlan(cmd.Context(), configPath, runner)
	if err != nil {
		return err
	}

	if dryRun {
		return printPlan(p, asJSON)
	}

	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	opts := applyOptions{
		mode:       policy.ResolveModeFromManifest(manifest),
		autoYes:    autoYes,
		quiet:      quiet,
		configPath: configPath,
		configDir:  filepath.Dir(configPath),
		stateDir:   stateDir,
		manifest:   manifest,
		now:        time.Now,
	}
	return executeApply(cmd, runner, p, opts)
}

func executeApply(cmd *cobra.Command, runner discovery.Runner, p plan.Plan, opts applyOptions) error {
	applyErr := runApplyActions(cmd, runner, p, opts)
	if histErr := appendApplyHistory(opts, p, applyErr); histErr != nil && applyErr == nil {
		return histErr
	}
	if applyErr != nil {
		return applyErr
	}
	if err := persistApplyState(opts, p); err != nil {
		return err
	}
	maybeAutoPushConfig(cmd, opts.configPath, opts.manifest)
	return nil
}

func runApplyActions(cmd *cobra.Command, runner discovery.Runner, p plan.Plan, opts applyOptions) error {
	out := cmd.ErrOrStderr()
	if len(p.Actions) == 0 {
		fmt.Fprintln(out, "No changes.")
		return nil
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
		ConfigPath: opts.configPath,
		ConfigDir:  opts.configDir,
		Mode:       opts.mode,
		AutoYes:    opts.autoYes,
		Quiet:      opts.quiet,
		Progress:   progress,
		Confirm:    stdinConfirmer{in: cmd.InOrStdin(), out: out},
		Stdout:     brewOut,
		Stderr:     brewOut,
	}
	if session != nil && total > 0 {
		engineOpts.OnPhase = session.SetPhase
	}

	result, err := engine.Apply(cmd.Context(), runner, p, engineOpts)

	n := result.Taps + result.Formulae + result.Casks + result.Removed + result.Defaults + result.Linked

	if session != nil {
		session.Finish(ui.Summary{
			Taps:     result.Taps,
			Formulae: result.Formulae,
			Casks:    result.Casks,
			Removed:  result.Removed,
			Defaults: result.Defaults,
			Linked:   result.Linked,
			Renames:  result.Renames,
			Skipped:  result.Skipped,
			Failures: result.Failures,
		})
		printCaskRenameAdvice(out, renames)
		printUnsupportedActions(out, skipped)
	} else {
		if result.Taps > 0 {
			fmt.Fprintf(out, "Added %d tap(s).\n", result.Taps)
		}
		if result.Formulae > 0 {
			fmt.Fprintf(out, "Installed %d formula(s).\n", result.Formulae)
		}
		if result.Casks > 0 {
			fmt.Fprintf(out, "Installed %d cask(s).\n", result.Casks)
		}
		if result.Removed > 0 {
			fmt.Fprintf(out, "Removed %d package(s).\n", result.Removed)
		}
		if result.Defaults > 0 {
			fmt.Fprintf(out, "Updated %d macOS default(s).\n", result.Defaults)
		}
		if result.Linked > 0 {
			fmt.Fprintf(out, "Updated %d file link(s).\n", result.Linked)
		}
		printCaskRenameAdvice(out, renames)
		if len(skipped) > 0 {
			fmt.Fprintf(out, "Skipped %d action(s) not yet supported by apply:\n", len(skipped))
			printUnsupportedActions(out, skipped)
		}
		if n == 0 && len(skipped) == 0 && len(renames) == 0 && err == nil {
			fmt.Fprintln(out, "No changes.")
		} else if n == 0 && len(skipped)+len(renames) == len(p.Actions) && len(renames) == 0 {
			fmt.Fprintln(out, "No actions to apply.")
		}
	}
	return err
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
	for _, a := range renames {
		line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprintf(out, "☕ Update packages.lua to use the new cask names (%d).\n", len(renames))
}

func printUnsupportedActions(out io.Writer, skipped []plan.Action) {
	for _, a := range skipped {
		line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
		fmt.Fprintf(out, "  %s\n", line)
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

func appendApplyHistory(opts applyOptions, p plan.Plan, applyErr error) error {
	if opts.stateDir == "" {
		return nil
	}
	now := time.Now
	if opts.now != nil {
		now = opts.now
	}
	at := now()
	hash, err := state.ManifestHash(opts.manifest)
	if err != nil {
		return fmt.Errorf("history manifest hash: %w", err)
	}
	entry := state.NewHistoryEntry(p, hash, at, applyErr)
	if _, err := state.AppendHistory(opts.stateDir, entry, at); err != nil {
		return fmt.Errorf("persist history: %w", err)
	}
	return nil
}

func persistApplyState(opts applyOptions, p plan.Plan) error {
	if opts.stateDir == "" {
		return nil
	}
	now := time.Now
	if opts.now != nil {
		now = opts.now
	}
	at := now()
	if err := state.PersistApplyState(opts.stateDir, opts.manifest, p, at); err != nil {
		return fmt.Errorf("persist state: %w", err)
	}
	result, err := backup.SnapshotAndMirror(opts.stateDir, opts.manifest, at)
	if err != nil {
		return fmt.Errorf("snapshot/mirror: %w", err)
	}
	_ = result
	return nil
}
