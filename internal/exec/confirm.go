package exec

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

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
