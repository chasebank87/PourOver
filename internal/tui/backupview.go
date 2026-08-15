package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

type backupTab int

const (
	backupTabBackup backupTab = iota
	backupTabRestore
)

type snapshotItem struct {
	name string
	path string
}

// BackupModel offers Backup action and Restore with local snapshot selection.
type BackupModel struct {
	configPath string
	stateDir   string
	home       HomeModel

	tab       backupTab
	snapshots []snapshotItem
	cursor    int

	confirm ConfirmModel
	busy    bool
	status  string
	err     string
	loading bool

	width  int
	height int
}

// NewBackupModel constructs a backup/restore view, returning to home on esc when idle.
func NewBackupModel(configPath string, home HomeModel) BackupModel {
	stateDir, err := paths.DefaultStateDir()
	m := BackupModel{
		configPath: configPath,
		home:       home,
		loading:    true,
	}
	if err != nil {
		m.err = err.Error()
		m.loading = false
		return m
	}
	m.stateDir = stateDir
	return m
}

type snapshotsLoadedMsg struct {
	snapshots []snapshotItem
	err       error
}

type backupDoneMsg struct {
	result engine.BackupResult
	err    error
}

type restoreDoneMsg struct {
	result engine.RestoreResult
	err    error
}

func (m BackupModel) Init() tea.Cmd {
	return loadSnapshotsCmd(m.stateDir)
}

func loadSnapshotsCmd(stateDir string) tea.Cmd {
	return func() tea.Msg {
		items, err := listSnapshots(stateDir)
		return snapshotsLoadedMsg{snapshots: items, err: err}
	}
}

func listSnapshots(stateDir string) ([]snapshotItem, error) {
	dir := paths.SnapshotsDir(stateDir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	items := make([]snapshotItem, 0, len(names))
	for _, name := range names {
		items = append(items, snapshotItem{
			name: name,
			path: filepath.Join(dir, name),
		})
	}
	return items, nil
}

func runBackupCmd(configPath, stateDir string) tea.Cmd {
	return func() tea.Msg {
		manifest, err := config.LoadManifest(configPath)
		if err != nil {
			return backupDoneMsg{err: fmt.Errorf("load config: %w", err)}
		}
		result, err := engine.Backup(context.Background(), engine.BackupOptions{
			StateDir: stateDir,
			Manifest: manifest,
		})
		return backupDoneMsg{result: result, err: err}
	}
}

func runRestoreCmd(configPath, stateDir, snapshot string) tea.Cmd {
	return func() tea.Msg {
		var manifest config.Manifest
		if m, err := config.LoadManifest(configPath); err == nil {
			manifest = m
		}
		result, err := engine.Restore(context.Background(), engine.RestoreOptions{
			StateDir: stateDir,
			Manifest: manifest,
			Snapshot: snapshot,
		})
		return restoreDoneMsg{result: result, err: err}
	}
}

func (m BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case snapshotsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.snapshots = nil
			return m, nil
		}
		m.err = ""
		m.snapshots = msg.snapshots
		if m.cursor >= len(m.snapshots) && len(m.snapshots) > 0 {
			m.cursor = len(m.snapshots) - 1
		}
		if len(m.snapshots) == 0 {
			m.cursor = 0
		}
		return m, nil

	case backupDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.err = ""
		m.status = formatBackupResult(msg.result)
		return m, loadSnapshotsCmd(m.stateDir)

	case restoreDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.status = ""
			return m, nil
		}
		m.err = ""
		m.status = fmt.Sprintf("Restored from %s into %s", msg.result.SnapshotPath, msg.result.StateDir)
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}

		if m.confirm.Active {
			switch msg.String() {
			case "esc":
				m.confirm.Active = false
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			var answered *bool
			m.confirm, answered = m.confirm.Update(msg)
			if answered == nil {
				return m, nil
			}
			if !*answered {
				return m, nil
			}
			if m.cursor < 0 || m.cursor >= len(m.snapshots) {
				return m, nil
			}
			snap := m.snapshots[m.cursor]
			m.busy = true
			m.err = ""
			m.status = ""
			return m, runRestoreCmd(m.configPath, m.stateDir, snap.path)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "tab", "right", "l":
			if m.tab == backupTabBackup {
				m.tab = backupTabRestore
			} else {
				m.tab = backupTabBackup
			}
			return m, nil
		case "left", "h":
			if m.tab == backupTabRestore {
				m.tab = backupTabBackup
			} else {
				m.tab = backupTabRestore
			}
			return m, nil
		case "up", "k":
			if m.tab == backupTabRestore && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.tab == backupTabRestore && m.cursor < len(m.snapshots)-1 {
				m.cursor++
			}
		case "enter":
			return m.activate()
		case "r":
			m.loading = true
			m.err = ""
			return m, loadSnapshotsCmd(m.stateDir)
		}
	}
	return m, nil
}

func (m BackupModel) activate() (tea.Model, tea.Cmd) {
	switch m.tab {
	case backupTabBackup:
		m.busy = true
		m.err = ""
		m.status = ""
		return m, runBackupCmd(m.configPath, m.stateDir)
	case backupTabRestore:
		if len(m.snapshots) == 0 {
			return m, nil
		}
		snap := m.snapshots[m.cursor]
		m.confirm = ConfirmModel{
			Prompt: fmt.Sprintf("Restore state from %s?", snap.name),
			Active: true,
		}
		return m, nil
	}
	return m, nil
}

func formatBackupResult(r engine.BackupResult) string {
	var b strings.Builder
	b.WriteString("Snapshot: ")
	b.WriteString(r.LocalSnapshot)
	if r.MirroredTo != "" {
		b.WriteString("\nMirrored to: ")
		b.WriteString(r.MirroredTo)
	} else if r.ICloudEnabled {
		b.WriteString("\nMirrored to: skipped (path unavailable)")
	}
	return b.String()
}

func (m BackupModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	if m.stateDir != "" {
		b.WriteString(styleMuted.Render("state: " + m.stateDir))
		b.WriteString("\n")
	}
	b.WriteString(styleSummary.Render("Backup / Restore"))
	b.WriteString("\n\n")

	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	if m.busy {
		b.WriteString(styleMuted.Render("working…"))
		b.WriteString("\n")
	} else if m.confirm.Active {
		b.WriteString(m.confirm.View())
	} else {
		switch m.tab {
		case backupTabBackup:
			b.WriteString(m.renderBackupTab())
		case backupTabRestore:
			b.WriteString(m.renderRestoreTab())
		}
	}

	if m.status != "" && !m.confirm.Active {
		b.WriteString("\n")
		b.WriteString(styleSummary.Render(m.status))
		b.WriteString("\n")
	}
	if m.err != "" && !m.confirm.Active && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleAccent.Render("error: " + m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render(m.footer()))
	b.WriteString("\n")
	return b.String()
}

func (m BackupModel) renderTabs() string {
	backupLabel := "Backup"
	restoreLabel := "Restore"
	if m.tab == backupTabBackup {
		backupLabel = styleSelected.Render("[Backup]")
		restoreLabel = styleMenu.Render("Restore")
	} else {
		backupLabel = styleMenu.Render("Backup")
		restoreLabel = styleSelected.Render("[Restore]")
	}
	return backupLabel + "  " + restoreLabel
}

func (m BackupModel) renderBackupTab() string {
	var b strings.Builder
	b.WriteString(styleMenu.Render("> Run backup"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Creates a local snapshot and mirrors to iCloud when enabled."))
	b.WriteString("\n")
	return b.String()
}

func (m BackupModel) renderRestoreTab() string {
	var b strings.Builder
	switch {
	case m.loading:
		b.WriteString(styleMuted.Render("loading snapshots…"))
		b.WriteString("\n")
	case len(m.snapshots) == 0:
		b.WriteString(styleMuted.Render("no snapshots"))
		b.WriteString("\n")
	default:
		for i, item := range m.snapshots {
			line := item.name
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
	return b.String()
}

func (m BackupModel) footer() string {
	if m.busy {
		return "please wait…"
	}
	if m.confirm.Active {
		return "y yes · n no · esc cancel"
	}
	if m.tab == backupTabBackup {
		return "tab switch · enter run backup · esc back · q quit"
	}
	return "↑/↓ or j/k · enter restore · tab switch · r refresh · esc back · q quit"
}
