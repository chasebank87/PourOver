package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/configimport"
	"golang.org/x/term"
)

// FilePickItem is one row in the interactive file-import checklist.
type FilePickItem struct {
	Candidate configimport.FileCandidate
	Selected  bool
	Managed   bool // already in files.links
	IsDir     bool
}

// PickFileCandidates runs an interactive multi-select checklist on stdout/stderr.
// Returns the selected candidates (may be empty if the user confirms none).
// ErrPickAborted is returned when the user cancels.
func PickFileCandidates(items []FilePickItem) ([]configimport.FileCandidate, error) {
	if len(items) == 0 {
		return nil, nil
	}
	m := filePickModel{items: items}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin))
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	out, ok := final.(filePickModel)
	if !ok {
		return nil, fmt.Errorf("unexpected picker result type %T", final)
	}
	if out.aborted {
		return nil, ErrPickAborted
	}
	var chosen []configimport.FileCandidate
	for _, it := range out.items {
		if it.Selected {
			chosen = append(chosen, it.Candidate)
		}
	}
	return chosen, nil
}

// ErrPickAborted is returned when the user cancels the file picker.
var ErrPickAborted = fmt.Errorf("file selection cancelled")

// CanPickFiles reports whether stdin/stderr support an interactive picker.
func CanPickFiles() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stderr.Fd()))
}

// BuildFilePickItems prepares checklist rows with default selection rules.
func BuildFilePickItems(candidates []configimport.FileCandidate, managed map[string]struct{}) []FilePickItem {
	items := make([]FilePickItem, 0, len(candidates))
	for _, c := range candidates {
		_, isManaged := managed[c.TargetDecl]
		isDir := false
		if st, err := os.Lstat(c.TargetPath); err == nil {
			isDir = st.IsDir()
		}
		items = append(items, FilePickItem{
			Candidate: c,
			Selected:  configimport.DefaultFileCandidateSelected(c, isManaged),
			Managed:   isManaged,
			IsDir:     isDir,
		})
	}
	return items
}

type filePickModel struct {
	items   []FilePickItem
	cursor  int
	aborted bool
	done    bool
}

func (m filePickModel) Init() tea.Cmd { return nil }

func (m filePickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.aborted = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.items[m.cursor].Selected = !m.items[m.cursor].Selected
		case "a":
			for i := range m.items {
				m.items[i].Selected = true
			}
		case "n":
			for i := range m.items {
				m.items[i].Selected = false
			}
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m filePickModel) View() string {
	var b strings.Builder
	b.WriteString(Brand().Render("☕ PourOver"))
	b.WriteString("  ")
	b.WriteString(Accent().Render("import files"))
	b.WriteString("\n")
	b.WriteString(Muted().Render(strings.Repeat("─", 40)))
	b.WriteString("\n")
	b.WriteString(Muted().Render("Select paths to manage (space toggle). ~/.config apps are opt-in."))
	b.WriteString("\n\n")
	for i, it := range m.items {
		mark := "[ ]"
		if it.Selected {
			mark = "[x]"
		}
		kind := "file"
		if it.IsDir {
			kind = "dir "
		}
		extra := ""
		if it.Managed {
			extra = " " + Muted().Render("(managed)")
		}
		line := fmt.Sprintf("%s %s %s%s", mark, kind, it.Candidate.TargetDecl, extra)
		prefix := "  "
		if i == m.cursor {
			prefix = Accent().Render("> ")
			line = Brand().Render(fmt.Sprintf("%s %s %s", mark, kind, it.Candidate.TargetDecl)) + extra
		}
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(Muted().Render("↑/↓ · space · a all · n none · enter confirm · esc cancel"))
	b.WriteString("\n")
	return b.String()
}

// PrintFilePickSummary writes a plain list of chosen targets (for dry-run / logs).
func PrintFilePickSummary(w io.Writer, chosen []configimport.FileCandidate) {
	for _, c := range chosen {
		fmt.Fprintf(w, "  %s\n", c.TargetDecl)
	}
}
