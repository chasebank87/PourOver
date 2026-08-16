package engine

import (
	"io"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// Progress reports a single progress line to the frontend.
type Progress func(line string)

// ApplyOptions configures a reconcile apply run.
type ApplyOptions struct {
	ConfigPath   string
	ConfigDir    string               // for declared files; defaults to dirname(ConfigPath)
	StateDir     string               // for file backup-on-replace
	Mode         config.UninstallMode // empty → resolve from manifest
	FileReplace  config.FileReplaceMode
	FilesMode    config.FilesMode // empty → resolve from manifest (callers should set)
	AutoYes      bool
	Quiet        bool
	DryRun       bool
	Progress     Progress
	Confirm      Confirmer
	OnPhase      func(phase string) // optional; called before each mutation phase
	Stdout       io.Writer          // optional; brew/mas log sink
	Stderr       io.Writer
	Now          func() time.Time // optional; timestamps for file backups
	GenerationID string           // activation generation for file blob payloads
	// MasRunner executes mas CLI mutations (nil → NewExecMasRunner).
	MasRunner discovery.MasRunner
	// PAMSudoLocalPath / PAMSudoPath override /etc/pam.d paths (tests inject temp dirs).
	PAMSudoLocalPath string
	PAMSudoPath      string
	// BeforeAuth parks fancy UI / prints a hint before sudo prompts on /dev/tty
	// (PAM elevation does not stream through Stdout, so Session.Write never runs).
	BeforeAuth func()
	// BeforePrompt parks fancy UI before y/n confirms (prune / uninstall) so the
	// live progress bar is not glued onto the prompt.
	BeforePrompt func()
}
