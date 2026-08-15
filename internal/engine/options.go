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
	FilesMode   config.FilesMode // empty → resolve from manifest (callers should set)
	AutoYes     bool
	Quiet       bool
	DryRun      bool
	Progress    Progress
	Confirm     Confirmer
	OnPhase     func(phase string) // optional; called before each mutation phase
	Stdout      io.Writer          // optional; brew log sink
	Stderr      io.Writer
	Now         func() time.Time // optional; timestamps for file backups
	// PAMSudoLocalPath / PAMSudoPath override /etc/pam.d paths (tests inject temp dirs).
	PAMSudoLocalPath string
	PAMSudoPath      string
}
