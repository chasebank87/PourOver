package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/selfupdate"
)

// SelfUpdateModel runs engine.SelfUpdate and shows the result.
// Busy gate ignores keys while downloading; esc returns home when idle.
type SelfUpdateModel struct {
	home HomeModel

	busy   bool
	result selfupdate.Result
	err    string
	done   bool

	width  int
	height int
}

// NewSelfUpdateModel constructs a self-update view that starts checking immediately.
func NewSelfUpdateModel(home HomeModel) SelfUpdateModel {
	return SelfUpdateModel{
		home: home,
		busy: true,
	}
}

type selfUpdateDoneMsg struct {
	result selfupdate.Result
	err    error
}

func (m SelfUpdateModel) Init() tea.Cmd {
	return runSelfUpdateCmd()
}

func runSelfUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		res, err := engine.SelfUpdate(selfupdate.Options{})
		return selfUpdateDoneMsg{result: res, err: err}
	}
}

func (m SelfUpdateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case selfUpdateDoneMsg:
		m.busy = false
		m.done = true
		if msg.err != nil {
			m.err = msg.err.Error()
			m.result = selfupdate.Result{}
			return m, nil
		}
		m.err = ""
		m.result = msg.result
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "r":
			m.busy = true
			m.done = false
			m.err = ""
			m.result = selfupdate.Result{}
			return m, runSelfUpdateCmd()
		}
	}
	return m, nil
}

func (m SelfUpdateModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleSummary.Render("Self-update"))
	b.WriteString("\n\n")

	switch {
	case m.busy:
		b.WriteString(styleMuted.Render("checking for updates…"))
		b.WriteString("\n")
	case m.err != "":
		b.WriteString(styleFail.Render("error: " + m.err))
		b.WriteString("\n")
	case m.done:
		b.WriteString(styleSummary.Render(formatSelfUpdateResult(m.result)))
		b.WriteString("\n")
	default:
		b.WriteString(styleMuted.Render("ready"))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.busy {
		b.WriteString(styleMuted.Render("updating…"))
	} else {
		b.WriteString(styleMuted.Render("r retry · esc back · q quit"))
	}
	b.WriteString("\n")
	return b.String()
}

func formatSelfUpdateResult(r selfupdate.Result) string {
	updated := "no"
	if r.Updated {
		updated = "yes"
	}
	return fmt.Sprintf("Updated: %s\nCurrentTag: %s\nLatestTag: %s", updated, r.CurrentTag, r.LatestTag)
}
