package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Header prints the standard PourOver command chrome: brand, mode, rule.
// Colors apply only when Enabled(w, false).
func Header(w io.Writer, mode string) {
	brand := "☕ PourOver"
	rule := strings.Repeat("─", 40)
	if Enabled(w, false) {
		brand = Brand().Render(brand)
		mode = Accent().Render(mode)
		rule = Muted().Render(rule)
	}
	fmt.Fprintln(w, brand+"  "+mode)
	fmt.Fprintln(w, rule)
}

// Successf prints a success line (green when Enabled).
func Successf(w io.Writer, format string, args ...any) {
	printfStyled(w, Success(), format, args...)
}

// Warnf prints a warning line (orange when Enabled).
func Warnf(w io.Writer, format string, args ...any) {
	printfStyled(w, Warning(), format, args...)
}

// Failf prints a failure line (red when Enabled).
func Failf(w io.Writer, format string, args ...any) {
	printfStyled(w, Fail(), format, args...)
}

// Mutedf prints a muted chrome/info line when Enabled.
func Mutedf(w io.Writer, format string, args ...any) {
	printfStyled(w, Muted(), format, args...)
}

// Errorf prints "Error: …" on w (red when Enabled). Used by the CLI root.
func Errorf(w io.Writer, err error) {
	if err == nil {
		return
	}
	Failf(w, "Error: %v", err)
}

func printfStyled(w io.Writer, style lipgloss.Style, format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	if Enabled(w, false) {
		line = style.Render(line)
	}
	fmt.Fprintln(w, line)
}
