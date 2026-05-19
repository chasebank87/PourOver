package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

// NewPlanCmd returns the plan subcommand.
func NewPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show changes that apply would make",
		RunE:  runPlan,
	}
}

func runPlan(cmd *cobra.Command, args []string) error {
	root := cmd.Root()
	verbose, err := root.PersistentFlags().GetBool("verbose")
	if err != nil {
		return err
	}
	configFlag, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return err
	}
	asJSON, err := root.PersistentFlags().GetBool("json")
	if err != nil {
		return err
	}

	configPath, err := paths.ResolveConfigFile(configFlag)
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

	p, err := buildPlan(cmd.Context(), configPath, discovery.NewExecRunner())
	if err != nil {
		return err
	}

	if asJSON {
		data, err := plan.RenderJSON(p)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	}

	fmt.Print(plan.RenderText(p))
	return nil
}

func buildPlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("load config: %w", err)
	}

	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover brew: %w", err)
	}
	brewPlan := plan.BuildBrewPlan(manifest.Packages, brewState)

	configDir := filepath.Dir(configPath)
	linkStatuses, err := discovery.DiscoverFileLinks(manifest.Files.Links, configDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover files: %w", err)
	}
	filePlan, err := plan.BuildFilePlan(linkStatuses)
	if err != nil {
		return plan.Plan{}, err
	}

	return plan.MergePlans(brewPlan, filePlan), nil
}
