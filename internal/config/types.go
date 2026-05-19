package config

// UninstallMode controls how apply handles packages not in the manifest.
type UninstallMode string

const (
	UninstallModeSafe           UninstallMode = "safe"
	UninstallModeStrict         UninstallMode = "strict"
	UninstallModeNonDestructive UninstallMode = "non_destructive"
)

// Manifest is the normalized declarative desired state (from pourover.lua).
type Manifest struct {
	Packages Packages `json:"packages"`
	Files    Files    `json:"files"`
	Policy   Policy   `json:"policy"`
	Backup   Backup   `json:"backup"`
}

// Packages lists Homebrew formulae and casks to reconcile.
type Packages struct {
	Formulae []string `json:"formulae,omitempty"`
	Casks    []string `json:"casks,omitempty"`
}

// Files declares dotfiles and other paths to reconcile on disk.
type Files struct {
	Links   []FileLink    `json:"links,omitempty"`
	Managed []ManagedFile `json:"managed,omitempty"`
}

// FileLink creates or updates a symlink from Source to Target.
type FileLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ManagedFile copies or templates a file from Source to Target.
type ManagedFile struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Policy holds safety and behavior options.
type Policy struct {
	UninstallMode UninstallMode `json:"uninstall_mode"`
}

// Backup configures snapshot mirroring (e.g. iCloud).
type Backup struct {
	ICloud ICloudBackup `json:"icloud"`
}

// ICloudBackup mirrors state snapshots to iCloud Drive when enabled.
type ICloudBackup struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
}
