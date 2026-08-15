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
	Security MacOSSecurity `json:"security"`
}

// MacOSSecurity holds security-related macOS settings (PAM, etc.).
type MacOSSecurity struct {
	PAM MacOSPAM `json:"pam"`
}

// MacOSPAM holds PAM service configuration.
type MacOSPAM struct {
	SudoLocal SudoLocalPAM `json:"sudo_local"`
}

// SudoLocalPAM configures /etc/pam.d/sudo_local (Touch ID / Watch / reattach).
// Omitted sudo_local table means unmanaged (Configured=false).
// When the table is present, Enable defaults to true unless set to false.
type SudoLocalPAM struct {
	Configured  bool // true if sudo_local table present
	Enable      bool
	Reattach    bool
	TouchIDAuth bool
	WatchIDAuth bool
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

// TapSpec is a Homebrew tap with an optional trust flag (default trusted).
type TapSpec struct {
	Name    string `json:"name"`
	Trusted bool   `json:"trusted"` // default true when decoded from Lua string/table
}

// MasApp is a Mac App Store app declared by display name and numeric ID.
type MasApp struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

// Packages lists Homebrew taps, formulae, casks, and MAS apps to reconcile.
type Packages struct {
	Taps          []TapSpec `json:"taps,omitempty"`
	Formulae      []string  `json:"formulae,omitempty"`
	Casks         []string  `json:"casks,omitempty"`
	Mas           []MasApp  `json:"mas,omitempty"`
	MasConfigured bool      `json:"-"` // true if mas key present in Lua
}

// TapNames returns the tap repository names in declaration order.
func (p Packages) TapNames() []string {
	names := make([]string, len(p.Taps))
	for i, tap := range p.Taps {
		names[i] = tap.Name
	}
	return names
}

// Files declares dotfiles and other paths to reconcile on disk.
type Files struct {
	Links     []FileLink     `json:"links,omitempty"`
	Managed   []ManagedFile  `json:"managed,omitempty"`
	Templates []TemplateFile `json:"templates,omitempty"`
	Unlink    []string       `json:"unlink,omitempty"` // target paths (~ ok)
}

// FileLink creates or updates a symlink from Source to Target.
type FileLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// ManagedFile copies a file from Source to Target.
type ManagedFile struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// TemplateFile renders a text/template Source to Target (V2 Phase 5).
type TemplateFile struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// FileReplaceMode controls what happens when a link/managed target blocks replace.
type FileReplaceMode string

const (
	FileReplaceError  FileReplaceMode = "error"
	FileReplaceBackup FileReplaceMode = "backup"
)

// FilesMode mirrors uninstall_mode for PourOver-owned file prune.
type FilesMode string

const (
	FilesModeSafe           FilesMode = "safe"
	FilesModeStrict         FilesMode = "strict"
	FilesModeNonDestructive FilesMode = "non_destructive"
)

// Policy holds safety and behavior options.
type Policy struct {
	UninstallMode UninstallMode   `json:"uninstall_mode"`
	FileReplace   FileReplaceMode `json:"file_replace,omitempty"`
	FilesMode     FilesMode       `json:"files_mode,omitempty"` // default safe
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
