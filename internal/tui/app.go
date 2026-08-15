package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the PourOver TUI with the home model as root.
func Run() error {
	p := tea.NewProgram(NewHomeModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
