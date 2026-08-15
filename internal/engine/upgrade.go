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
	Quiet    bool
	Progress Progress
	Stdout   io.Writer
	Stderr   io.Writer
}

// UpgradeResult summarizes brew upgrade outcomes for declared outdated packages.
type UpgradeResult struct {
	Upgraded int
	Failures int
}

// BuildUpgradePlan loads config, discovers brew + outdated packages, and returns
// the upgrade-only plan for declared formulae/casks that are installed and outdated.
func BuildUpgradePlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
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
	return plan.BuildUpgradePlan(manifest.Packages, brewState), nil
}

// UpgradePackages runs formula_upgrade and cask_upgrade actions via brew.
// It does not self-update pourover, build apply plans, or create UI sessions.
func UpgradePackages(ctx context.Context, runner discovery.Runner, p plan.Plan, opts UpgradeOptions) (UpgradeResult, error) {
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

	n, err := exec.ApplyUpgrades(ctx, mutRunner, p, progress)
	return UpgradeResult{Upgraded: n, Failures: failures}, err
}
