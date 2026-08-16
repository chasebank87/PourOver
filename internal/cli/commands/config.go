package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns the config parent command.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage PourOver config (iCloud mirror, git sync)",
		Long: `Configure backup and sync for PourOver config.

iCloud mirrors state snapshots. Git sync treats ~/.pourover as a git repo
for disaster recovery of the Lua config itself.`,
	}
	cmd.AddCommand(
		newConfigICloudCmd(),
		newConfigGitCmd(),
		newConfigPushCmd(),
		newConfigPullCmd(),
	)
	return cmd
}

func newConfigICloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "icloud",
		Short: "Enable or disable iCloud state snapshot mirroring",
	}
	cmd.AddCommand(newConfigICloudEnableCmd(), newConfigICloudDisableCmd())
	return cmd
}

func newConfigICloudEnableCmd() *cobra.Command {
	var pathFlag string
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable iCloud snapshot mirroring in pourover.lua",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigICloudEnable(cmd, pathFlag, cmd.Flags().Changed("path"))
		},
	}
	cmd.Flags().StringVar(&pathFlag, "path", "", "override iCloud mirror directory")
	return cmd
}

func newConfigICloudDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable iCloud snapshot mirroring in pourover.lua",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigICloudDisable(cmd)
		},
	}
}

func runConfigICloudEnable(cmd *cobra.Command, icloudPath string, setPath bool) error {
	configPath, _, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	st, err := engine.EnableICloud(configPath, icloudPath, setPath)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	ui.Successf(out, "☕ iCloud mirror enabled in %s", configPath)
	if st.ICloudAvailable {
		ui.Mutedf(out, "Mirror path: %s", st.ICloudPath)
	} else {
		ui.Warnf(out, "Mirror path currently unavailable (is iCloud Drive signed in?)")
	}
	ui.Mutedf(out, "Tip: run `pourover doctor` or `pourover backup` to verify.")
	return nil
}

func runConfigICloudDisable(cmd *cobra.Command) error {
	configPath, _, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	if err := engine.DisableICloud(configPath); err != nil {
		return err
	}
	ui.Successf(cmd.OutOrStdout(), "☕ iCloud mirror disabled in %s", configPath)
	return nil
}

func newConfigGitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Set up or restore config from a git remote",
	}
	cmd.AddCommand(newConfigGitSetupCmd(), newConfigGitRestoreCmd())
	return cmd
}

func newConfigGitSetupCmd() *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:   "setup <url>",
		Short: "Init git in the config dir, set remote, and push",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGitSetup(cmd, args[0], branch)
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "main", "branch name for backup.git")
	return cmd
}

func newConfigGitRestoreCmd() *cobra.Command {
	var (
		branch string
		force  bool
	)
	cmd := &cobra.Command{
		Use:   "restore <url>",
		Short: "Clone a remote into ~/.pourover (emergency restore)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGitRestore(cmd, args[0], branch, force)
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "main", "branch to clone")
	cmd.Flags().BoolVar(&force, "force", false, "replace non-empty config dir")
	return cmd
}

func runConfigGitSetup(cmd *cobra.Command, url, branch string) error {
	configPath, cfgDir, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	if err := requireConfigFile(configPath); err != nil {
		return err
	}
	if branch == "" {
		branch = "main"
	}
	if err := configgit.Init(cfgDir); err != nil {
		return err
	}
	if err := configgit.EnsureBranch(cfgDir, branch); err != nil {
		return err
	}
	if err := configgit.EnsureGitignore(cfgDir); err != nil {
		return err
	}
	if err := configgit.EnsureRemote(cfgDir, url); err != nil {
		return err
	}
	git := config.GitBackup{
		Enabled:  true,
		Remote:   url,
		AutoPush: true,
		Branch:   branch,
	}
	if err := config.PatchGitFile(configPath, git); err != nil {
		return err
	}

	dirty, err := configgit.StatusDirty(cfgDir)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if dirty {
		if err := configgit.AddAll(cfgDir); err != nil {
			return err
		}
		if err := configgit.Commit(cfgDir, configgit.SyncCommitMessage(time.Now())); err != nil {
			return err
		}
		ui.Successf(out, "☕ Created initial commit.")
	}
	if err := configgit.Push(cfgDir, branch); err != nil {
		return fmt.Errorf("push failed (is the remote empty/reachable and auth configured?): %w", err)
	}
	ui.Successf(out, "☕ Git config sync enabled for %s", cfgDir)
	ui.Mutedf(out, "Remote: %s (branch %s)", url, branch)
	ui.Mutedf(out, "Successful apply/import will auto-push when the config tree is dirty.")
	return nil
}

func runConfigGitRestore(cmd *cobra.Command, url, branch string, force bool) error {
	_, cfgDir, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	empty, err := configgit.DirEmpty(cfgDir)
	if err != nil {
		return err
	}
	if !empty {
		if !force {
			return fmt.Errorf("config dir %s is not empty (use --force to replace)", cfgDir)
		}
		if err := os.RemoveAll(cfgDir); err != nil {
			return fmt.Errorf("remove existing config dir: %w", err)
		}
	} else if _, err := os.Stat(cfgDir); err == nil {
		// exists but empty — remove so clone can create it
		if err := os.Remove(cfgDir); err != nil {
			return err
		}
	}

	if branch == "" {
		branch = "main"
	}
	if err := configgit.Clone(url, cfgDir, branch); err != nil {
		return err
	}
	configPath := filepath.Join(cfgDir, paths.ConfigFileName)
	if _, err := config.LoadManifest(configPath); err != nil {
		return fmt.Errorf("cloned repo but config invalid at %s: %w", configPath, err)
	}
	ui.Successf(cmd.OutOrStdout(), "☕ Restored PourOver config from %s into %s", url, cfgDir)
	ui.Mutedf(cmd.OutOrStdout(), "Next: run `pourover apply` to reconcile this machine.")
	return nil
}

func newConfigPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Commit and push config dir changes to the git remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPush(cmd)
		},
	}
}

func newConfigPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull config dir updates from the git remote",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPull(cmd)
		},
	}
}

func runConfigPush(cmd *cobra.Command) error {
	configPath, _, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	result, err := engine.PushConfig(configPath)
	if err != nil {
		return err
	}
	if !result.Pushed {
		ui.Mutedf(cmd.OutOrStdout(), "Nothing to push (already synced).")
		return nil
	}
	ui.Successf(cmd.OutOrStdout(), "☕ Pushed config changes to %s", result.Remote)
	return nil
}

func runConfigPull(cmd *cobra.Command) error {
	configPath, _, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	if err := engine.PullConfig(configPath); err != nil {
		return err
	}
	_, cfgDir, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	ui.Successf(cmd.OutOrStdout(), "☕ Pulled config updates into %s", cfgDir)
	return nil
}

func resolveConfigPath(cmd *cobra.Command) (configPath, cfgDir string, err error) {
	configFlag, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return "", "", err
	}
	if configFlag != "" {
		return configFlag, filepath.Dir(configFlag), nil
	}
	cfgDir, err = paths.DefaultConfigDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(cfgDir, paths.ConfigFileName), cfgDir, nil
}

func requireConfigFile(configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s (run `pourover init` first)", configPath)
		}
		return err
	}
	return nil
}

// maybeAutoPushConfig soft-fails git auto-push after apply/import when enabled.
func maybeAutoPushConfig(cmd *cobra.Command, configPath string, manifest config.Manifest) {
	if !manifest.Backup.Git.Enabled || !manifest.Backup.Git.AutoPush {
		return
	}
	cfgDir := filepath.Dir(configPath)
	errOut := cmd.ErrOrStderr()
	if !configgit.IsRepo(cfgDir) {
		ui.Warnf(errOut, "warning: backup.git.enabled but %s is not a git repo", cfgDir)
		return
	}
	branch := manifest.Backup.Git.Branch
	if branch == "" {
		branch = "main"
	}
	pushed, err := configgit.CommitAndPushIfDirty(cfgDir, branch, time.Now())
	if err != nil {
		ui.Warnf(errOut, "warning: config git auto-push failed: %v", err)
		return
	}
	if pushed {
		ui.Successf(errOut, "☕ config synced to git remote")
	}
}
