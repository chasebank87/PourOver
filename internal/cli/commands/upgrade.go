package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
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
(like brew update), then runs brew upgrade for each declared formula/cask
that is already installed, then rebuilds the apply plan and reconciles.
Use --dry-run to preview package upgrade and apply actions (skips self-update).
Use --skip-self-update to only upgrade packages.
By default each action is printed as it runs; use --quiet for summary-only output.`,
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
		fmt.Fprintf(cmd.ErrOrStderr(), "using config %s\n", configPath)
	}

	runner := discovery.NewExecRunner()
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	brewState, err := discovery.DiscoverBrew(cmd.Context(), runner)
	if err != nil {
		return fmt.Errorf("discover brew: %w", err)
	}
	upgradePlan := plan.BuildUpgradePlan(manifest.Packages, brewState)

	if dryRun {
		applyPlan, err := buildPlan(cmd.Context(), configPath, runner)
		if err != nil {
			return err
		}
		combined := plan.MergePlans(upgradePlan, applyPlan)
		return printPlan(combined, asJSON)
	}

	out := cmd.ErrOrStderr()
	progress := applyProgress(out, quiet)
	mutRunner := brewRunnerWithProgress(runner, out, quiet)
	if len(upgradePlan.Actions) == 0 {
		fmt.Fprintln(out, "No package upgrades.")
	} else {
		n, err := exec.ApplyUpgrades(cmd.Context(), mutRunner, upgradePlan, progress)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "Upgraded %d package(s).\n", n)
	}

	applyPlan, err := buildPlan(cmd.Context(), configPath, runner)
	if err != nil {
		return err
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
	return executeApply(cmd, runner, applyPlan, opts)
}

// buildUpgradePlanForTest is used by tests with a custom runner.
func buildUpgradePlanForTest(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return plan.Plan{}, err
	}
	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return plan.Plan{}, err
	}
	return plan.BuildUpgradePlan(manifest.Packages, brewState), nil
}
