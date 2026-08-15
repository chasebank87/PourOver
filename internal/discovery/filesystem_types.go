package discovery

import "github.com/chasebank87/PourOver/internal/config"

// LinkStatusKind describes the target path for a declared file link.
type LinkStatusKind string

const (
	LinkStatusMissing LinkStatusKind = "missing"
	LinkStatusCorrect LinkStatusKind = "correct"
	LinkStatusWrong   LinkStatusKind = "wrong"
	LinkStatusBlocked LinkStatusKind = "blocked"
)

// FileLinkStatus is the discovered state of one declared symlink.
type FileLinkStatus struct {
	Link       config.FileLink
	SourcePath string // resolved absolute source
	TargetPath string // resolved absolute target
	Kind       LinkStatusKind
	// ActualTarget is the canonical path the symlink currently points to (wrong links only).
	ActualTarget string
}

// ManagedStatusKind describes the target path for a declared managed file copy.
type ManagedStatusKind string

const (
	ManagedStatusMissing ManagedStatusKind = "missing"
	ManagedStatusSame    ManagedStatusKind = "same"
	ManagedStatusDiffer  ManagedStatusKind = "differ"
	ManagedStatusBlocked ManagedStatusKind = "blocked" // unexpected target type (e.g. directory)
)

// ManagedStatus is the discovered state of one declared managed file.
type ManagedStatus struct {
	File       config.ManagedFile
	SourcePath string // resolved absolute source
	TargetPath string // resolved absolute target
	Kind       ManagedStatusKind
}

// UnlinkStatusKind describes an explicit unlink path.
type UnlinkStatusKind string

const (
	UnlinkStatusMissing UnlinkStatusKind = "missing"
	UnlinkStatusRemove  UnlinkStatusKind = "remove"
)

// UnlinkStatus is the discovered state of one explicit unlink path.
type UnlinkStatus struct {
	Path       string // as declared in config
	TargetPath string // resolved absolute path
	Kind       UnlinkStatusKind
}

// TemplateStatusKind describes the target path for a declared template render.
type TemplateStatusKind string

const (
	TemplateStatusMissing TemplateStatusKind = "missing"
	TemplateStatusSame    TemplateStatusKind = "same"
	TemplateStatusDiffer  TemplateStatusKind = "differ"
	TemplateStatusBlocked TemplateStatusKind = "blocked" // unexpected target type (e.g. directory)
)

// TemplateStatus is the discovered state of one declared template file.
type TemplateStatus struct {
	File       config.TemplateFile
	SourcePath string // resolved absolute source template
	TargetPath string // resolved absolute target
	Rendered   string // rendered template body
	Current    string // existing target contents (empty when missing/blocked)
	Kind       TemplateStatusKind
}
