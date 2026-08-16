package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/chasebank87/PourOver/internal/ui"
)

// Layout aliases of the shared CLI palette (no second hex set).
var (
	styleTitle    lipgloss.Style
	styleMuted    lipgloss.Style
	styleSummary  lipgloss.Style
	styleMenu     = lipgloss.NewStyle()
	styleSelected lipgloss.Style
	styleAccent   lipgloss.Style
	styleSuccess  lipgloss.Style
	styleWarning  lipgloss.Style
	styleFail     lipgloss.Style
)

func init() {
	styleTitle = ui.Brand()
	styleMuted = ui.Muted()
	styleSummary = ui.Accent()
	styleSelected = ui.Brand()
	styleAccent = ui.Accent()
	styleSuccess = ui.Success()
	styleWarning = ui.Warning()
	styleFail = ui.Fail()
}

func runSummaryStyle(summary string) lipgloss.Style {
	switch {
	case strings.Contains(summary, "failure"):
		return styleFail
	case strings.Contains(summary, "skipped"), strings.Contains(summary, "rename"):
		return styleWarning
	default:
		return styleSuccess
	}
}
