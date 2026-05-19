package cli

import "github.com/spf13/cobra"

// GlobalOptions holds persistent CLI flags (set from the root command).
type GlobalOptions struct {
	Verbose    bool
	ConfigPath string
	JSON       bool
}

// global is populated when the root command registers persistent flags.
var global GlobalOptions

// Global returns the active global CLI options.
func Global() GlobalOptions {
	return global
}

func registerPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&global.Verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringVar(&global.ConfigPath, "config", "", "path to pourover.lua (default: ~/.pourover/pourover.lua)")
	cmd.PersistentFlags().BoolVar(&global.JSON, "json", false, "machine-readable output (plan and related commands)")
}
