package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/state"
)

type historyItem struct {
	name  string
	path  string
	entry state.HistoryEntry
}

// HistoryModel lists apply history entries (newest first), with optional detail.
type HistoryModel struct {
	stateDir string
	home     HomeModel
	entries  []historyItem
	cursor   int
	detail   bool
	err      string
	loading  bool
	width    int
	height   int
}

// NewHistoryModel constructs a history list view, returning to home on esc.
func NewHistoryModel(home HomeModel) HistoryModel {
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return HistoryModel{home: home, err: err.Error()}
	}
	return HistoryModel{
		stateDir: stateDir,
		home:     home,
		loading:  true,
	}
}

type historyLoadedMsg struct {
	entries []historyItem
	err     error
}

func (m HistoryModel) Init() tea.Cmd {
	return loadHistoryCmd(m.stateDir)
}

func loadHistoryCmd(stateDir string) tea.Cmd {
	return func() tea.Msg {
		items, err := loadHistoryEntries(stateDir)
		return historyLoadedMsg{entries: items, err: err}
	}
}

func loadHistoryEntries(stateDir string) ([]historyItem, error) {
	dir := paths.HistoryDir(stateDir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var items []historyItem
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entry state.HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		items = append(items, historyItem{name: e.Name(), path: path, entry: entry})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].name > items[j].name
	})
	return items, nil
}

func (m HistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case historyLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.entries = nil
			return m, nil
		}
		m.err = ""
		m.entries = msg.entries
		if m.cursor >= len(m.entries) && len(m.entries) > 0 {
			m.cursor = len(m.entries) - 1
		}
		if len(m.entries) == 0 {
			m.cursor = 0
		}
		return m, nil

	case tea.KeyMsg:
		if m.detail {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "esc":
				m.detail = false
				return m, nil
			}
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
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 {
				m.detail = true
			}
		case "r":
			m.loading = true
			m.err = ""
			m.detail = false
			return m, loadHistoryCmd(m.stateDir)
		}
	}
	return m, nil
}

func (m HistoryModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("state: " + m.stateDir))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render("History"))
	b.WriteString("\n\n")

	if m.detail && len(m.entries) > 0 {
		b.WriteString(m.renderDetail())
		return b.String()
	}

	switch {
	case m.loading:
		b.WriteString(styleMuted.Render("loading history…"))
		b.WriteString("\n")
	case m.err != "":
		b.WriteString(styleFail.Render("error: " + m.err))
		b.WriteString("\n")
	case len(m.entries) == 0:
		b.WriteString(styleMuted.Render("no history"))
		b.WriteString("\n")
	default:
		for i, item := range m.entries {
			line := formatHistorySummary(item.entry)
			cursor := "  "
			if i == m.cursor {
				cursor = styleAccent.Render("> ")
				line = styleSelected.Render(line)
			} else {
				line = styleMenu.Render(line)
			}
			b.WriteString(cursor)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("↑/↓ or j/k · enter detail · r refresh · esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m HistoryModel) renderDetail() string {
	var b strings.Builder
	item := m.entries[m.cursor]
	e := item.entry

	status := "ok"
	if !e.Success {
		status = "FAIL"
	}
	statusLine := fmt.Sprintf("[%s] %s", status, e.Timestamp)
	if e.Success {
		b.WriteString(styleSuccess.Render(statusLine))
	} else {
		b.WriteString(styleFail.Render(statusLine))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("actions: %d\n", e.ActionCount))
	if e.ManifestHash != "" {
		b.WriteString(styleMuted.Render("manifest: " + e.ManifestHash))
		b.WriteString("\n")
	}
	if e.Error != "" {
		b.WriteString(styleFail.Render("error: " + e.Error))
		b.WriteString("\n")
	}
	if len(e.Actions) > 0 {
		b.WriteString("\n")
		for _, a := range e.Actions {
			b.WriteString(fmt.Sprintf("  %s %s\n", a.Type, a.Name))
		}
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}

func formatHistorySummary(e state.HistoryEntry) string {
	status := "ok"
	if !e.Success {
		status = "FAIL"
	}
	return fmt.Sprintf("[%s] %s — %d action(s)", status, e.Timestamp, e.ActionCount)
}
