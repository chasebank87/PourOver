package commands

import (
	"github.com/chasebank87/PourOver/internal/selfupdate"
	"github.com/chasebank87/PourOver/internal/version"
	"github.com/spf13/cobra"
)

// NewSelfUpdateCmd returns the self-update subcommand.
func NewSelfUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "self-update",
		Short: "Update the pourover binary from GitHub Releases",
		Long: `Self-update checks GitHub Releases for a newer PourOver build and replaces
the running binary in place (similar to how brew update refreshes Homebrew).`,
		RunE: runSelfUpdate,
	}
}

func runSelfUpdate(cmd *cobra.Command, args []string) error {
	_, err := selfupdate.CheckAndApply(selfupdate.Options{
		Current: version.Version,
		Stdout:  cmd.OutOrStdout(),
	})
	return err
}
