package discovery

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	// Stdout/Stderr, when set, receive streamed brew output (in addition to capture).
	Stdout io.Writer
	Stderr io.Writer
}

// NewExecRunner returns an ExecRunner with defaults.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		Path:            "brew",
		Timeout:         DefaultBrewTimeout,
		MutationTimeout: DefaultBrewMutationTimeout,
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
	if isBrewMutation(args) {
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
	if r.Stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdoutBuf, r.Stdout)
	}
	if r.Stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderrBuf, r.Stderr)
	}

	err := cmd.Run()
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

func isBrewMutation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "install", "uninstall", "remove", "reinstall", "upgrade", "tap", "untap":
		return true
	default:
		return false
	}
}
