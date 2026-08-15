package commands

import (
	"fmt"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/scaffold"
	"github.com/spf13/cobra"
)

// NewInitCmd returns the init subcommand.
func NewInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold ~/.pourover config",
		Long: `Create ~/.pourover with pourover.lua, packages.lua, and an example
config/ tree for file-link sources. Refuses to overwrite existing files
unless --force is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing scaffold files")
	return cmd
}

func runInit(cmd *cobra.Command, force bool) error {
	root := cmd.Root()
	configFlag, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return err
	}

	var cfgDir string
	if configFlag != "" {
		cfgDir = filepath.Dir(configFlag)
	} else {
		cfgDir, err = paths.DefaultConfigDir()
		if err != nil {
			return err
		}
	}

	if err := scaffold.InitConfigDir(cfgDir, force); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Initialized PourOver config in %s\n", cfgDir)
	return nil
}
