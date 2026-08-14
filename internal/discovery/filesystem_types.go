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
