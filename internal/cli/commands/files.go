package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewFilesCmd returns the files parent command (unmanage, etc.).
func NewFilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Manage PourOver file links",
		Long: `Inspect and change which live paths PourOver manages via files.links.

Directory links own the whole tree on apply (copy, not symlink). Use
unmanage to stop managing an app config folder without deleting it.`,
	}
	cmd.AddCommand(newFilesUnmanageCmd())
	return cmd
}

func newFilesUnmanageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unmanage <target> [target...]",
		Short: "Stop managing file/dir targets without deleting live paths",
		Long: `Remove matching files.links entries from pourover.lua and clear related
owned_files in lock.json. Live paths under home (e.g. ~/.config/cursor) and
copies under ~/.pourover/config/ are left untouched.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFilesUnmanage(cmd, args)
		},
	}
}

func runFilesUnmanage(cmd *cobra.Command, targets []string) error {
	configPath, _, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s", configPath)
		}
		return err
	}
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}
	result, err := engine.UnmanageFiles(engine.UnmanageFilesOptions{
		ConfigPath: configPath,
		StateDir:   stateDir,
		Targets:    targets,
	})
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	for _, link := range result.RemovedLinks {
		ui.Successf(out, "☕ Unmanaged %s (was %s)", link.Target, filepath.ToSlash(link.Source))
	}
	if n := len(result.ClearedOwned); n > 0 {
		ui.Mutedf(out, "Cleared %d owned path(s) from lock.json", n)
	}
	ui.Mutedf(out, "Live files were not deleted. Run `pourover plan` to confirm.")
	return nil
}
