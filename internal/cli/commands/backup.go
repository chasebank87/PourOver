package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewBackupCmd returns the backup subcommand.
func NewBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Snapshot state and sync to iCloud",
		Long: `Write a local state snapshot under Application Support and, when
backup.icloud.enabled is true and iCloud Drive is available, mirror it.`,
		RunE: runBackup,
	}
}

func runBackup(cmd *cobra.Command, args []string) error {
	configPath, verbose, _, err := planDisplayOptions(cmd)
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s (run `pourover init` to scaffold)", configPath)
		}
		return fmt.Errorf("config file: %w", err)
	}
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	if verbose {
		ui.Mutedf(cmd.ErrOrStderr(), "state dir %s", stateDir)
	}

	result, err := engine.Backup(cmd.Context(), engine.BackupOptions{
		StateDir: stateDir,
		Manifest: manifest,
		Now:      time.Now,
	})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	ui.Successf(out, "☕ Snapshot written to %s", result.LocalSnapshot)
	if result.MirroredTo != "" {
		ui.Successf(out, "☕ Mirrored to %s", result.MirroredTo)
	} else if result.ICloudEnabled {
		ui.Warnf(out, "☕ iCloud mirror skipped (path unavailable)")
	}
	return nil
}

// NewRestoreCmd returns the restore subcommand.
func NewRestoreCmd() *cobra.Command {
	var snapshot string
	var fromICloud bool
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore state from a snapshot",
		Long: `Restore lock.json and last-plan.json from a local snapshot (default: latest)
or from iCloud with --icloud. Use --snapshot to pick a specific snapshot directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRestore(cmd, snapshot, fromICloud)
		},
	}
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "path to a snapshot directory")
	cmd.Flags().BoolVar(&fromICloud, "icloud", false, "restore the latest snapshot from the iCloud mirror")
	return cmd
}

func runRestore(cmd *cobra.Command, snapshot string, fromICloud bool) error {
	configPath, verbose, _, err := planDisplayOptions(cmd)
	if err != nil {
		return err
	}
	manifest := config.Manifest{}
	if _, err := os.Stat(configPath); err == nil {
		manifest, err = config.LoadManifest(configPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
	} else if fromICloud && snapshot == "" {
		return fmt.Errorf("config not found at %s (needed for iCloud path); pass --snapshot", configPath)
	}

	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	result, err := engine.Restore(cmd.Context(), engine.RestoreOptions{
		StateDir:   stateDir,
		Manifest:   manifest,
		Snapshot:   snapshot,
		FromICloud: fromICloud,
	})
	if err != nil {
		return err
	}
	if verbose {
		ui.Mutedf(cmd.ErrOrStderr(), "restoring from %s", result.SnapshotPath)
	}
	ui.Successf(cmd.OutOrStdout(), "☕ Restored state from %s into %s", result.SnapshotPath, result.StateDir)
	return nil
}
