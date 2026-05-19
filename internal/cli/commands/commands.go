package commands

import "github.com/spf13/cobra"

// NewInitCmd returns the init subcommand.
func NewInitCmd() *cobra.Command {
	return notImplemented("init", "Scaffold ~/.pourover config")
}

// NewPlanCmd returns the plan subcommand.
func NewPlanCmd() *cobra.Command {
	return notImplemented("plan", "Show changes that apply would make")
}

// NewApplyCmd returns the apply subcommand.
func NewApplyCmd() *cobra.Command {
	return notImplemented("apply", "Reconcile system state to match config")
}

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
