package exec

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/chasebank87/PourOver/internal/paths"
)

const maxConfirmList = 20

// ConfirmYes prints prompt to out and returns true if the user answers yes/y.
// Empty input, EOF, and any other answer are treated as no.
func ConfirmYes(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}

// FormatListPrompt builds a multi-line y/n prompt: title, one item per line, then "Proceed?".
// Paths under $HOME are shown with ~. Long lists are truncated.
func FormatListPrompt(title string, items []string) string {
	var b strings.Builder
	label := "item"
	if len(items) != 1 {
		label = "items"
	}
	fmt.Fprintf(&b, "%s (%d %s):\n", title, len(items), label)
	n := len(items)
	extra := 0
	if n > maxConfirmList {
		extra = n - maxConfirmList
		n = maxConfirmList
	}
	for _, item := range items[:n] {
		fmt.Fprintf(&b, "  %s\n", paths.DisplayHome(item))
	}
	if extra > 0 {
		fmt.Fprintf(&b, "  … and %d more\n", extra)
	}
	b.WriteString("Proceed?")
	return b.String()
}
