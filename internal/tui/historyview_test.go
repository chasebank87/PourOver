package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/state"
)

func TestHistoryView_ListsNewestFirst(t *testing.T) {
	t.Parallel()

	m := HistoryModel{
		stateDir: "/tmp/state",
		entries: []historyItem{
			{
				name: "2026-08-15T12-00-00Z.json",
				entry: state.HistoryEntry{
					Timestamp:   "2026-08-15T12:00:00Z",
					Success:     true,
					ActionCount: 3,
				},
			},
			{
				name: "2026-08-14T12-00-00Z.json",
				entry: state.HistoryEntry{
					Timestamp:   "2026-08-14T12:00:00Z",
					Success:     false,
					ActionCount: 1,
					Error:       "brew failed",
				},
			},
		},
	}
	view := m.View()
	if !strings.Contains(view, "2026-08-15T12:00:00Z") {
		t.Fatalf("View() = %q, want newest timestamp", view)
	}
	if !strings.Contains(view, "ok") || !strings.Contains(view, "3 action") {
		t.Fatalf("View() = %q, want success summary", view)
	}
	if !strings.Contains(view, "FAIL") || !strings.Contains(view, "1 action") {
		t.Fatalf("View() = %q, want failure summary", view)
	}

	iNew := strings.Index(view, "2026-08-15T12:00:00Z")
	iOld := strings.Index(view, "2026-08-14T12:00:00Z")
	if iNew < 0 || iOld < 0 || iNew > iOld {
		t.Fatalf("View() = %q, want newest entry before older", view)
	}
}

func TestHistoryView_EmptyAndLoading(t *testing.T) {
	t.Parallel()

	empty := HistoryModel{}
	if !strings.Contains(empty.View(), "no history") {
		t.Fatalf("empty View() = %q, want empty-state message", empty.View())
	}

	loading := HistoryModel{loading: true}
	if !strings.Contains(loading.View(), "loading") {
		t.Fatalf("loading View() = %q, want loading state", loading.View())
	}
}

func TestHistoryUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := HistoryModel{home: home}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestHistoryUpdate_EscFromDetailReturnsList(t *testing.T) {
	m := HistoryModel{
		entries: []historyItem{{
			entry: state.HistoryEntry{Timestamp: "2026-08-15T12:00:00Z", Success: true},
		}},
		detail: true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HistoryModel)
	if !ok {
		t.Fatalf("got %T, want HistoryModel", next)
	}
	if hm.detail {
		t.Fatal("detail should clear on esc")
	}
}

func TestHistoryUpdate_EnterOpensDetail(t *testing.T) {
	m := HistoryModel{
		entries: []historyItem{{
			entry: state.HistoryEntry{
				Timestamp:   "2026-08-15T12:00:00Z",
				Success:     true,
				ActionCount: 2,
			},
		}},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	hm := next.(HistoryModel)
	if !hm.detail {
		t.Fatal("expected detail after enter")
	}
	view := hm.View()
	if !strings.Contains(view, "2026-08-15T12:00:00Z") {
		t.Fatalf("detail View() = %q, want timestamp", view)
	}
}

func TestHistoryUpdate_JKNavigate(t *testing.T) {
	m := HistoryModel{
		entries: []historyItem{
			{entry: state.HistoryEntry{Timestamp: "a"}},
			{entry: state.HistoryEntry{Timestamp: "b"}},
		},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	hm := next.(HistoryModel)
	if hm.cursor != 1 {
		t.Fatalf("j: cursor = %d, want 1", hm.cursor)
	}

	next, _ = hm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	hm = next.(HistoryModel)
	if hm.cursor != 0 {
		t.Fatalf("k: cursor = %d, want 0", hm.cursor)
	}
}

func TestHistoryUpdate_QQuits(t *testing.T) {
	m := HistoryModel{}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestLoadHistoryEntries_NewestFirst(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	older := state.HistoryEntry{Timestamp: "2026-08-14T12:00:00Z", Success: true, ActionCount: 1}
	newer := state.HistoryEntry{Timestamp: "2026-08-15T12:00:00Z", Success: false, ActionCount: 2, Error: "x"}
	if _, err := state.AppendHistory(dir, older, time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("AppendHistory older: %v", err)
	}
	if _, err := state.AppendHistory(dir, newer, time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("AppendHistory newer: %v", err)
	}

	items, err := loadHistoryEntries(dir)
	if err != nil {
		t.Fatalf("loadHistoryEntries: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
	if items[0].entry.Timestamp != newer.Timestamp {
		t.Fatalf("first = %+v, want newest", items[0].entry)
	}
	if items[1].entry.Timestamp != older.Timestamp {
		t.Fatalf("second = %+v, want older", items[1].entry)
	}
}

func TestLoadHistoryEntries_MissingDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "missing-state")
	items, err := loadHistoryEntries(dir)
	if err != nil {
		t.Fatalf("loadHistoryEntries missing dir: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %v, want empty", items)
	}
}

func TestHomeUpdate_EnterOnHistoryOpensHistoryView(t *testing.T) {
	m := newTestHome()
	m.cursor = 4 // History

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	hm, ok := next.(HistoryModel)
	if !ok {
		t.Fatalf("got %T, want HistoryModel", next)
	}
	if cmd == nil {
		t.Fatal("expected Init/load command when opening history view")
	}
	_ = hm
}

func TestHomeUpdate_EnterOnBackupStillStub(t *testing.T) {
	m := newTestHome()
	m.cursor = 5 // Backup/Restore still stubbed

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	hm := next.(HomeModel)
	if hm.stub == "" {
		t.Fatal("expected stub screen after selecting Backup/Restore")
	}

	next, _ = hm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm = next.(HomeModel)
	if hm.stub != "" {
		t.Fatalf("stub = %q after esc, want empty", hm.stub)
	}
}
