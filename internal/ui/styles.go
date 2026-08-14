package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Coffee-forward palette (warm brown / cream — not purple-default AI CLI looks).
var (
	colorBrand  = lipgloss.Color("#A67C52")
	colorMuted  = lipgloss.Color("#8A7A68")
	colorAccent = lipgloss.Color("#D4A574")
	colorOK     = lipgloss.Color("#6B8E5A")
	colorFail   = lipgloss.Color("#C45C4A")
	colorBarOn  = lipgloss.Color("#A67C52")
	colorBarOff = lipgloss.Color("#4A4038")
)

var (
	styleBrand  = lipgloss.NewStyle().Bold(true).Foreground(colorBrand)
	styleMode   = lipgloss.NewStyle().Foreground(colorAccent)
	styleMuted  = lipgloss.NewStyle().Foreground(colorMuted)
	styleOK     = lipgloss.NewStyle().Foreground(colorOK)
	styleFail   = lipgloss.NewStyle().Foreground(colorFail)
	styleBarOn  = lipgloss.NewStyle().Foreground(colorBarOn)
	styleBarOff = lipgloss.NewStyle().Foreground(colorBarOff)
)

// ForcePlain disables ANSI styling for tests.
func ForcePlain() {
	lipgloss.SetColorProfile(termenv.Ascii)
	styleBrand = lipgloss.NewStyle().Bold(true)
	styleMode = lipgloss.NewStyle()
	styleMuted = lipgloss.NewStyle()
	styleOK = lipgloss.NewStyle()
	styleFail = lipgloss.NewStyle()
	styleBarOn = lipgloss.NewStyle()
	styleBarOff = lipgloss.NewStyle()
}
