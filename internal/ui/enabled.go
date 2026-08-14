package ui

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Enabled reports whether fancy progress UI should run for w.
// False when quiet paths apply, NO_COLOR is set, or w is not a terminal.
func Enabled(w io.Writer, quiet bool) bool {
	if quiet {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
