package discovery

// BrewState is the set of Homebrew packages currently installed.
// Taps are not tracked in v1 (see plan D4).
type BrewState struct {
	Formulae []string
	Casks    []string
}
