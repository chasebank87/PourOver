package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

func TestBackupView_TabsAndHints(t *testing.T) {
	t.Parallel()

	m := BackupModel{tab: backupTabBackup}
	view := m.View()
	if !strings.Contains(view, "Backup") || !strings.Contains(view, "Restore") {
		t.Fatalf("View() = %q, want Backup/Restore tabs", view)
	}
	if !strings.Contains(view, "Run backup") {
		t.Fatalf("View() = %q, want backup action hint", view)
	}

	m.tab = backupTabRestore
	view = m.View()
	if !strings.Contains(view, "no snapshots") {
		t.Fatalf("empty restore View() = %q, want empty-state message", view)
	}
}

func TestBackupView_ListsSnapshotsNewestFirst(t *testing.T) {
	t.Parallel()

	m := BackupModel{
		tab: backupTabRestore,
		snapshots: []snapshotItem{
			{name: "2026-08-15T12-00-00Z", path: "/s/new"},
			{name: "2026-08-14T12-00-00Z", path: "/s/old"},
		},
	}
	view := m.View()
	iNew := strings.Index(view, "2026-08-15T12-00-00Z")
	iOld := strings.Index(view, "2026-08-14T12-00-00Z")
	if iNew < 0 || iOld < 0 || iNew > iOld {
		t.Fatalf("View() = %q, want newest snapshot before older", view)
	}
}

func TestBackupUpdate_TabSwitchesMode(t *testing.T) {
	m := BackupModel{tab: backupTabBackup}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm := next.(BackupModel)
	if bm.tab != backupTabRestore {
		t.Fatalf("tab = %v, want restore", bm.tab)
	}

	next, _ = bm.Update(tea.KeyMsg{Type: tea.KeyTab})
	bm = next.(BackupModel)
	if bm.tab != backupTabBackup {
		t.Fatalf("tab = %v, want backup", bm.tab)
	}
}

func TestBackupUpdate_JKNavigateSnapshots(t *testing.T) {
	m := BackupModel{
		tab: backupTabRestore,
		snapshots: []snapshotItem{
			{name: "a"},
			{name: "b"},
		},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	bm := next.(BackupModel)
	if bm.cursor != 1 {
		t.Fatalf("j: cursor = %d, want 1", bm.cursor)
	}

	next, _ = bm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	bm = next.(BackupModel)
	if bm.cursor != 0 {
		t.Fatalf("k: cursor = %d, want 0", bm.cursor)
	}
}

func TestBackupUpdate_EnterOnRestoreOpensConfirm(t *testing.T) {
	m := BackupModel{
		tab: backupTabRestore,
		snapshots: []snapshotItem{
			{name: "2026-08-15T12-00-00Z", path: "/tmp/snap"},
		},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm := next.(BackupModel)
	if !bm.confirm.Active {
		t.Fatal("expected confirm after enter on restore")
	}
	if !strings.Contains(bm.confirm.Prompt, "2026-08-15T12-00-00Z") {
		t.Fatalf("prompt = %q, want snapshot name", bm.confirm.Prompt)
	}
	view := bm.View()
	if !strings.Contains(view, "y") || !strings.Contains(view, "n") {
		t.Fatalf("View() = %q, want y/n confirm", view)
	}
}

func TestBackupUpdate_ConfirmYStartsRestore(t *testing.T) {
	m := BackupModel{
		tab: backupTabRestore,
		snapshots: []snapshotItem{
			{name: "snap", path: "/tmp/snap"},
		},
		confirm: ConfirmModel{Prompt: "Restore from snap?", Active: true},
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	bm := next.(BackupModel)
	if bm.confirm.Active {
		t.Fatal("confirm should clear after y")
	}
	if !bm.busy {
		t.Fatal("expected busy during restore")
	}
	if cmd == nil {
		t.Fatal("expected restore command")
	}
}

func TestBackupUpdate_ConfirmNCancels(t *testing.T) {
	m := BackupModel{
		tab: backupTabRestore,
		snapshots: []snapshotItem{
			{name: "snap", path: "/tmp/snap"},
		},
		confirm: ConfirmModel{Prompt: "Restore from snap?", Active: true},
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	bm := next.(BackupModel)
	if bm.confirm.Active {
		t.Fatal("confirm should clear after n")
	}
	if bm.busy {
		t.Fatal("cancel must not start restore")
	}
	if cmd != nil {
		t.Fatal("expected no command on cancel")
	}
}

func TestBackupUpdate_EnterOnBackupStartsBackup(t *testing.T) {
	m := BackupModel{
		tab:        backupTabBackup,
		configPath: "/tmp/pourover.lua",
		stateDir:   "/tmp/state",
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm := next.(BackupModel)
	if !bm.busy {
		t.Fatal("expected busy during backup")
	}
	if cmd == nil {
		t.Fatal("expected backup command")
	}
}

func TestBackupUpdate_BackupDoneShowsResult(t *testing.T) {
	m := BackupModel{tab: backupTabBackup, busy: true}

	next, _ := m.Update(backupDoneMsg{
		result: engine.BackupResult{
			LocalSnapshot: "/tmp/state/snapshots/2026-08-15T12-00-00Z",
			MirroredTo:    "/tmp/icloud/snapshots/2026-08-15T12-00-00Z",
		},
	})
	bm := next.(BackupModel)
	if bm.busy {
		t.Fatal("busy should clear after done")
	}
	view := bm.View()
	if !strings.Contains(view, "/tmp/state/snapshots/2026-08-15T12-00-00Z") {
		t.Fatalf("View() = %q, want LocalSnapshot", view)
	}
	if !strings.Contains(view, "Mirrored") || !strings.Contains(view, "/tmp/icloud") {
		t.Fatalf("View() = %q, want MirroredTo", view)
	}
}

func TestBackupUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := BackupModel{home: home}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestBackupUpdate_EscDuringConfirmCancels(t *testing.T) {
	m := BackupModel{
		tab:     backupTabRestore,
		confirm: ConfirmModel{Prompt: "Restore?", Active: true},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	bm, ok := next.(BackupModel)
	if !ok {
		t.Fatalf("got %T, want BackupModel", next)
	}
	if bm.confirm.Active {
		t.Fatal("esc should cancel confirm")
	}
}

func TestBackupUpdate_QQuitsWhenIdle(t *testing.T) {
	m := BackupModel{}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestBackupUpdate_QIgnoredWhenBusy(t *testing.T) {
	m := BackupModel{busy: true}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q must not quit while busy")
	}
}

func TestListSnapshots_NewestFirst(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	snapRoot := paths.SnapshotsDir(stateDir)
	for _, name := range []string{"2026-08-14T12-00-00Z", "2026-08-15T12-00-00Z"} {
		if err := os.MkdirAll(filepath.Join(snapRoot, name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	items, err := listSnapshots(stateDir)
	if err != nil {
		t.Fatalf("listSnapshots: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].name != "2026-08-15T12-00-00Z" {
		t.Fatalf("first = %q, want newest", items[0].name)
	}
	if items[1].name != "2026-08-14T12-00-00Z" {
		t.Fatalf("second = %q, want older", items[1].name)
	}
}

func TestListSnapshots_MissingDir(t *testing.T) {
	t.Parallel()

	items, err := listSnapshots(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("listSnapshots missing: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %v, want empty", items)
	}
}

func TestHomeUpdate_EnterOnBackupOpensBackupView(t *testing.T) {
	m := newTestHome()
	m.cursor = 5 // Backup/Restore

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm, ok := next.(BackupModel)
	if !ok {
		t.Fatalf("got %T, want BackupModel", next)
	}
	if cmd == nil {
		t.Fatal("expected Init/load command when opening backup view")
	}
	if bm.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", bm.configPath, m.configPath)
	}
}
