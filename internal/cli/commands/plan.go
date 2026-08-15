package commands

import (
	"context"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

// NewPlanCmd returns the plan subcommand.
func NewPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show changes that apply would make",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showPlan(cmd, discovery.NewExecRunner())
		},
	}
}

func buildPlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	return engine.BuildPlan(ctx, configPath, runner)
}

func buildPlanWith(ctx context.Context, configPath string, runner discovery.Runner, defaultsRunner discovery.DefaultsRunner) (plan.Plan, error) {
	return engine.BuildPlanWith(ctx, configPath, runner, defaultsRunner, nil, "")
}
