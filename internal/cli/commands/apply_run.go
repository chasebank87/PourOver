package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/state"
	"github.com/spf13/cobra"
)

type applyOptions struct {
	mode      config.UninstallMode
	autoYes   bool
	configDir string
	stateDir  string
	manifest  config.Manifest
	now       func() time.Time
}

func runApply(cmd *cobra.Command, dryRun, autoYes bool) error {
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
		fmt.Fprintf(os.Stderr, "using config %s\n", configPath)
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
		mode:      policy.ResolveModeFromManifest(manifest),
		autoYes:   autoYes,
		configDir: filepath.Dir(configPath),
		stateDir:  stateDir,
		manifest:  manifest,
		now:       time.Now,
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
	return persistApplyState(opts, p)
}

func runApplyActions(cmd *cobra.Command, runner discovery.Runner, p plan.Plan, opts applyOptions) error {
	out := cmd.ErrOrStderr()
	skipped := exec.UnsupportedApplyActions(p)
	if len(p.Actions) == 0 {
		fmt.Fprintln(out, "No changes.")
		return nil
	}

	// Phase 1: Homebrew (installs then removes)
	formulae, err := exec.ApplyFormulaInstalls(cmd.Context(), runner, p)
	if err != nil {
		return err
	}
	casks, err := exec.ApplyCaskInstalls(cmd.Context(), runner, p)
	if err != nil {
		return err
	}

	confirm := removeConfirmer(cmd, opts.autoYes)
	removed, err := exec.ApplyRemoves(cmd.Context(), runner, p, opts.mode, confirm)
	if err != nil {
		return err
	}

	// Phase 2: file links
	linked, err := exec.ApplyFileLinks(p, opts.configDir)
	if err != nil {
		return err
	}

	n := formulae + casks + removed + linked

	if formulae > 0 {
		fmt.Fprintf(out, "Installed %d formula(s).\n", formulae)
	}
	if casks > 0 {
		fmt.Fprintf(out, "Installed %d cask(s).\n", casks)
	}
	if removed > 0 {
		fmt.Fprintf(out, "Removed %d package(s).\n", removed)
	}
	if linked > 0 {
		fmt.Fprintf(out, "Updated %d file link(s).\n", linked)
	}
	if len(skipped) > 0 {
		fmt.Fprintf(out, "Skipped %d action(s) not yet supported by apply:\n", len(skipped))
		for _, a := range skipped {
			line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
			fmt.Fprintf(out, "  %s\n", line)
		}
	}
	if n == 0 && len(skipped) == 0 {
		fmt.Fprintln(out, "No changes.")
	} else if n == 0 && len(skipped) == len(p.Actions) {
		fmt.Fprintln(out, "No actions to apply.")
	}
	return nil
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

func removeConfirmer(cmd *cobra.Command, autoYes bool) exec.ConfirmRemoves {
	return func(names []string) bool {
		if autoYes {
			return true
		}
		prompt := fmt.Sprintf("Uninstall undeclared packages: %s?", strings.Join(names, ", "))
		return exec.ConfirmYes(cmd.InOrStdin(), cmd.ErrOrStderr(), prompt)
	}
}
