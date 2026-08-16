package discovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// masListVersionSuffix matches a trailing " (version)" from `mas list` output
// (e.g. "Server (5.7.1)"). Only the final parenthetical is stripped so names
// like "App (Beta) Edition" keep mid-name parentheses.
var masListVersionSuffix = regexp.MustCompile(`\s+\([^)]+\)$`)

// DefaultMasTimeout is the maximum time allowed for a single mas discovery invocation.
const DefaultMasTimeout = 30 * time.Second

// DefaultMasMutationTimeout is used for install/uninstall/upgrade (downloads can be slow).
const DefaultMasMutationTimeout = 30 * time.Minute

// MasRunner executes mas CLI commands. Implementations must be safe for unit tests
// (use a fake in tests; ExecMasRunner calls the real mas binary).
type MasRunner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// MasInstalled is one Mac App Store app reported by `mas list`.
type MasInstalled struct {
	ID   int64
	Name string
}

// MasState is the set of Mac App Store apps currently installed (and optionally outdated).
type MasState struct {
	Apps     []MasInstalled // from list
	Outdated []int64        // ids from outdated; nil = not discovered
}

// ExecMasRunner runs the mas binary on the host.
type ExecMasRunner struct {
	// Path to the mas executable (default "mas").
	Path string
	// Timeout per discovery-style invocation (default DefaultMasTimeout).
	Timeout time.Duration
	// MutationTimeout for install/uninstall/upgrade (default DefaultMasMutationTimeout).
	MutationTimeout time.Duration
	// Stdin, when set, is attached to the mas process (default os.Stdin for mutations).
	Stdin io.Reader
	// Stdout/Stderr, when set, receive streamed mas output (in addition to capture).
	Stdout io.Writer
	Stderr io.Writer
}

// NewExecMasRunner returns an ExecMasRunner with defaults.
func NewExecMasRunner() *ExecMasRunner {
	return &ExecMasRunner{
		Path:            "mas",
		Timeout:         DefaultMasTimeout,
		MutationTimeout: DefaultMasMutationTimeout,
	}
}

// WithOutput returns a shallow copy that streams mas stdout/stderr.
func (r *ExecMasRunner) WithOutput(stdout, stderr io.Writer) *ExecMasRunner {
	cp := *r
	cp.Stdout = stdout
	cp.Stderr = stderr
	return &cp
}

// Run executes `mas` with the given arguments.
func (r *ExecMasRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	path := r.Path
	if path == "" {
		path = "mas"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultMasTimeout
	}
	mutation := isMasMutation(args)
	if mutation {
		timeout = r.MutationTimeout
		if timeout == 0 {
			timeout = DefaultMasMutationTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Discovery must not hang while mas auto-indexes Spotlight (mas 7).
	if !mutation {
		cmd.Env = append(os.Environ(), "MAS_NO_AUTO_INDEX=1")
	}

	if mutation {
		if r.Stdin != nil {
			cmd.Stdin = r.Stdin
		} else {
			cmd.Stdin = os.Stdin
		}
	}

	if mutation && (r.Stdout != nil || r.Stderr != nil) {
		if r.Stdout != nil {
			cmd.Stdout = io.MultiWriter(&stdoutBuf, r.Stdout)
		}
		if r.Stderr != nil {
			cmd.Stderr = io.MultiWriter(&stderrBuf, r.Stderr)
		}
	}

	err := cmd.Run()
	out := stdoutBuf.Bytes()
	if err != nil {
		stderr := strings.TrimSpace(stderrBuf.String())
		if stderr != "" {
			return out, fmt.Errorf("mas %s: %w: %s", strings.Join(args, " "), err, stderr)
		}
		return out, fmt.Errorf("mas %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

func isMasMutation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "install", "uninstall", "upgrade", "update", "get", "purchase", "lucky":
		return true
	default:
		return false
	}
}

// DiscoverMas lists installed Mac App Store apps via `mas list`.
// Outdated is left nil (not discovered); call DiscoverMasOutdated separately.
func DiscoverMas(ctx context.Context, r MasRunner) (MasState, error) {
	out, err := r.Run(ctx, "list")
	if err != nil {
		return MasState{}, fmt.Errorf("list mas apps: %w", err)
	}
	return MasState{
		Apps: parseMasList(out),
	}, nil
}

// IsMasNotFound reports whether err indicates the mas executable is missing
// from PATH (LookPath / exec.ErrNotFound), including wrapped errors from Run.
func IsMasNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

// DiscoverMasOutdated lists outdated Mac App Store app IDs via `mas outdated`.
func DiscoverMasOutdated(ctx context.Context, r MasRunner) ([]int64, error) {
	out, err := r.Run(ctx, "outdated")
	if err != nil {
		return nil, fmt.Errorf("list outdated mas apps: %w", err)
	}
	return parseMasOutdated(out), nil
}

// parseMasList parses `mas list` stdout: "ID Name… [(version)]" per line.
// Trailing " (version)" is stripped from Name. Blank lines and lines whose
// first field is not an integer ID are skipped.
func parseMasList(out []byte) []MasInstalled {
	var apps []MasInstalled
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		id, name, ok := parseMasIDNameLine(line)
		if !ok {
			continue
		}
		apps = append(apps, MasInstalled{ID: id, Name: name})
	}
	return apps
}

// parseMasOutdated parses `mas outdated` stdout for app IDs.
// Resilient: first whitespace-separated field as int64 ID; skip blank/non-ID lines.
// Name/version remainder (e.g. "Xcode (15.4 -> 16.0)") is ignored.
func parseMasOutdated(out []byte) []int64 {
	var ids []int64
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		id, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func parseMasIDNameLine(line string) (id int64, name string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, "", false
	}
	id, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, "", false
	}
	// Preserve spacing in the display name (everything after the ID token).
	rest := strings.TrimSpace(line[len(fields[0]):])
	if rest == "" {
		return 0, "", false
	}
	name = masListVersionSuffix.ReplaceAllString(rest, "")
	if name == "" {
		return 0, "", false
	}
	return id, name, true
}
