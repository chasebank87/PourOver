package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
)

var _ engine.Confirmer = (*AsyncConfirmer)(nil)

// AsyncConfirmer bridges engine.Confirmer (sync) to Bubble Tea (async).
// Confirm blocks the apply goroutine until Answer is called from Update.
type AsyncConfirmer struct {
	ask chan string
	ans chan bool
}

// NewAsyncConfirmer creates a channel-backed confirmer for TUI apply runs.
func NewAsyncConfirmer() *AsyncConfirmer {
	return &AsyncConfirmer{
		ask: make(chan string),
		ans: make(chan bool),
	}
}

// Confirm implements engine.Confirmer.
func (c *AsyncConfirmer) Confirm(prompt string) bool {
	c.ask <- prompt
	return <-c.ans
}

// Answer unblocks a waiting Confirm call.
func (c *AsyncConfirmer) Answer(yes bool) {
	c.ans <- yes
}

type confirmNeededMsg struct {
	Prompt string
}

func waitConfirmCmd(c *AsyncConfirmer) tea.Cmd {
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		prompt, ok := <-c.ask
		if !ok {
			return nil
		}
		return confirmNeededMsg{Prompt: prompt}
	}
}

// ConfirmModel is a y/n modal for destructive prompts (e.g. uninstalls).
type ConfirmModel struct {
	Prompt string
	Active bool
}

// Update handles y/n while Active. answered is non-nil when the user answered.
func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, *bool) {
	if !m.Active {
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "y", "Y":
		yes := true
		m.Active = false
		return m, &yes
	case "n", "N":
		yes := false
		m.Active = false
		return m, &yes
	}
	return m, nil
}

// View renders the confirm prompt.
func (m ConfirmModel) View() string {
	if !m.Active {
		return ""
	}
	var b strings.Builder
	b.WriteString(styleSummary.Render("Confirm"))
	b.WriteString("\n")
	b.WriteString(m.Prompt)
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("y yes · n no"))
	b.WriteString("\n")
	return b.String()
}
