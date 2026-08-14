package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/exec"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

func runApply(cmd *cobra.Command, dryRun bool) error {
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

	return executeApply(cmd, runner, p)
}

func executeApply(cmd *cobra.Command, runner discovery.Runner, p plan.Plan) error {
	skipped := exec.UnsupportedApplyActions(p)
	if len(p.Actions) == 0 {
		fmt.Fprintln(os.Stderr, "No changes.")
		return nil
	}

	formulae, err := exec.ApplyFormulaInstalls(cmd.Context(), runner, p)
	if err != nil {
		return err
	}
	casks, err := exec.ApplyCaskInstalls(cmd.Context(), runner, p)
	if err != nil {
		return err
	}
	n := formulae + casks

	if formulae > 0 {
		fmt.Fprintf(os.Stderr, "Installed %d formula(s).\n", formulae)
	}
	if casks > 0 {
		fmt.Fprintf(os.Stderr, "Installed %d cask(s).\n", casks)
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "Skipped %d action(s) not yet supported by apply:\n", len(skipped))
		for _, a := range skipped {
			line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
			fmt.Fprintf(os.Stderr, "  %s\n", line)
		}
	}
	if n == 0 && len(skipped) == len(p.Actions) {
		fmt.Fprintln(os.Stderr, "No installs to apply.")
	}
	return nil
}
