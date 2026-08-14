package commands

import "github.com/spf13/cobra"

// NewDoctorCmd returns the doctor subcommand.
func NewDoctorCmd() *cobra.Command {
	return notImplemented("doctor", "Check prerequisites and environment health")
}
