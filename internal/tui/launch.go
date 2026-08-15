package tui

// LaunchEnv describes the environment used to decide whether to auto-start the TUI.
type LaunchEnv struct {
	Interactive bool
	Args        []string // args without program name; nil/empty = no subcommand
	CI          bool     // CI=true
}

// ShouldAutoLaunch reports whether the TUI should start automatically (no explicit `tui` command).
// Explicit `pourover tui` is handled by the CLI subcommand and must not auto-launch.
func ShouldAutoLaunch(env LaunchEnv) bool {
	if !env.Interactive || env.CI {
		return false
	}
	if len(env.Args) > 0 {
		return false
	}
	return true
}
