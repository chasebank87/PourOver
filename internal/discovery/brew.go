package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DefaultBrewTimeout is the maximum time allowed for a single brew discovery invocation.
const DefaultBrewTimeout = 30 * time.Second

// DefaultBrewIdleTimeout kills a silent brew mutation after this long with no
// stdout/stderr (heartbeat lines do not count). Large cask restores stay alive
// while Homebrew is still printing progress.
const DefaultBrewIdleTimeout = 15 * time.Minute

// DefaultBrewAbsoluteTimeout is a safety cap for one brew mutation so a wedged
// process cannot run forever.
const DefaultBrewAbsoluteTimeout = 6 * time.Hour

// DefaultBrewMutationTimeout is the absolute mutation safety cap (alias).
const DefaultBrewMutationTimeout = DefaultBrewAbsoluteTimeout

// Runner executes brew CLI commands. Implementations must be safe for unit tests
// (use a fake in tests; ExecRunner calls the real brew binary).
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// ExecRunner runs the brew binary on the host.
type ExecRunner struct {
	// Path to the brew executable (default "brew").
	Path string
	// Timeout per discovery-style invocation (default DefaultBrewTimeout).
	Timeout time.Duration
	// MutationTimeout is the absolute safety cap for install/upgrade/uninstall
	// (default DefaultBrewAbsoluteTimeout).
	MutationTimeout time.Duration
	// IdleTimeout kills a mutation after this long with no brew output
	// (default DefaultBrewIdleTimeout). Set negative to disable.
	IdleTimeout time.Duration
	// HeartbeatInterval for silent streamed mutations (default DefaultBrewHeartbeatInterval).
	// Set negative to disable heartbeats in tests.
	HeartbeatInterval time.Duration
	// Stdin, when set, is attached to the brew process (default os.Stdin for mutations).
	Stdin io.Reader
	// Stdout/Stderr, when set, receive streamed brew output (in addition to capture).
	Stdout io.Writer
	Stderr io.Writer
}

// NewExecRunner returns an ExecRunner with defaults.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		Path:              "brew",
		Timeout:           DefaultBrewTimeout,
		MutationTimeout:   DefaultBrewAbsoluteTimeout,
		IdleTimeout:       DefaultBrewIdleTimeout,
		HeartbeatInterval: DefaultBrewHeartbeatInterval,
	}
}

// WithOutput returns a shallow copy that streams brew stdout/stderr.
func (r *ExecRunner) WithOutput(stdout, stderr io.Writer) *ExecRunner {
	cp := *r
	cp.Stdout = stdout
	cp.Stderr = stderr
	return &cp
}

// Run executes `brew` with the given arguments.
func (r *ExecRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	path := r.Path
	if path == "" {
		path = "brew"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultBrewTimeout
	}
	mutation := isBrewMutation(args)
	idleTimeout := time.Duration(0)
	if mutation {
		timeout = r.MutationTimeout
		if timeout == 0 {
			timeout = DefaultBrewAbsoluteTimeout
		}
		idleTimeout = r.IdleTimeout
		if idleTimeout == 0 {
			idleTimeout = DefaultBrewIdleTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	idleCtx := ctx
	idleCancel := func() {}
	if mutation && idleTimeout > 0 {
		idleCtx, idleCancel = context.WithCancel(ctx)
		defer idleCancel()
	}

	cmd := exec.CommandContext(idleCtx, path, args...)
	if mutation {
		configureBrewMutationProcess(cmd)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	// Apply should install missing packages only — never silently upgrade via
	// brew install's default "upgrade if already installed" behavior.
	if len(args) > 0 && args[0] == "install" {
		cmd.Env = append(os.Environ(), "HOMEBREW_NO_INSTALL_UPGRADE=1")
	}

	// Mutations may prompt (sudo password, cask installer confirms). Discovery
	// commands stay non-interactive.
	if mutation {
		if r.Stdin != nil {
			cmd.Stdin = r.Stdin
		} else {
			cmd.Stdin = os.Stdin
		}
	}

	var activity *activityWriter
	if mutation {
		// Always track brew output for idle kill; stream to UI when configured.
		hbOut := r.Stderr
		if hbOut == nil {
			hbOut = r.Stdout
		}
		activity = newActivityWriter(hbOut)
		if r.Stdout != nil && r.Stdout == r.Stderr {
			cmd.Stdout = io.MultiWriter(&stdoutBuf, activity)
			cmd.Stderr = io.MultiWriter(&stderrBuf, activity)
		} else {
			if r.Stdout != nil {
				cmd.Stdout = io.MultiWriter(&stdoutBuf, r.Stdout, activityTouch{activity})
			} else {
				cmd.Stdout = io.MultiWriter(&stdoutBuf, activityTouch{activity})
			}
			cmd.Stderr = io.MultiWriter(&stderrBuf, activity)
		}
		interval := r.HeartbeatInterval
		if interval == 0 {
			interval = DefaultBrewHeartbeatInterval
		}
		if interval > 0 && hbOut != nil {
			stopHB := startBrewHeartbeat(hbOut, activity, args, interval)
			defer stopHB()
		}
		if idleTimeout > 0 {
			stopIdle := startBrewIdleCancel(idleCancel, activity, idleTimeout)
			defer stopIdle()
		}
	}

	err := cmd.Run()
	if f, ok := r.Stdout.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	if f, ok := r.Stderr.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	if activity != nil {
		_ = activity.Flush()
	}
	out := stdoutBuf.Bytes()
	if err != nil {
		stderr := strings.TrimSpace(stderrBuf.String())
		if stderr != "" {
			return out, fmt.Errorf("brew %s: %w: %s", strings.Join(args, " "), err, stderr)
		}
		return out, fmt.Errorf("brew %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// activityTouch updates last-activity without writing bytes (for the other stream).
type activityTouch struct {
	a *activityWriter
}

func (t activityTouch) Write(p []byte) (int, error) {
	t.a.touchOutput()
	return len(p), nil
}

func isBrewMutation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "install", "uninstall", "remove", "reinstall", "upgrade", "tap", "untap", "trust", "update":
		return true
	default:
		return false
	}
}

// BrewPrefix returns the Homebrew prefix via `brew --prefix` or
// `brew --prefix <formula>` when formula is non-empty.
// Works for installed and (typically) uninstalled formulae; stub in unit tests.
// Pass an empty formula for the Homebrew root prefix (e.g. /opt/homebrew).
func BrewPrefix(ctx context.Context, runner Runner, formula string) (string, error) {
	args := []string{"--prefix"}
	label := "brew --prefix"
	if formula != "" {
		args = append(args, formula)
		label = "brew --prefix " + formula
	}
	out, err := runner.Run(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	prefix := strings.TrimSpace(string(out))
	if prefix == "" {
		return "", fmt.Errorf("%s: empty output", label)
	}
	return prefix, nil
}
