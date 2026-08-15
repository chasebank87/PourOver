package engine

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
)

// BuildPlan loads config at configPath, discovers current state via runner, and
// returns the merged reconcile plan. Defaults discovery uses the system
// defaults runner (same as the former CLI buildPlan).
func BuildPlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	return BuildPlanWith(ctx, configPath, runner, discovery.NewExecDefaultsRunner())
}

// BuildPlanWith is like BuildPlan but accepts an explicit DefaultsRunner
// (useful for tests that stub macOS defaults).
func BuildPlanWith(ctx context.Context, configPath string, runner discovery.Runner, defaultsRunner discovery.DefaultsRunner) (plan.Plan, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("load config: %w", err)
	}

	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover brew: %w", err)
	}
	brewPlan := plan.BuildBrewPlan(manifest.Packages, brewState)
	brewPlan, err = plan.AdviseCaskRenames(ctx, runner, brewPlan, brewState.Casks)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("detect cask renames: %w", err)
	}

	desired := config.FlattenDefaults(manifest.MacOS.Defaults)
	statuses, err := discovery.DiscoverDefaults(ctx, defaultsRunner, desired)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover macos defaults: %w", err)
	}
	defaultsPlan := plan.BuildDefaultsPlan(statuses)

	configDir := filepath.Dir(configPath)
	linkStatuses, err := discovery.DiscoverFileLinks(manifest.Files.Links, configDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover files: %w", err)
	}
	filePlan, err := plan.BuildFilePlan(linkStatuses)
	if err != nil {
		return plan.Plan{}, err
	}

	// brew → macos defaults → file links
	return plan.MergePlans(brewPlan, defaultsPlan, filePlan), nil
}
