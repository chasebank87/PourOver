package engine

import (
	"io"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
)

// Progress reports a single progress line to the frontend.
type Progress func(line string)

// ApplyOptions configures a reconcile apply run.
type ApplyOptions struct {
	ConfigPath  string
	ConfigDir   string               // for file links; defaults to dirname(ConfigPath)
	StateDir    string               // for file backup-on-replace
	Mode        config.UninstallMode // empty → resolve from manifest
	FileReplace config.FileReplaceMode
	AutoYes     bool
	Quiet       bool
	DryRun      bool
	Progress    Progress
	Confirm     Confirmer
	OnPhase     func(phase string) // optional; called before each mutation phase
	Stdout      io.Writer          // optional; brew log sink
	Stderr      io.Writer
	Now         func() time.Time // optional; timestamps for file backups
}
