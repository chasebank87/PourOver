package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFriendlyErr_ConfigMissing(t *testing.T) {
	t.Parallel()

	cases := []error{
		fmt.Errorf("load config: open /tmp/pourover.lua: no such file or directory"),
		errors.New("config not found at ~/.pourover/pourover.lua"),
	}
	for _, err := range cases {
		got := friendlyErr(err)
		if got != "config missing — run pourover init" {
			t.Fatalf("friendlyErr(%q) = %q, want config missing message", err, got)
		}
	}
}

func TestFriendlyErr_BrewNotFoundNotMislabelled(t *testing.T) {
	t.Parallel()

	err := errors.New(`discover brew: exec: "brew": executable file not found in $PATH`)
	got := friendlyErr(err)
	if strings.Contains(got, "config missing") {
		t.Fatalf("friendlyErr(%q) = %q, must not claim missing config", err, got)
	}
	if !strings.Contains(got, "brew") && !strings.Contains(got, "discover brew") {
		t.Fatalf("friendlyErr(%q) = %q, want brew/discover context preserved", err, got)
	}
}

func TestHomeUpdate_DownMovesSelection(t *testing.T) {
	m := newTestHome()
	if m.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", m.cursor)
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", hm.cursor)
	}
}

func TestHomeUpdate_UpMovesSelection(t *testing.T) {
	m := newTestHome()
	m.cursor = 2

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", hm.cursor)
	}
}

func TestHomeUpdate_JKNavigate(t *testing.T) {
	m := newTestHome()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	hm := next.(HomeModel)
	if hm.cursor != 1 {
		t.Fatalf("j: cursor = %d, want 1", hm.cursor)
	}

	next, _ = hm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	hm = next.(HomeModel)
	if hm.cursor != 0 {
		t.Fatalf("k: cursor = %d, want 0", hm.cursor)
	}
}

func TestHomeUpdate_EnterOnQuitQuits(t *testing.T) {
	m := newTestHome()
	m.cursor = quitMenuIndex(m)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestHomeUpdate_QQuits(t *testing.T) {
	m := newTestHome()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func newTestHome() HomeModel {
	return HomeModel{
		items:      defaultMenuItems(),
		configPath: "/tmp/pourover.lua",
		driftLine:  "drift: 0 actions",
		doctorLine: "doctor: not checked",
	}
}

func quitMenuIndex(m HomeModel) int {
	for i, item := range m.items {
		if item.id == menuQuit {
			return i
		}
	}
	panic("quit menu item missing")
}
