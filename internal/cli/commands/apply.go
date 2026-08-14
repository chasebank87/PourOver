package commands

import (
	"github.com/spf13/cobra"
)

// NewApplyCmd returns the apply subcommand.
func NewApplyCmd() *cobra.Command {
	var dryRun bool
	var autoYes bool
	var quiet bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Reconcile system state to match config",
		Long: `Apply reconciles the system toward config: Homebrew packages (installs and
policy-aware removes), macOS defaults, then file symlinks.

On an interactive terminal, apply shows a PourOver header and progress bar with
colored status while Homebrew output scrolls underneath (restyled with ☕).
Use --quiet for summary-only output. Respects NO_COLOR.
In safe uninstall mode, removes require confirmation unless --yes is set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(cmd, dryRun, autoYes, quiet)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without modifying the system (same as plan)")
	cmd.Flags().BoolVar(&autoYes, "yes", false, "skip confirmation prompts (for CI / non-interactive use)")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress per-action progress output")
	return cmd
}
