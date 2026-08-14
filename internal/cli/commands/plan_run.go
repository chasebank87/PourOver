package commands

import (
	"fmt"
	"os"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

// showPlan loads config, discovers state, builds a plan, and prints it.
func showPlan(cmd *cobra.Command, runner discovery.Runner) error {
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

	p, err := buildPlan(cmd.Context(), configPath, runner)
	if err != nil {
		return err
	}

	return printPlan(p, asJSON)
}

func planDisplayOptions(cmd *cobra.Command) (configPath string, verbose, asJSON bool, err error) {
	root := cmd.Root()
	verbose, err = root.PersistentFlags().GetBool("verbose")
	if err != nil {
		return "", false, false, err
	}
	configFlag, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return "", false, false, err
	}
	asJSON, err = root.PersistentFlags().GetBool("json")
	if err != nil {
		return "", false, false, err
	}
	configPath, err = paths.ResolveConfigFile(configFlag)
	if err != nil {
		return "", false, false, err
	}
	return configPath, verbose, asJSON, nil
}

func printPlan(p plan.Plan, asJSON bool) error {
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
