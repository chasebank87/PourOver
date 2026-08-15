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

// ResolveFileReplace returns the file replace mode. Empty or unknown values
// default to error (blocked targets fail the plan). "force" is an alias for backup.
func ResolveFileReplace(value string) config.FileReplaceMode {
	switch config.FileReplaceMode(value) {
	case config.FileReplaceBackup, config.FileReplaceMode("force"):
		return config.FileReplaceBackup
	case config.FileReplaceError, "":
		return config.FileReplaceError
	default:
		return config.FileReplaceError
	}
}

// ResolveFileReplaceFromManifest reads policy.file_replace from the manifest.
func ResolveFileReplaceFromManifest(m config.Manifest) config.FileReplaceMode {
	return ResolveFileReplace(string(m.Policy.FileReplace))
}
