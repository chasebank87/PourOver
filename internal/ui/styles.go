package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Coffee-forward palette (warm brown / cream — not purple-default AI CLI looks).
var (
	colorBrand   = lipgloss.Color("#A67C52")
	colorMuted   = lipgloss.Color("#8A7A68")
	colorAccent  = lipgloss.Color("#D4A574")
	colorSuccess = lipgloss.Color("#6B8E5A")
	colorWarning = lipgloss.Color("#C9843A")
	colorFail    = lipgloss.Color("#C45C4A")
	colorBarOn   = lipgloss.Color("#A67C52")
	colorBarOff  = lipgloss.Color("#4A4038")
)

var (
	styleBrand        lipgloss.Style
	styleMode         lipgloss.Style
	styleMuted        lipgloss.Style
	styleOK           lipgloss.Style
	styleWarning      lipgloss.Style
	styleFail         lipgloss.Style
	styleBarOn        lipgloss.Style
	styleBarOff       lipgloss.Style
	styleAccentPrompt lipgloss.Style
)

func init() {
	applyColorStyles()
}

func applyColorStyles() {
	styleBrand = lipgloss.NewStyle().Bold(true).Foreground(colorBrand)
	styleMode = lipgloss.NewStyle().Foreground(colorAccent)
	styleMuted = lipgloss.NewStyle().Foreground(colorMuted)
	styleOK = lipgloss.NewStyle().Foreground(colorSuccess)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning)
	styleFail = lipgloss.NewStyle().Foreground(colorFail)
	styleBarOn = lipgloss.NewStyle().Foreground(colorBarOn)
	styleBarOff = lipgloss.NewStyle().Foreground(colorBarOff)
	styleAccentPrompt = lipgloss.NewStyle().Foreground(colorAccent)
}

// Brand is the coffee title style (headers, selected rows).
func Brand() lipgloss.Style { return styleBrand }

// Muted is chrome, hints, and pending plan lines.
func Muted() lipgloss.Style { return styleMuted }

// Accent is in-progress chrome (phase, cursor), not a status color.
func Accent() lipgloss.Style { return styleMode }

// Success is completed work (green).
func Success() lipgloss.Style { return styleOK }

// Warning is advisory / skipped / drift (orange).
func Warning() lipgloss.Style { return styleWarning }

// Fail is errors and failed checks (red).
func Fail() lipgloss.Style { return styleFail }

// ForcePlain disables ANSI styling for tests.
func ForcePlain() {
	lipgloss.SetColorProfile(termenv.Ascii)
	styleBrand = lipgloss.NewStyle().Bold(true)
	styleMode = lipgloss.NewStyle()
	styleMuted = lipgloss.NewStyle()
	styleOK = lipgloss.NewStyle()
	styleWarning = lipgloss.NewStyle()
	styleFail = lipgloss.NewStyle()
	styleBarOn = lipgloss.NewStyle()
	styleBarOff = lipgloss.NewStyle()
	styleAccentPrompt = lipgloss.NewStyle()
}
