package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

// UnmanageModel lists files.links and removes selected targets from management.
type UnmanageModel struct {
	configPath string
	home       HomeModel

	links   []config.FileLink
	sel     []bool
	cursor  int
	busy    bool
	status  string
	err     string
	width   int
	height  int
}

// NewUnmanageModel constructs the unmanage view.
func NewUnmanageModel(configPath string, home HomeModel) UnmanageModel {
	return UnmanageModel{configPath: configPath, home: home}
}

type unmanageLoadMsg struct {
	links []config.FileLink
	err   error
}

type unmanageDoneMsg struct {
	result engine.UnmanageFilesResult
	err    error
}

func (m UnmanageModel) Init() tea.Cmd {
	return loadUnmanageLinks(m.configPath)
}

func loadUnmanageLinks(configPath string) tea.Cmd {
	return func() tea.Msg {
		man, err := config.LoadManifest(configPath)
		if err != nil {
			return unmanageLoadMsg{err: err}
		}
		return unmanageLoadMsg{links: append([]config.FileLink(nil), man.Files.Links...)}
	}
}

func runUnmanageCmd(configPath string, targets []string) tea.Cmd {
	return func() tea.Msg {
		stateDir, err := paths.DefaultStateDir()
		if err != nil {
			return unmanageDoneMsg{err: err}
		}
		result, err := engine.UnmanageFiles(engine.UnmanageFilesOptions{
			ConfigPath: configPath,
			StateDir:   stateDir,
			Targets:    targets,
		})
		return unmanageDoneMsg{result: result, err: err}
	}
}

func (m UnmanageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case unmanageLoadMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.links = msg.links
		m.sel = make([]bool, len(m.links))
		m.err = ""
		return m, nil
	case unmanageDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		var b strings.Builder
		for _, l := range msg.result.RemovedLinks {
			fmt.Fprintf(&b, "unmanaged %s\n", l.Target)
		}
		if n := len(msg.result.ClearedOwned); n > 0 {
			fmt.Fprintf(&b, "cleared %d owned path(s)\n", n)
		}
		m.status = strings.TrimRight(b.String(), "\n")
		m.err = ""
		return m, loadUnmanageLinks(m.configPath)
	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.links)-1 {
				m.cursor++
			}
		case " ":
			if len(m.sel) == len(m.links) && m.cursor < len(m.sel) {
				m.sel[m.cursor] = !m.sel[m.cursor]
			}
		case "enter":
			var targets []string
			for i, on := range m.sel {
				if on && i < len(m.links) {
					targets = append(targets, m.links[i].Target)
				}
			}
			if len(targets) == 0 {
				m.err = "select at least one link (space)"
				return m, nil
			}
			m.busy = true
			m.err = ""
			m.status = ""
			return m, runUnmanageCmd(m.configPath, targets)
		}
	}
	return m, nil
}

func (m UnmanageModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render("Unmanage files"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Stop managing paths without deleting live files."))
	b.WriteString("\n\n")

	if m.busy {
		b.WriteString(styleMuted.Render("updating…"))
		b.WriteString("\n")
	} else if len(m.links) == 0 {
		b.WriteString(styleMuted.Render("No files.links declared."))
		b.WriteString("\n")
	} else {
		for i, link := range m.links {
			mark := "[ ]"
			if i < len(m.sel) && m.sel[i] {
				mark = "[x]"
			}
			line := fmt.Sprintf("%s %s", mark, link.Target)
			cursor := "  "
			if i == m.cursor {
				cursor = styleAccent.Render("> ")
				line = styleSelected.Render(line)
			} else {
				line = styleMenu.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
	}

	if m.status != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleSummary.Render(m.status))
		b.WriteString("\n")
	}
	if m.err != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleFail.Render("error: " + m.err))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("↑/↓ · space select · enter unmanage · esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}
