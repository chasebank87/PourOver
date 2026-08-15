package engine

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/plan"
)

// UpgradeOptions configures package upgrade execution (not self-update).
type UpgradeOptions struct {
	Quiet     bool
	Progress  Progress
	Stdout    io.Writer
	Stderr    io.Writer
	MasRunner discovery.MasRunner // nil → NewExecMasRunner when mas upgrades run
}

// UpgradeResult summarizes brew/mas upgrade outcomes for declared outdated packages.
type UpgradeResult struct {
	Upgraded int
	Failures int
}

// BuildUpgradePlan loads config, discovers brew + outdated packages (and mas
// outdated when packages.mas is configured), and returns the upgrade-only plan
// for declared formulae/casks/apps that are installed and outdated.
// When masRunner is nil and mas is configured, NewExecMasRunner is used.
func BuildUpgradePlan(ctx context.Context, configPath string, runner discovery.Runner, masRunner discovery.MasRunner) (plan.Plan, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("load config: %w", err)
	}
	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover brew: %w", err)
	}
	outdated, err := discovery.DiscoverOutdated(ctx, runner)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover outdated: %w", err)
	}
	brewState.OutdatedFormulae = outdated.Formulae
	if brewState.OutdatedFormulae == nil {
		brewState.OutdatedFormulae = []string{}
	}
	brewState.OutdatedCasks = outdated.Casks
	if brewState.OutdatedCasks == nil {
		brewState.OutdatedCasks = []string{}
	}

	masState := discovery.MasState{}
	if manifest.Packages.MasConfigured {
		if masRunner == nil {
			masRunner = discovery.NewExecMasRunner()
		}
		masOutdated, err := discovery.DiscoverMasOutdated(ctx, masRunner)
		if err != nil {
			return plan.Plan{}, fmt.Errorf("discover mas outdated: %w", err)
		}
		if masOutdated == nil {
			masOutdated = []int64{}
		}
		masState.Outdated = masOutdated
	}

	return plan.BuildUpgradePlan(manifest.Packages, brewState, masState), nil
}

// UpgradePackages runs formula_upgrade, cask_upgrade, and mas_upgrade actions.
// It does not self-update pourover, build apply plans, or create UI sessions.
func UpgradePackages(ctx context.Context, runner discovery.Runner, p plan.Plan, opts UpgradeOptions) (UpgradeResult, error) {
	brewOut := opts.Stderr
	if brewOut == nil {
		brewOut = opts.Stdout
	}
	mutRunner := brewRunnerWithProgress(runner, brewOut, opts.Quiet)
	masRunner := masRunnerWithProgress(opts.MasRunner, brewOut, opts.Quiet)

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

	n, err := exec.ApplyUpgrades(ctx, mutRunner, resolveMasRunner(masRunner), p, progress)
	return UpgradeResult{Upgraded: n, Failures: failures}, err
}
