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

// DefaultBrewMutationTimeout is used for install/upgrade/uninstall (downloads can be slow).
const DefaultBrewMutationTimeout = 30 * time.Minute

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
	// MutationTimeout for install/upgrade/uninstall (default DefaultBrewMutationTimeout).
	MutationTimeout time.Duration
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
		MutationTimeout:   DefaultBrewMutationTimeout,
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
	if mutation {
		timeout = r.MutationTimeout
		if timeout == 0 {
			timeout = DefaultBrewMutationTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
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
	streamOut := r.Stdout
	streamErr := r.Stderr
	if mutation && (r.Stdout != nil || r.Stderr != nil) {
		// Track activity across both streams; heartbeats go to stderr stream if
		// present, else stdout (apply wires both to the same styled writer).
		hbOut := r.Stderr
		if hbOut == nil {
			hbOut = r.Stdout
		}
		activity = newActivityWriter(hbOut)
		if r.Stdout != nil {
			if r.Stdout == r.Stderr {
				streamOut = activity
				streamErr = activity
			} else {
				streamOut = io.MultiWriter(r.Stdout, activityTouch{activity})
				streamErr = activity
			}
		} else {
			streamErr = activity
		}
		interval := r.HeartbeatInterval
		if interval == 0 {
			interval = DefaultBrewHeartbeatInterval
		}
		if interval > 0 {
			stop := startBrewHeartbeat(activity, activity, args, interval)
			defer stop()
		}
	}

	if streamOut != nil {
		cmd.Stdout = io.MultiWriter(&stdoutBuf, streamOut)
	}
	if streamErr != nil {
		cmd.Stderr = io.MultiWriter(&stderrBuf, streamErr)
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
	t.a.touch()
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
