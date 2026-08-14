package commands

import "github.com/spf13/cobra"

// NewDoctorCmd returns the doctor subcommand.
func NewDoctorCmd() *cobra.Command {
	return notImplemented("doctor", "Check prerequisites and environment health")
}

// NewBackupCmd returns the backup subcommand.
func NewBackupCmd() *cobra.Command {
	return notImplemented("backup", "Snapshot state and sync to iCloud")
}

// NewRestoreCmd returns the restore subcommand.
func NewRestoreCmd() *cobra.Command {
	return notImplemented("restore", "Restore state from a snapshot")
}
