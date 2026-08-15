package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Coffee-forward palette aligned with internal/ui (warm brown / cream).
var (
	colorBrand  = lipgloss.Color("#A67C52")
	colorMuted  = lipgloss.Color("#8A7A68")
	colorAccent = lipgloss.Color("#D4A574")
)

var (
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(colorBrand)
	styleMuted    = lipgloss.NewStyle().Foreground(colorMuted)
	styleSummary  = lipgloss.NewStyle().Foreground(colorAccent)
	styleMenu     = lipgloss.NewStyle()
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(colorBrand)
	styleAccent   = lipgloss.NewStyle().Foreground(colorAccent)
)
