package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/chasebank87/PourOver/internal/cli/commands"
	"github.com/chasebank87/PourOver/internal/tui"
	"github.com/chasebank87/PourOver/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Exit codes (D6).
const (
	ExitOK           = 0
	ExitFailure      = 1
	ExitInvalidUsage = 2
)

// Execute runs the root command and exits with a documented status code.
func Execute() {
	os.Exit(Run(os.Args[1:]))
}

// Run executes the CLI with the given args (without program name) and returns an exit code.
func Run(args []string) int {
	if tui.ShouldAutoLaunch(tui.LaunchEnv{
		Interactive: isInteractive(),
		Args:        args,
		CI:          isCI(),
	}) {
		if err := tui.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return ExitFailure
		}
		return ExitOK
	}

	root := NewRootCommand()
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if isUsageError(err) {
			return ExitInvalidUsage
		}
		return ExitFailure
	}
	return ExitOK
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func isCI() bool {
	return os.Getenv("CI") == "true"
}

func isUsageError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "unknown shorthand flag") ||
		strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "accepts ") // e.g. "accepts 0 arg(s)"
}

// NewRootCommand returns the pourover root cobra command.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pourover",
		Short: "Declarative macOS environment manager",
		Long: `PourOver is a declarative macOS environment manager.

It reconciles Homebrew packages, dotfiles, and config files from a
declarative config (~/.pourover/) with one command: pourover apply.`,
		Version:       version.String(),
		SilenceErrors: true,
	}
	registerPersistentFlags(cmd)
	cmd.AddCommand(
		commands.NewInitCmd(),
		commands.NewImportCmd(),
		commands.NewConfigCmd(),
		commands.NewPlanCmd(),
		commands.NewApplyCmd(),
		commands.NewUpgradeCmd(),
		commands.NewSelfUpdateCmd(),
		commands.NewDoctorCmd(),
		commands.NewBackupCmd(),
		commands.NewRestoreCmd(),
		commands.NewTUICmd(),
	)
	return cmd
}
