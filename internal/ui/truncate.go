package ui

import (
	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const defaultTermCols = 80

// terminalColumns returns the width of the writer's terminal, or a fallback.
func terminalColumns(w interface{ Fd() uintptr }) int {
	cols, _, err := term.GetSize(int(w.Fd()))
	if err != nil || cols <= 0 {
		return defaultTermCols
	}
	return cols
}

func writerColumns(out interface{}) int {
	type fder interface{ Fd() uintptr }
	if f, ok := out.(fder); ok {
		return terminalColumns(f)
	}
	return defaultTermCols
}

// truncateDisplay shortens s to at most max display columns, appending tail.
// ANSI sequences are preserved; max <= 0 yields "".
func truncateDisplay(s string, max int, tail string) string {
	if max <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= max {
		return s
	}
	return ansi.Truncate(s, max, tail)
}
