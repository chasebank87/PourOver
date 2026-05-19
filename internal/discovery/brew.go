package discovery

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultBrewTimeout is the maximum time allowed for a single brew invocation.
const DefaultBrewTimeout = 30 * time.Second

// Runner executes brew CLI commands. Implementations must be safe for unit tests
// (use a fake in tests; ExecRunner calls the real brew binary).
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// ExecRunner runs the brew binary on the host.
type ExecRunner struct {
	// Path to the brew executable (default "brew").
	Path string
	// Timeout per invocation (default DefaultBrewTimeout).
	Timeout time.Duration
}

// NewExecRunner returns an ExecRunner with defaults.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{
		Path:    "brew",
		Timeout: DefaultBrewTimeout,
	}
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

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) > 0 {
			return nil, fmt.Errorf("brew %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("brew %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
