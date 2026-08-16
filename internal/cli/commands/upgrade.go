package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewUpgradeCmd returns the upgrade subcommand.
func NewUpgradeCmd() *cobra.Command {
	var dryRun bool
	var autoYes bool
	var skipSelf bool
	var quiet bool
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update pourover, upgrade declared packages, then reapply",
		Long: `Upgrade first self-updates the pourover binary from GitHub Releases
(like brew update), then upgrades each declared formula/cask (brew) and
Mac App Store app (mas) that is installed and outdated, then rebuilds the
apply plan and reconciles.
Use --dry-run to preview package upgrade and apply actions (skips self-update).
Use --skip-self-update to only upgrade packages.
By default each action is printed as it runs on interactive terminals with a
progress bar; use --quiet for summary-only output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd, dryRun, autoYes, skipSelf, quiet)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview upgrade and apply actions without modifying the system")
	cmd.Flags().BoolVar(&autoYes, "yes", false, "skip confirmation prompts during the apply phase")
	cmd.Flags().BoolVar(&skipSelf, "skip-self-update", false, "do not self-update the pourover binary")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress per-action progress output")
	return cmd
}

func runUpgrade(cmd *cobra.Command, dryRun, autoYes, skipSelf, quiet bool) error {
	if !dryRun && !skipSelf {
		if err := runSelfUpdate(cmd, nil); err != nil {
			return fmt.Errorf("self-update: %w", err)
		}
	}

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
	if _, err := config.LoadManifest(configPath); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	upgradePlan, err := engine.BuildUpgradePlan(cmd.Context(), configPath, runner, nil)
	if err != nil {
		return err
	}

	if dryRun {
		applyPlan, err := buildPlan(cmd.Context(), configPath, runner)
		if err != nil {
			return err
		}
		combined := plan.MergePlans(upgradePlan, applyPlan)
		return printPlan(combined, asJSON)
	}

	out := cmd.ErrOrStderr()
	var session *ui.Session
	var progress engine.Progress
	var brewOut io.Writer = out
	fancy := ui.Enabled(out, quiet) && len(upgradePlan.Actions) > 0
	if fancy {
		session = ui.NewSession(out, "upgrade")
		session.Start(len(upgradePlan.Actions))
		session.SetPhase("packages")
		progress = session.ProgressAdapter()
		brewOut = session
	} else {
		if p := applyProgress(out, quiet); p != nil {
			progress = engine.Progress(p)
		}
	}

	var upgradeErr error
	if len(upgradePlan.Actions) == 0 {
		if ui.Enabled(out, quiet) {
			ui.Mutedf(out, "☕ No package upgrades.")
		} else {
			fmt.Fprintln(out, "No package upgrades.")
		}
	} else {
		result, err := engine.UpgradePackages(cmd.Context(), runner, upgradePlan, engine.UpgradeOptions{
			Quiet:    quiet,
			Progress: progress,
			Stdout:   brewOut,
			Stderr:   brewOut,
		})
		upgradeErr = err
		if session != nil {
			failures := result.Failures
			if failures == 0 {
				failures = session.FailureCount()
			}
			session.Finish(ui.Summary{Upgraded: result.Upgraded, Failures: failures})
		} else if result.Upgraded > 0 || result.Failures > 0 {
			ui.WriteSummary(out, ui.Summary{Upgraded: result.Upgraded, Failures: result.Failures}, fancy)
		}
	}

	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return errors.Join(upgradeErr, err)
	}
	applyRes, err := engine.BuildPlanResult(cmd.Context(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, stateDir, time.Now())
	if err != nil {
		return errors.Join(upgradeErr, err)
	}
	opts := applyOptions{
		mode:         policy.ResolveModeFromManifest(applyRes.Manifest),
		autoYes:      autoYes,
		quiet:        quiet,
		configPath:   configPath,
		configDir:    filepath.Dir(configPath),
		stateDir:     stateDir,
		manifest:     applyRes.Manifest,
		generationID: applyRes.GenerationID,
		now:          time.Now,
	}
	return errors.Join(upgradeErr, executeApply(cmd, runner, applyRes.Plan, opts))
}

// buildUpgradePlanForTest is used by tests with a custom runner.
func buildUpgradePlanForTest(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	return engine.BuildUpgradePlan(ctx, configPath, runner, nil)
}
