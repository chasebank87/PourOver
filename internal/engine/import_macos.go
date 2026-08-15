package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configimport"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// ImportMacOSOptions configures catalog defaults snapshot into macos.lua.
type ImportMacOSOptions struct {
	Force      bool
	DryRun     bool
	ConfigDir  string
	ConfigPath string
	Runner     discovery.DefaultsRunner
}

// ImportMacOSResult summarizes a macos defaults import for CLI/frontends.
type ImportMacOSResult struct {
	DryRun         bool
	Force          bool
	MacOSPath      string
	Lua            string
	ReadCount      int
	Added          int
	Warnings       []string
	EnsuredRequire bool
	HasSystemScope bool
	AdminNote      string
}

// ImportMacOS snapshots curated catalog defaults and merges them into macos.lua.
func ImportMacOS(ctx context.Context, opts ImportMacOSOptions) (ImportMacOSResult, error) {
	if opts.ConfigDir == "" || opts.ConfigPath == "" {
		return ImportMacOSResult{}, fmt.Errorf("config dir and path are required")
	}
	if opts.Runner == nil {
		return ImportMacOSResult{}, fmt.Errorf("defaults runner is required")
	}

	result := ImportMacOSResult{
		DryRun:    opts.DryRun,
		Force:     opts.Force,
		MacOSPath: filepath.Join(opts.ConfigDir, "macos.lua"),
	}

	desired := config.CatalogDesiredSettings()
	entries, warnings, err := discovery.SnapshotCatalogDefaults(ctx, opts.Runner, desired)
	if err != nil {
		return result, fmt.Errorf("snapshot macos defaults: %w", err)
	}
	result.Warnings = warnings
	result.ReadCount = len(entries)

	discovered := macOSDefaultsFromEntries(entries)
	result.HasSystemScope = hasSystemScope(entries, desired)
	if result.HasSystemScope {
		result.AdminNote = "system-scope keys (e.g. loginwindow) need admin privileges on apply"
	}

	existing := config.MacOSDefaults{}
	if _, err := os.Stat(opts.ConfigPath); err == nil {
		if m, loadErr := config.LoadManifest(opts.ConfigPath); loadErr == nil {
			existing = m.MacOS.Defaults
		}
	}

	merged, added := configimport.MergeMacOSDefaults(existing, discovered, opts.Force)
	result.Added = added
	body := configimport.FormatMacOSLua(merged)
	result.Lua = body

	if opts.DryRun {
		return result, nil
	}

	if _, err := os.Stat(opts.ConfigPath); err != nil {
		if os.IsNotExist(err) {
			return result, fmt.Errorf("no pourover.lua at %s: run pourover init", opts.ConfigPath)
		}
		return result, err
	}

	if err := os.WriteFile(result.MacOSPath, []byte(body), 0o644); err != nil {
		return result, err
	}

	rootBytes, err := os.ReadFile(opts.ConfigPath)
	if err != nil {
		return result, err
	}
	patched, changed := ensureMacOSRequire(string(rootBytes))
	if changed {
		if err := os.WriteFile(opts.ConfigPath, []byte(patched), 0o644); err != nil {
			return result, err
		}
		result.EnsuredRequire = true
	}

	return result, nil
}

func macOSDefaultsFromEntries(entries []discovery.SnapshotEntry) config.MacOSDefaults {
	d := config.MacOSDefaults{
		Sections: make(map[string]map[string]config.SettingValue),
	}
	for _, e := range entries {
		if d.Sections[e.Section] == nil {
			d.Sections[e.Section] = make(map[string]config.SettingValue)
		}
		d.Sections[e.Section][e.Key] = e.Value
	}
	return d
}

func hasSystemScope(entries []discovery.SnapshotEntry, desired []config.DesiredSetting) bool {
	scopeBySection := make(map[string]string, len(desired))
	for _, d := range desired {
		if d.Scope != "" {
			scopeBySection[d.Section] = d.Scope
		}
	}
	for _, e := range entries {
		if scopeBySection[e.Section] == "system" {
			return true
		}
	}
	return false
}

var (
	reRequireMacOS      = regexp.MustCompile(`require\s*\(\s*["']macos["']\s*\)`)
	reLocalPackagesReq  = regexp.MustCompile(`(?m)^local packages = require\(["']packages["']\)\s*$`)
	rePackagesTableField = regexp.MustCompile(`(?m)^(\s*)packages\s*=\s*packages\s*,?\s*$`)
	reReturnOpen         = regexp.MustCompile(`(?m)^return\s*\{\s*$`)
)

// ensureMacOSRequire surgically adds local macos = require("macos") and macos = macos
// in the return table when missing. Leaves files that already require macos intact.
func ensureMacOSRequire(src string) (string, bool) {
	if reRequireMacOS.MatchString(src) {
		return src, false
	}

	out := src
	if loc := reLocalPackagesReq.FindStringIndex(out); loc != nil {
		insert := loc[1]
		out = out[:insert] + "\nlocal macos = require(\"macos\")" + out[insert:]
	} else {
		out = "local macos = require(\"macos\")\n" + out
	}

	if loc := rePackagesTableField.FindStringIndex(out); loc != nil {
		line := out[loc[0]:loc[1]]
		indent := leadingWS(line)
		insert := loc[1]
		out = out[:insert] + "\n" + indent + "macos = macos," + out[insert:]
	} else if loc := reReturnOpen.FindStringIndex(out); loc != nil {
		insert := loc[1]
		out = out[:insert] + "\n  macos = macos," + out[insert:]
	} else {
		return src, false
	}
	return out, true
}

func leadingWS(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}
