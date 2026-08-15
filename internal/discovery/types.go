package discovery

import "strings"

// BrewState is the set of Homebrew taps and packages currently installed.
type BrewState struct {
	// Taps is every tapped repository (including homebrew/core and homebrew/cask).
	Taps []string
	// Formulae is every installed formula (including dependencies).
	Formulae []string
	// FormulaeRequested is formulae installed on request (not as dependencies).
	// Used for undeclared-package remove planning. Nil means "same as Formulae"
	// (test / fallback behavior).
	FormulaeRequested []string
	// Casks is every installed cask.
	Casks []string
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
