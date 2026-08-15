package commands

import (
	"github.com/chasebank87/PourOver/internal/tui"
	"github.com/spf13/cobra"
)

// NewTUICmd returns the tui subcommand (always launches the interactive TUI).
func NewTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive PourOver TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}
}
