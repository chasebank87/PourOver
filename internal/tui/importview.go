package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
)

type importOpt int

const (
	importOptPackages importOpt = iota
	importOptFiles
	importOptForce
	importOptDryRun
	importOptRun
	importOptCount
)

// ImportModel offers CLI-mirrored import flags and runs engine.Import.
type ImportModel struct {
	configPath string
	configDir  string
	home       HomeModel

	packages bool
	files    bool
	force    bool
	dryRun   bool
	cursor   importOpt

	busy   bool
	status string
	err    string

	width  int
	height int
}

// NewImportModel constructs an import view, returning to home on esc when idle.
func NewImportModel(configPath string, home HomeModel) ImportModel {
	return ImportModel{
		configPath: configPath,
		configDir:  filepath.Dir(configPath),
		home:       home,
		packages:   true,
		files:      true,
		force:      false,
		dryRun:     false,
		cursor:     importOptPackages,
	}
}

type importDoneMsg struct {
	result engine.ImportResult
	err    error
}

func (m ImportModel) Init() tea.Cmd {
	return nil
}

func runImportCmd(configDir, configPath string, packages, files, force, dryRun bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if !dryRun {
				// Avoid importing cli/commands (cobra); match CLI intent with a clear prompt.
				return importDoneMsg{err: fmt.Errorf("config missing — run pourover init")}
			}
		} else if err != nil {
			return importDoneMsg{err: err}
		}

		var runner discovery.Runner
		if packages {
			runner = discovery.NewExecRunner()
		}
		result, err := engine.Import(context.Background(), runner, engine.ImportOptions{
			ConfigDir:  configDir,
			ConfigPath: configPath,
			Packages:   packages,
			Files:      files,
			DryRun:     dryRun,
			Force:      force,
		})
		return importDoneMsg{result: result, err: err}
	}
}

func (m ImportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case importDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.err = ""
		m.status = formatImportResult(msg.result)
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < importOptCount-1 {
				m.cursor++
			}
		case " ":
			if m.cursor != importOptRun {
				return m.activate()
			}
		case "enter":
			return m.activate()
		}
	}
	return m, nil
}

func (m ImportModel) activate() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case importOptPackages:
		m.packages = !m.packages
		return m, nil
	case importOptFiles:
		m.files = !m.files
		return m, nil
	case importOptForce:
		m.force = !m.force
		return m, nil
	case importOptDryRun:
		m.dryRun = !m.dryRun
		return m, nil
	case importOptRun:
		m.busy = true
		m.err = ""
		m.status = ""
		return m, runImportCmd(m.configDir, m.configPath, m.packages, m.files, m.force, m.dryRun)
	}
	return m, nil
}

func formatImportResult(r engine.ImportResult) string {
	var b strings.Builder
	if r.PackagesDone {
		pkgCount := len(r.Taps) + len(r.Formulae) + len(r.Casks)
		fmt.Fprintf(&b, "packages: %d (%d taps, %d formulae, %d casks)",
			pkgCount, len(r.Taps), len(r.Formulae), len(r.Casks))
		b.WriteString("\n")
	}
	if r.FilesDone {
		fmt.Fprintf(&b, "files: %d link(s)", len(r.Links))
		b.WriteString("\n")
		if n := len(r.SkippedLinks); n > 0 {
			fmt.Fprintf(&b, "skipped: %d existing link(s)", n)
			b.WriteString("\n")
		}
	}
	if r.DryRun {
		b.WriteString("Dry run only; no files were modified.")
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m ImportModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render("Import"))
	b.WriteString("\n\n")

	if m.busy {
		b.WriteString(styleMuted.Render("importing…"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderToggle(importOptPackages, "Packages", m.packages))
		b.WriteString(m.renderToggle(importOptFiles, "Files", m.files))
		b.WriteString(m.renderToggle(importOptForce, "Force", m.force))
		b.WriteString(m.renderToggle(importOptDryRun, "Dry-run", m.dryRun))
		b.WriteString(m.renderRun())
	}

	if m.status != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleSummary.Render(m.status))
		b.WriteString("\n")
	}
	if m.err != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleAccent.Render("error: " + m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render(m.footer()))
	b.WriteString("\n")
	return b.String()
}

func (m ImportModel) renderToggle(opt importOpt, label string, on bool) string {
	mark := "[ ]"
	if on {
		mark = "[x]"
	}
	line := mark + " " + label
	cursor := "  "
	if m.cursor == opt {
		cursor = styleAccent.Render("> ")
		line = styleSelected.Render(line)
	} else {
		line = styleMenu.Render(line)
	}
	return cursor + line + "\n"
}

func (m ImportModel) renderRun() string {
	line := "Run import"
	cursor := "  "
	if m.cursor == importOptRun {
		cursor = styleAccent.Render("> ")
		line = styleSelected.Render(line)
	} else {
		line = styleMenu.Render(line)
	}
	return cursor + line + "\n"
}

func (m ImportModel) footer() string {
	if m.busy {
		return "please wait…"
	}
	return "↑/↓ or j/k · space/enter toggle · enter run · esc back · q quit"
}
