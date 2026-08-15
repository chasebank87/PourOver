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
	MacOS    MacOS    `json:"macos"`
}

// MacOS holds declarative macOS preferences (nix-darwin-style defaults).
type MacOS struct {
	Defaults MacOSDefaults `json:"defaults"`
}

// MacOSDefaults groups named nix-darwin-style domains plus a custom escape hatch.
type MacOSDefaults struct {
	// Sections is keyed by catalog section name (dock, finder, NSGlobalDomain, …).
	Sections map[string]map[string]SettingValue `json:"sections,omitempty"`
	Custom   map[string]map[string]SettingValue `json:"custom,omitempty"`
}

// SettingKind is the typed value kind for a defaults write.
type SettingKind string

const (
	SettingBool   SettingKind = "bool"
	SettingInt    SettingKind = "int"
	SettingFloat  SettingKind = "float"
	SettingString SettingKind = "string"
	SettingArray  SettingKind = "array" // path lists (e.g. dock.persistent-apps)
)

// SettingValue is a typed preference value (bool/int/float/string/array).
type SettingValue struct {
	Kind   SettingKind `json:"kind"`
	Bool   bool        `json:"bool,omitempty"`
	Int    int64       `json:"int,omitempty"`
	Float  float64     `json:"float,omitempty"`
	String string      `json:"string,omitempty"`
	Array  []string    `json:"array,omitempty"`
}

// Packages lists Homebrew taps, formulae, and casks to reconcile.
type Packages struct {
	Taps     []string `json:"taps,omitempty"`
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

// Backup configures snapshot mirroring and config git sync.
type Backup struct {
	ICloud ICloudBackup `json:"icloud"`
	Git    GitBackup    `json:"git"`
}

// ICloudBackup mirrors state snapshots to iCloud Drive when enabled.
type ICloudBackup struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
}

// GitBackup keeps ~/.pourover as a git repo synced to a remote (e.g. GitHub).
type GitBackup struct {
	Enabled  bool   `json:"enabled"`
	Remote   string `json:"remote,omitempty"`
	AutoPush bool   `json:"auto_push"`
	Branch   string `json:"branch,omitempty"`
}
