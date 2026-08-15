package engine

import (
	"io"

	"github.com/chasebank87/PourOver/internal/config"
)

// Progress reports a single progress line to the frontend.
type Progress func(line string)

// ApplyOptions configures a reconcile apply run.
type ApplyOptions struct {
	ConfigPath string
	Mode       config.UninstallMode // empty → resolve from manifest
	AutoYes    bool
	Quiet      bool
	DryRun     bool
	Progress   Progress
	Confirm    Confirmer
	Stdout     io.Writer // optional; brew log sink
	Stderr     io.Writer
}
