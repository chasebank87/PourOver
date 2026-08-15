package cli_test

import (
	"slices"
	"testing"

	"github.com/chasebank87/PourOver/internal/cli"
	"github.com/spf13/cobra"
)

func TestRootCommand_HasCoreSubcommands(t *testing.T) {
	names := subcommandNames(cli.NewRootCommand())

	required := []string{"init", "import", "config", "plan", "apply", "upgrade", "self-update", "doctor", "backup", "restore", "tui"}
	for _, name := range required {
		if !slices.Contains(names, name) {
			t.Errorf("missing subcommand %q; registered: %v", name, names)
		}
	}
}

func TestRun_ExitCodes(t *testing.T) {
	if code := cli.Run([]string{"--help"}); code != cli.ExitOK {
		t.Fatalf("--help exit = %d, want %d", code, cli.ExitOK)
	}
	if code := cli.Run([]string{"not-a-command"}); code != cli.ExitInvalidUsage {
		t.Fatalf("unknown command exit = %d, want %d", code, cli.ExitInvalidUsage)
	}
}

func subcommandNames(cmd *cobra.Command) []string {
	children := cmd.Commands()
	names := make([]string, 0, len(children))
	for _, c := range children {
		names = append(names, c.Name())
	}
	return names
}
