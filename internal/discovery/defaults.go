package discovery

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
)

// DefaultDefaultsTimeout is the max time for a single defaults invocation.
const DefaultDefaultsTimeout = 15 * time.Second

// DefaultsRunner runs the macOS `defaults` CLI.
type DefaultsRunner interface {
	Defaults(ctx context.Context, args ...string) ([]byte, error)
}

// ExecDefaultsRunner shells out to `defaults`.
type ExecDefaultsRunner struct {
	Path    string
	Timeout time.Duration
}

// NewExecDefaultsRunner returns an ExecDefaultsRunner with defaults.
func NewExecDefaultsRunner() *ExecDefaultsRunner {
	return &ExecDefaultsRunner{Path: "defaults", Timeout: DefaultDefaultsTimeout}
}

// Defaults runs `defaults` with the given args.
func (r *ExecDefaultsRunner) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	path := r.Path
	if path == "" {
		path = "defaults"
	}
	timeout := r.Timeout
	if timeout == 0 {
		timeout = DefaultDefaultsTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return nil, &DefaultsExitError{
				Args:   args,
				Stderr: strings.TrimSpace(string(exit.Stderr)),
				Err:    err,
			}
		}
		return nil, fmt.Errorf("defaults %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// DefaultsExitError is a non-zero exit from `defaults`.
type DefaultsExitError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *DefaultsExitError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("defaults %s: %v: %s", strings.Join(e.Args, " "), e.Err, e.Stderr)
	}
	return fmt.Sprintf("defaults %s: %v", strings.Join(e.Args, " "), e.Err)
}

func (e *DefaultsExitError) Unwrap() error { return e.Err }

// SettingStatus is the current vs desired state for one preference.
type SettingStatus struct {
	Desired config.DesiredSetting
	Current config.SettingValue
	Found   bool
	Drift   bool
}

// DiscoverDefaults reads current values for each desired setting.
func DiscoverDefaults(ctx context.Context, runner DefaultsRunner, desired []config.DesiredSetting) ([]SettingStatus, error) {
	out := make([]SettingStatus, 0, len(desired))
	for _, d := range desired {
		raw, found, err := readDefault(ctx, runner, d.Domain, d.Key)
		if err != nil {
			return nil, err
		}
		st := SettingStatus{Desired: d, Found: found}
		if found {
			cur, err := ParseDefaultsRead(raw, d.Value.Kind)
			if err != nil {
				// Unparseable current value → treat as drift so we rewrite.
				st.Drift = true
			} else {
				st.Current = cur
				st.Drift = !SettingValuesEqual(cur, d.Value)
			}
		} else {
			st.Drift = true
		}
		out = append(out, st)
	}
	return out, nil
}

func readDefault(ctx context.Context, runner DefaultsRunner, domain, key string) (string, bool, error) {
	out, err := runner.Defaults(ctx, "read", domain, key)
	if err != nil {
		if isDefaultsMissing(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(string(out)), true, nil
}

func isDefaultsMissing(err error) bool {
	var exit *DefaultsExitError
	if !asDefaultsExit(err, &exit) {
		return false
	}
	msg := strings.ToLower(exit.Stderr)
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not exist")
}

func asDefaultsExit(err error, target **DefaultsExitError) bool {
	if e, ok := err.(*DefaultsExitError); ok {
		*target = e
		return true
	}
	return false
}

// ParseDefaultsRead converts `defaults read` stdout into a SettingValue of the expected kind.
func ParseDefaultsRead(raw string, want config.SettingKind) (config.SettingValue, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"`)
	switch want {
	case config.SettingBool:
		switch strings.ToLower(raw) {
		case "1", "true", "yes":
			return config.SettingValue{Kind: config.SettingBool, Bool: true}, nil
		case "0", "false", "no":
			return config.SettingValue{Kind: config.SettingBool, Bool: false}, nil
		default:
			return config.SettingValue{}, fmt.Errorf("cannot parse bool from %q", raw)
		}
	case config.SettingInt:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			f, ferr := strconv.ParseFloat(raw, 64)
			if ferr != nil {
				return config.SettingValue{}, fmt.Errorf("cannot parse int from %q", raw)
			}
			n = int64(f)
		}
		return config.SettingValue{Kind: config.SettingInt, Int: n}, nil
	case config.SettingFloat:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return config.SettingValue{}, fmt.Errorf("cannot parse float from %q", raw)
		}
		return config.SettingValue{Kind: config.SettingFloat, Float: f}, nil
	case config.SettingString:
		return config.SettingValue{Kind: config.SettingString, String: raw}, nil
	default:
		return config.SettingValue{}, fmt.Errorf("unsupported kind %q", want)
	}
}

// SettingValuesEqual compares two typed preference values.
func SettingValuesEqual(a, b config.SettingValue) bool {
	if a.Kind != b.Kind {
		// int/float numeric equality
		if (a.Kind == config.SettingInt || a.Kind == config.SettingFloat) &&
			(b.Kind == config.SettingInt || b.Kind == config.SettingFloat) {
			return settingNumber(a) == settingNumber(b)
		}
		return false
	}
	switch a.Kind {
	case config.SettingBool:
		return a.Bool == b.Bool
	case config.SettingInt:
		return a.Int == b.Int
	case config.SettingFloat:
		return a.Float == b.Float
	case config.SettingString:
		as, _ := expandHome(a.String)
		bs, _ := expandHome(b.String)
		return as == bs
	default:
		return false
	}
}

func settingNumber(v config.SettingValue) float64 {
	if v.Kind == config.SettingFloat {
		return v.Float
	}
	return float64(v.Int)
}
