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
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade declared packages then reapply config",
		Long: `Upgrade runs brew upgrade for each declared formula/cask that is already
installed, then rebuilds the apply plan and reconciles (installs, removes,
file links). Use --dry-run to preview upgrade and apply actions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(cmd, dryRun, autoYes)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview upgrade and apply actions without modifying the system")
	cmd.Flags().BoolVar(&autoYes, "yes", false, "skip confirmation prompts during the apply phase")
	return cmd
}

func runUpgrade(cmd *cobra.Command, dryRun, autoYes bool) error {
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
	if len(upgradePlan.Actions) == 0 {
		fmt.Fprintln(out, "No package upgrades.")
	} else {
		n, err := exec.ApplyUpgrades(cmd.Context(), runner, upgradePlan)
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
		mode:      policy.ResolveModeFromManifest(manifest),
		autoYes:   autoYes,
		configDir: filepath.Dir(configPath),
		stateDir:  stateDir,
		manifest:  manifest,
		now:       time.Now,
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
