package commands

import (
	"github.com/spf13/cobra"
)

// NewApplyCmd returns the apply subcommand.
func NewApplyCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Reconcile system state to match config",
		Long: `Apply reconciles the system toward config. Formula and cask installs are supported;
removes and file links are skipped until later milestones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without modifying the system (same as plan)")
	return cmd
}
