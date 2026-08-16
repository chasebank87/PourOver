package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configimport"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/ui"
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

type importPhase int

const (
	importPhaseOptions importPhase = iota
	importPhasePick
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

	phase    importPhase
	pick     []ui.FilePickItem
	pickCur  int
	managed  map[string]struct{}

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
		phase:      importPhaseOptions,
	}
}

type importDoneMsg struct {
	result engine.ImportResult
	err    error
}

type importCandidatesMsg struct {
	items []ui.FilePickItem
	err   error
}

func (m ImportModel) Init() tea.Cmd {
	return nil
}

func loadImportCandidates(configPath, home string, force bool) tea.Cmd {
	return func() tea.Msg {
		managed := map[string]struct{}{}
		if man, err := config.LoadManifest(configPath); err == nil {
			for _, l := range man.Files.Links {
				managed[l.Target] = struct{}{}
			}
		}
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return importCandidatesMsg{err: err}
			}
		}
		cands, err := configimport.ExistingImportable(configimport.DefaultHomeCandidates(home))
		if err != nil {
			return importCandidatesMsg{err: err}
		}
		var selectable []configimport.FileCandidate
		for _, c := range cands {
			if !force {
				if _, ok := managed[c.TargetDecl]; ok {
					continue
				}
			}
			selectable = append(selectable, c)
		}
		return importCandidatesMsg{items: ui.BuildFilePickItems(selectable, managed)}
	}
}

func runImportCmd(configDir, configPath string, packages, files, force, dryRun bool, fileTargets []string, filesAll bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			if !dryRun {
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
			ConfigDir:   configDir,
			ConfigPath:  configPath,
			Packages:    packages,
			Files:       files,
			DryRun:      dryRun,
			Force:       force,
			FileTargets: fileTargets,
			FilesAll:    filesAll,
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

	case importCandidatesMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		if len(msg.items) == 0 {
			// Nothing new to pick; still run packages-only / empty files.
			m.busy = true
			return m, runImportCmd(m.configDir, m.configPath, m.packages, m.files, m.force, m.dryRun, nil, true)
		}
		m.phase = importPhasePick
		m.pick = msg.items
		m.pickCur = 0
		m.err = ""
		return m, nil

	case importDoneMsg:
		m.busy = false
		m.phase = importPhaseOptions
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
		if m.phase == importPhasePick {
			return m.updatePick(msg)
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

func (m ImportModel) updatePick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.phase = importPhaseOptions
		m.pick = nil
		return m, nil
	case "up", "k":
		if m.pickCur > 0 {
			m.pickCur--
		}
	case "down", "j":
		if m.pickCur < len(m.pick)-1 {
			m.pickCur++
		}
	case " ":
		m.pick[m.pickCur].Selected = !m.pick[m.pickCur].Selected
	case "a":
		for i := range m.pick {
			m.pick[i].Selected = true
		}
	case "n":
		for i := range m.pick {
			m.pick[i].Selected = false
		}
	case "enter":
		var targets []string
		for _, it := range m.pick {
			if it.Selected {
				targets = append(targets, it.Candidate.TargetDecl)
			}
		}
		m.busy = true
		m.err = ""
		m.status = ""
		// Empty selection: still run (packages may be on); files with empty targets
		// means no new file imports when FileTargets is empty and FilesAll false —
		// pass FilesAll false and FileTargets empty only if files disabled.
		if m.files && len(targets) == 0 {
			return m, runImportCmd(m.configDir, m.configPath, m.packages, false, m.force, m.dryRun, nil, false)
		}
		return m, runImportCmd(m.configDir, m.configPath, m.packages, m.files, m.force, m.dryRun, targets, false)
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
		m.err = ""
		m.status = ""
		if m.files {
			m.busy = true
			return m, loadImportCandidates(m.configPath, "", m.force)
		}
		m.busy = true
		return m, runImportCmd(m.configDir, m.configPath, m.packages, false, m.force, m.dryRun, nil, false)
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
		if n := len(r.AddedLinks); n > 0 {
			fmt.Fprintf(&b, "added: %d link(s)", n)
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
		b.WriteString(styleMuted.Render("working…"))
		b.WriteString("\n")
	} else if m.phase == importPhasePick {
		b.WriteString(styleMuted.Render("Select paths to manage (~/.config is opt-in)"))
		b.WriteString("\n\n")
		for i, it := range m.pick {
			mark := "[ ]"
			if it.Selected {
				mark = "[x]"
			}
			kind := "file"
			if it.IsDir {
				kind = "dir"
			}
			line := fmt.Sprintf("%s %s %s", mark, kind, it.Candidate.TargetDecl)
			if it.Managed {
				line += " (managed)"
			}
			cursor := "  "
			if i == m.pickCur {
				cursor = styleAccent.Render("> ")
				line = styleSelected.Render(line)
			} else {
				line = styleMenu.Render(line)
			}
			b.WriteString(cursor + line + "\n")
		}
	} else {
		b.WriteString(m.renderToggle(importOptPackages, "Packages", m.packages))
		b.WriteString(m.renderToggle(importOptFiles, "Files", m.files))
		b.WriteString(m.renderToggle(importOptForce, "Force", m.force))
		b.WriteString(m.renderToggle(importOptDryRun, "Dry-run", m.dryRun))
		b.WriteString(m.renderRun())
	}

	if m.status != "" && !m.busy && m.phase == importPhaseOptions {
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
	if m.phase == importPhasePick {
		return "↑/↓ · space · a all · n none · enter import · esc back · q quit"
	}
	return "↑/↓ or j/k · space/enter toggle · enter run · esc back · q quit"
}
