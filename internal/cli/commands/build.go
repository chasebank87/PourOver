package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewBuildCmd returns the build subcommand (evaluate Lua → activation generation).
func NewBuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build an activation generation from config (no live system changes)",
		Long: `Evaluate pourover.lua and freeze packages plus file contents into an
activation generation under Application Support. Does not install packages
or write live home paths — use pourover apply for that.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(cmd)
		},
	}
}

func runBuild(cmd *cobra.Command) error {
	configPath, verbose, _, err := planDisplayOptions(cmd)
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
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	res, _, err := engine.BuildGeneration(configPath, stateDir, time.Now())
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	ui.Successf(out, "☕ Built generation %s (%d files)", res.Manifest.ID, len(res.Manifest.Files))
	ui.Mutedf(out, "  %s", res.Dir)
	return nil
}
