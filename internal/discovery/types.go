package discovery

// BrewState is the set of Homebrew packages currently installed.
// Taps are not tracked in v1 (see plan D4).
type BrewState struct {
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
