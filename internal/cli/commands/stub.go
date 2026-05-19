package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func notImplemented(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%q is not implemented yet", cmd.Name())
		},
	}
}
