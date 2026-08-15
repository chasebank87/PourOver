package tui

import (
	"fmt"

	_ "github.com/charmbracelet/bubbles" // pinned for upcoming home menus
	tea "github.com/charmbracelet/bubbletea"
)

const placeholder = "PourOver TUI — home coming soon"

type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return placeholder + "\n\n(press q to quit)\n"
}

// Run starts the PourOver TUI.
func Run() error {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
