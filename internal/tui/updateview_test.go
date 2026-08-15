package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/selfupdate"
)

func TestSelfUpdateView_BusyAndResult(t *testing.T) {
	t.Parallel()

	busy := SelfUpdateModel{busy: true}
	if !strings.Contains(busy.View(), "checking") {
		t.Fatalf("busy View() = %q, want checking state", busy.View())
	}

	done := SelfUpdateModel{
		done: true,
		result: selfupdate.Result{
			Updated:    true,
			CurrentTag: "0.1.0",
			LatestTag:  "0.2.0",
		},
	}
	view := done.View()
	if !strings.Contains(view, "Updated: yes") ||
		!strings.Contains(view, "CurrentTag: 0.1.0") ||
		!strings.Contains(view, "LatestTag: 0.2.0") {
		t.Fatalf("View() = %q, want Updated/CurrentTag/LatestTag", view)
	}
}

func TestSelfUpdateView_Error(t *testing.T) {
	t.Parallel()

	m := SelfUpdateModel{err: "network down", done: true}
	if !strings.Contains(m.View(), "network down") {
		t.Fatalf("View() = %q, want error text", m.View())
	}
}

func TestSelfUpdateUpdate_DoneMsg(t *testing.T) {
	m := SelfUpdateModel{busy: true}

	next, _ := m.Update(selfUpdateDoneMsg{
		result: selfupdate.Result{Updated: false, CurrentTag: "0.2.0", LatestTag: "0.2.0"},
	})
	sm := next.(SelfUpdateModel)
	if sm.busy || !sm.done {
		t.Fatalf("busy=%v done=%v after done msg", sm.busy, sm.done)
	}
	if sm.result.CurrentTag != "0.2.0" || sm.result.LatestTag != "0.2.0" {
		t.Fatalf("result = %+v", sm.result)
	}
}

func TestSelfUpdateUpdate_DoneMsgError(t *testing.T) {
	m := SelfUpdateModel{busy: true}

	next, _ := m.Update(selfUpdateDoneMsg{err: errors.New("boom")})
	sm := next.(SelfUpdateModel)
	if sm.busy || sm.err != "boom" {
		t.Fatalf("busy=%v err=%q", sm.busy, sm.err)
	}
}

func TestSelfUpdateUpdate_BusyGatesKeys(t *testing.T) {
	m := SelfUpdateModel{busy: true, home: newTestHome()}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	sm, ok := next.(SelfUpdateModel)
	if !ok {
		t.Fatalf("got %T, want SelfUpdateModel while busy", next)
	}
	if !sm.busy {
		t.Fatal("busy should remain while downloading")
	}
	if cmd != nil {
		t.Fatal("expected no command while busy")
	}
}

func TestSelfUpdateUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := SelfUpdateModel{home: home, done: true}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestSelfUpdateUpdate_QQuits(t *testing.T) {
	m := SelfUpdateModel{done: true}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestHomeUpdate_EnterOnSelfUpdateOpensView(t *testing.T) {
	m := newTestHome()
	m.cursor = selfUpdateMenuIndex(m)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	sm, ok := next.(SelfUpdateModel)
	if !ok {
		t.Fatalf("got %T, want SelfUpdateModel", next)
	}
	if !sm.busy {
		t.Fatal("expected busy=true when opening self-update view")
	}
	if cmd == nil {
		t.Fatal("expected Init/run command when opening self-update view")
	}
}

func selfUpdateMenuIndex(m HomeModel) int {
	for i, item := range m.items {
		if item.id == menuSelfUpdate {
			return i
		}
	}
	panic("self-update menu item missing")
}
