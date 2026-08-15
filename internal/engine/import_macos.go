package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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

	existing := loadExistingMacOSDefaults(opts)
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

	rootBytes, err := os.ReadFile(opts.ConfigPath)
	if err != nil {
		return result, err
	}
	patched, changed, err := ensureMacOSRequire(string(rootBytes))
	if err != nil {
		return result, err
	}

	if err := os.WriteFile(result.MacOSPath, []byte(body), 0o644); err != nil {
		return result, err
	}
	if changed {
		if err := os.WriteFile(opts.ConfigPath, []byte(patched), 0o644); err != nil {
			return result, err
		}
		result.EnsuredRequire = true
	}

	return result, nil
}

func loadExistingMacOSDefaults(opts ImportMacOSOptions) config.MacOSDefaults {
	macosPath := filepath.Join(opts.ConfigDir, "macos.lua")
	if _, err := os.Stat(macosPath); err == nil {
		if d, err := config.LoadMacOSModule(macosPath); err == nil {
			return d
		}
	}
	if _, err := os.Stat(opts.ConfigPath); err == nil {
		if m, err := config.LoadManifest(opts.ConfigPath); err == nil {
			return m.MacOS.Defaults
		}
	}
	return config.MacOSDefaults{}
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
	reRequireMacOS       = regexp.MustCompile(`require\s*\(\s*["']macos["']\s*\)`)
	reMacOSTableField    = regexp.MustCompile(`^\s*macos\s*=\s*(macos\s*,?|require\s*\()`)
	reLocalPackagesReq   = regexp.MustCompile(`(?m)^local packages = require\(["']packages["']\)\s*$`)
	rePackagesTableField = regexp.MustCompile(`(?m)^(\s*)packages\s*=\s*packages\s*,?\s*$`)
	reReturnOpen         = regexp.MustCompile(`(?m)^return\s*\{\s*$`)
)

// ensureMacOSRequire surgically adds local macos = require("macos") and macos = macos
// in the return table when missing.
// Returns (src, false, nil) when already wired; (patched, true, nil) when changed;
// error when macos is not wired and cannot be patched.
func ensureMacOSRequire(src string) (string, bool, error) {
	if macOSAlreadyWired(src) {
		return src, false, nil
	}

	out := src
	if !hasActiveMacOSRequire(out) {
		if loc := reLocalPackagesReq.FindStringIndex(out); loc != nil {
			insert := loc[1]
			out = out[:insert] + "\nlocal macos = require(\"macos\")" + out[insert:]
		} else {
			out = "local macos = require(\"macos\")\n" + out
		}
	}

	if !hasActiveMacOSField(out) {
		if loc := rePackagesTableField.FindStringIndex(out); loc != nil {
			line := out[loc[0]:loc[1]]
			indent := leadingWS(line)
			insert := loc[1]
			out = out[:insert] + "\n" + indent + "macos = macos," + out[insert:]
		} else if loc := reReturnOpen.FindStringIndex(out); loc != nil {
			insert := loc[1]
			out = out[:insert] + "\n  macos = macos," + out[insert:]
		} else {
			return "", false, fmt.Errorf(`could not add require("macos") to pourover.lua; add manually`)
		}
	}

	if !macOSAlreadyWired(out) {
		return "", false, fmt.Errorf(`could not add require("macos") to pourover.lua; add manually`)
	}
	return out, true, nil
}

func macOSAlreadyWired(src string) bool {
	return hasActiveMacOSRequire(src) && hasActiveMacOSField(src)
}

func hasActiveMacOSRequire(src string) bool {
	for _, line := range activeLuaLines(src) {
		if reRequireMacOS.MatchString(line) {
			return true
		}
	}
	return false
}

func hasActiveMacOSField(src string) bool {
	for _, line := range activeLuaLines(src) {
		if reMacOSTableField.MatchString(line) {
			return true
		}
	}
	return false
}

func activeLuaLines(src string) []string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		code := line
		if i := strings.Index(code, "--"); i >= 0 {
			code = code[:i]
		}
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		out = append(out, code)
	}
	return out
}

func leadingWS(s string) string {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}
