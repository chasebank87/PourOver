package policy

import "github.com/chasebank87/PourOver/internal/config"

// ResolveMode returns the uninstall mode for apply. Empty or unknown values
// default to safe (prompt before removes).
func ResolveMode(value string) config.UninstallMode {
	switch config.UninstallMode(value) {
	case config.UninstallModeSafe, config.UninstallModeStrict, config.UninstallModeNonDestructive:
		return config.UninstallMode(value)
	default:
		return config.UninstallModeSafe
	}
}

// ResolveModeFromManifest reads policy.uninstall_mode from the manifest.
func ResolveModeFromManifest(m config.Manifest) config.UninstallMode {
	return ResolveMode(string(m.Policy.UninstallMode))
}
