package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirmModel_ViewShowsPrompt(t *testing.T) {
	t.Parallel()

	m := ConfirmModel{Prompt: "Uninstall undeclared packages: foo?", Active: true}
	view := m.View()
	if !strings.Contains(view, "Uninstall undeclared packages: foo?") {
		t.Fatalf("View() = %q, want prompt", view)
	}
	if !strings.Contains(view, "y") || !strings.Contains(view, "n") {
		t.Fatalf("View() = %q, want y/n hints", view)
	}
}

func TestConfirmModel_YAnswersYes(t *testing.T) {
	t.Parallel()

	m := ConfirmModel{Prompt: "proceed?", Active: true}
	next, answered := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if answered == nil || !*answered {
		t.Fatalf("answered = %v, want true", answered)
	}
	if next.Active {
		t.Fatal("expected Active=false after answer")
	}
}

func TestConfirmModel_NAnswersNo(t *testing.T) {
	t.Parallel()

	m := ConfirmModel{Prompt: "proceed?", Active: true}
	next, answered := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if answered == nil || *answered {
		t.Fatalf("answered = %v, want false", answered)
	}
	if next.Active {
		t.Fatal("expected Active=false after answer")
	}
}

func TestConfirmModel_IgnoresKeysWhenInactive(t *testing.T) {
	t.Parallel()

	m := ConfirmModel{Prompt: "proceed?", Active: false}
	next, answered := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if answered != nil {
		t.Fatalf("answered = %v, want nil when inactive", answered)
	}
	if next.Active {
		t.Fatal("inactive confirm must stay inactive")
	}
}

func TestAsyncConfirmer_AskProducesConfirmNeededMsg(t *testing.T) {
	c := NewAsyncConfirmer()
	done := make(chan bool, 1)
	go func() {
		done <- c.Confirm("uninstall bar?")
	}()

	cmd := waitConfirmCmd(c)
	msg := cmd()
	needed, ok := msg.(confirmNeededMsg)
	if !ok {
		t.Fatalf("got %T, want confirmNeededMsg", msg)
	}
	if needed.Prompt != "uninstall bar?" {
		t.Fatalf("prompt = %q", needed.Prompt)
	}
	c.Answer(true)
	if got := <-done; !got {
		t.Fatal("expected Confirm to return true")
	}
}
