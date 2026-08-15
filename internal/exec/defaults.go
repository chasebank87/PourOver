package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/tty"
)

// DefaultsApplier writes preferences and restarts UI processes.
type DefaultsApplier interface {
	Defaults(ctx context.Context, args ...string) ([]byte, error)
	Killall(ctx context.Context, process string) error
}

// ExecDefaultsApplier shells out to `defaults` and `killall`.
type ExecDefaultsApplier struct {
	DefaultsPath string
	Timeout      time.Duration
}

// NewExecDefaultsApplier returns a host applier.
func NewExecDefaultsApplier() *ExecDefaultsApplier {
	return &ExecDefaultsApplier{DefaultsPath: "defaults", Timeout: discovery.DefaultDefaultsTimeout}
}

// Defaults runs the defaults binary.
func (a *ExecDefaultsApplier) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	path := a.DefaultsPath
	if path == "" {
		path = "defaults"
	}
	return runHostCmd(ctx, a.Timeout, path, args...)
}

// Killall sends SIGTERM to a process name (best-effort).
func (a *ExecDefaultsApplier) Killall(ctx context.Context, process string) error {
	_, err := runHostCmd(ctx, a.Timeout, "killall", process)
	return err
}

func runHostCmd(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	if timeout == 0 {
		timeout = discovery.DefaultDefaultsTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok && len(exit.Stderr) > 0 {
			return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(exit.Stderr)))
		}
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// DefaultsApplyOptions configures defaults_write apply (elevation for system domains).
type DefaultsApplyOptions struct {
	// BeforeAuth parks fancy UI / prints a hint before sudo Password: on /dev/tty.
	BeforeAuth func()
	// Timeout for elevated sudo defaults (0 → discovery.DefaultDefaultsTimeout).
	Timeout time.Duration
}

// ApplyDefaultsWrites runs defaults_write actions then restarts affected UI apps.
// Domains under /Library/Preferences are written via `sudo defaults` (admin).
func ApplyDefaultsWrites(ctx context.Context, applier DefaultsApplier, p plan.Plan, opts DefaultsApplyOptions, progress Progress) (int, error) {
	n := 0
	domains := map[string]struct{}{}
	for _, a := range p.Actions {
		if a.Type != plan.ActionDefaultsWrite {
			continue
		}
		report(progress, a)
		args, err := defaultsWriteArgs(a)
		if err != nil {
			return n, err
		}
		if isSystemDefaultsDomain(a.Domain) {
			if err := elevatedDefaultsWrite(ctx, opts.Timeout, args, opts.BeforeAuth); err != nil {
				return n, fmt.Errorf("writing system preference %s %s: %w", a.Domain, a.Key, err)
			}
		} else if _, err := applier.Defaults(ctx, args...); err != nil {
			return n, err
		}
		domains[a.Domain] = struct{}{}
		n++
	}
	if n == 0 {
		return 0, nil
	}
	for _, proc := range killallForDomains(domains) {
		_ = applier.Killall(ctx, proc) // process may not be running
	}
	return n, nil
}

func isSystemDefaultsDomain(domain string) bool {
	return strings.HasPrefix(domain, "/Library/Preferences")
}

// elevatedDefaultsWrite is sudoDefaultsWrite in production; tests may stub it.
var elevatedDefaultsWrite = sudoDefaultsWrite

func sudoDefaultsWrite(ctx context.Context, timeout time.Duration, args []string, beforeAuth func()) error {
	if beforeAuth != nil {
		beforeAuth()
	} else {
		tty.SyncPromptLine()
	}
	// Belt-and-suspenders: BeforeAuth already syncs, but ensure the tty cursor
	// is column 0 immediately before sudo opens /dev/tty for Password:.
	tty.SyncPromptLine()
	if timeout == 0 {
		timeout = discovery.DefaultDefaultsTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// Do not CombinedOutput: sudo prompts on /dev/tty and must keep the
	// terminal interactive. Stderr stays on the terminal for errors.
	cmd := exec.CommandContext(ctx, "sudo", append([]string{"defaults"}, args...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo defaults %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func defaultsWriteArgs(a plan.Action) ([]string, error) {
	if a.Domain == "" || a.Key == "" {
		return nil, fmt.Errorf("defaults_write: missing domain or key")
	}
	kind := config.SettingKind(a.Kind)
	val := a.Value
	if kind == config.SettingString {
		expanded, err := expandHomeForDefaults(val)
		if err != nil {
			return nil, err
		}
		val = expanded
	}
	switch kind {
	case config.SettingBool:
		return []string{"write", a.Domain, a.Key, "-bool", val}, nil
	case config.SettingInt:
		return []string{"write", a.Domain, a.Key, "-int", val}, nil
	case config.SettingFloat:
		return []string{"write", a.Domain, a.Key, "-float", val}, nil
	case config.SettingString:
		return []string{"write", a.Domain, a.Key, "-string", val}, nil
	case config.SettingArray:
		var paths []string
		if err := json.Unmarshal([]byte(val), &paths); err != nil {
			return nil, fmt.Errorf("defaults_write %s %s: invalid array value: %w", a.Domain, a.Key, err)
		}
		plist, err := config.EncodeDockPersistentPlist(a.Key, paths)
		if err != nil {
			return nil, fmt.Errorf("defaults_write %s %s: %w", a.Domain, a.Key, err)
		}
		// Pass XML plist as the value (no -type flag); defaults parses it as an array.
		return []string{"write", a.Domain, a.Key, plist}, nil
	default:
		return nil, fmt.Errorf("defaults_write %s %s: unsupported kind %q", a.Domain, a.Key, a.Kind)
	}
}

func expandHomeForDefaults(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func killallForDomains(domains map[string]struct{}) []string {
	need := map[string]struct{}{}
	for d := range domains {
		procs := config.KillallForDomain(d)
		if len(procs) == 0 && d != "" {
			need["SystemUIServer"] = struct{}{}
			continue
		}
		for _, p := range procs {
			need[p] = struct{}{}
		}
	}
	order := []string{"Dock", "Finder", "SystemUIServer", "Calendar", "Activity Monitor"}
	var out []string
	for _, p := range order {
		if _, ok := need[p]; ok {
			out = append(out, p)
		}
	}
	for p := range need {
		found := false
		for _, o := range order {
			if o == p {
				found = true
				break
			}
		}
		if !found {
			out = append(out, p)
		}
	}
	return out
}
