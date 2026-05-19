package cli

import (
	"os"

	"github.com/chasebank87/PourOver/internal/cli/commands"
	"github.com/spf13/cobra"
)

// Execute runs the root command.
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

// NewRootCommand returns the pourover root cobra command.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pourover",
		Short: "Declarative macOS environment manager",
		Long: `PourOver is a declarative macOS environment manager.

It reconciles Homebrew packages, dotfiles, and config files from a
		declarative config (~/.pourover/) with one command: pourover apply.`,
	}
	registerPersistentFlags(cmd)
	cmd.AddCommand(
		commands.NewInitCmd(),
		commands.NewPlanCmd(),
		commands.NewApplyCmd(),
		commands.NewDoctorCmd(),
		commands.NewBackupCmd(),
		commands.NewRestoreCmd(),
	)
	return cmd
}
