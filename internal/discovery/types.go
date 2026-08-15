package discovery

import "strings"

// BrewState is the set of Homebrew taps and packages currently installed.
type BrewState struct {
	// Taps is every tapped repository (including homebrew/core and homebrew/cask).
	Taps []string
	// TrustedTaps are non-official taps explicitly trusted via `brew trust`
	// (from `brew trust --json=v1`). Official taps are always trusted and may
	// be omitted from this list.
	TrustedTaps []string
	// Formulae is every installed formula (including dependencies).
	Formulae []string
	// FormulaeRequested is formulae installed on request (not as dependencies).
	// Used for undeclared-package remove planning. Nil means "same as Formulae"
	// (test / fallback behavior).
	FormulaeRequested []string
	// ProtectedFormulae are runtime dependencies of desired formulae (from
	// `brew deps --union`). They are never planned for removal even when also
	// installed-on-request. Matched by short name (tap prefix stripped).
	ProtectedFormulae []string
	// Casks is every installed cask.
	Casks []string
	// OutdatedFormulae / OutdatedCasks are set when DiscoverOutdated has run.
	// When nil, BuildUpgradePlan treats nothing as outdated (no upgrades).
	// When non-nil (including empty), only those names are upgrade candidates.
	OutdatedFormulae []string
	OutdatedCasks    []string
}

// RemovableFormulae returns formulae that may be considered for undeclared removal.
func (s BrewState) RemovableFormulae() []string {
	if s.FormulaeRequested != nil {
		return s.FormulaeRequested
	}
	return s.Formulae
}

// IsCoreTap reports whether name is a built-in Homebrew tap that must not be untapped.
func IsCoreTap(name string) bool {
	switch brewToken(name) {
	case "homebrew/core", "homebrew/cask":
		return true
	default:
		return false
	}
}

// IsOfficialTap reports taps under the homebrew org (always trusted by Homebrew).
func IsOfficialTap(name string) bool {
	t := brewToken(name)
	return t == "homebrew/core" || t == "homebrew/cask" || strings.HasPrefix(t, "homebrew/")
}

// NeedsExplicitTrust reports whether a tap requires `brew trust --tap` (Homebrew 6+).
func NeedsExplicitTrust(name string) bool {
	return !IsOfficialTap(name)
}

// RemovableTaps returns tapped repos that may be considered for undeclared removal
// (excludes homebrew/core and homebrew/cask).
func (s BrewState) RemovableTaps() []string {
	var out []string
	for _, name := range s.Taps {
		if IsCoreTap(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// DeclarableTaps returns taps suitable for packages.lua import (excludes core taps).
func DeclarableTaps(taps []string) []string {
	var out []string
	for _, name := range taps {
		if IsCoreTap(name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func brewToken(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// PackageKey normalizes a brew token for presence/remove matching: lowercase
// with tap prefix stripped (`mongodb/brew/foo` → `foo`).
func PackageKey(name string) string {
	t := brewToken(name)
	if i := strings.LastIndex(t, "/"); i >= 0 {
		return t[i+1:]
	}
	return t
}
