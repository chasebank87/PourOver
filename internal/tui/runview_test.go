package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
)

func TestRunView_ShowsPhaseAndLines(t *testing.T) {
	t.Parallel()

	m := RunModel{
		kind:       RunApply,
		configPath: "/tmp/pourover.lua",
		phase:      "formulae",
		lines:      []string{"installing git", "ok: git"},
	}
	view := m.View()
	if !strings.Contains(view, "Apply") {
		t.Fatalf("View() = %q, want Apply title", view)
	}
	if !strings.Contains(view, "formulae") {
		t.Fatalf("View() = %q, want phase label", view)
	}
	if !strings.Contains(view, "installing git") {
		t.Fatalf("View() = %q, want progress line", view)
	}
}

func TestRunUpdate_ProgressLineAppends(t *testing.T) {
	m := RunModel{kind: RunApply, configPath: "/tmp/x.lua"}

	next, _ := m.Update(progressLineMsg{line: "installing ripgrep"})
	rm := next.(RunModel)
	if len(rm.lines) != 1 || rm.lines[0] != "installing ripgrep" {
		t.Fatalf("lines = %#v", rm.lines)
	}
}

func TestRunUpdate_PhaseMsgUpdatesPhase(t *testing.T) {
	m := RunModel{kind: RunApply, phase: "taps"}

	next, _ := m.Update(phaseMsg{phase: "casks"})
	rm := next.(RunModel)
	if rm.phase != "casks" {
		t.Fatalf("phase = %q, want casks", rm.phase)
	}
}

func TestRunUpdate_DoneMsgShowsSummary(t *testing.T) {
	m := RunModel{kind: RunApply, lines: []string{"ok: git"}}

	summary := formatApplySummary(engine.ApplyResult{Formulae: 1, Linked: 2})
	next, _ := m.Update(runDoneMsg{summary: summary})
	rm := next.(RunModel)
	if !rm.done {
		t.Fatal("expected done=true")
	}
	view := rm.View()
	if !strings.Contains(view, summary) {
		t.Fatalf("View() = %q, want summary %q", view, summary)
	}
}

func TestRunUpdate_EscWhenDoneReturnsHome(t *testing.T) {
	home := newTestHome()
	m := RunModel{
		kind: RunApply,
		home: home,
		done: true,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := next.(HomeModel); !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
}

func TestRunUpdate_EscWhileRunningIgnored(t *testing.T) {
	home := newTestHome()
	m := RunModel{
		kind: RunApply,
		home: home,
		done: false,
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := next.(RunModel); !ok {
		t.Fatalf("got %T, want RunModel (esc ignored while running)", next)
	}
}

func TestRunUpdate_ConfirmNeededActivatesModal(t *testing.T) {
	m := RunModel{kind: RunApply}

	next, _ := m.Update(confirmNeededMsg{Prompt: "Uninstall undeclared packages: foo?"})
	rm := next.(RunModel)
	if !rm.confirm.Active {
		t.Fatal("expected confirm modal active")
	}
	if rm.confirm.Prompt != "Uninstall undeclared packages: foo?" {
		t.Fatalf("prompt = %q", rm.confirm.Prompt)
	}
	view := rm.View()
	if !strings.Contains(view, "Uninstall undeclared packages: foo?") {
		t.Fatalf("View() = %q, want confirm prompt", view)
	}
}

func TestRunUpdate_ConfirmYSendsAnswer(t *testing.T) {
	c := NewAsyncConfirmer()
	m := RunModel{kind: RunApply, confirmer: c}

	result := make(chan bool, 1)
	go func() {
		result <- c.Confirm("uninstall?")
	}()

	// Drain ask via wait cmd (engine would trigger this); then simulate UI answer.
	msg := waitConfirmCmd(c)()
	needed := msg.(confirmNeededMsg)
	next, _ := m.Update(needed)
	rm := next.(RunModel)

	next, cmd := rm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	rm = next.(RunModel)
	if rm.confirm.Active {
		t.Fatal("confirm should clear after y")
	}
	_ = cmd

	select {
	case ok := <-result:
		if !ok {
			t.Fatal("expected Confirm true after y")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Confirm answer")
	}
}

func TestFormatApplySummary(t *testing.T) {
	t.Parallel()
	s := formatApplySummary(engine.ApplyResult{Taps: 1, Formulae: 2, Casks: 3, Removed: 0, Defaults: 1, Linked: 4, Failures: 1})
	for _, want := range []string{"1 tap", "2 formula", "3 cask", "1 default", "4 link", "1 failure"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary %q missing %q", s, want)
		}
	}
}

func TestHomeUpdate_EnterOnApplyOpensRunView(t *testing.T) {
	m := newTestHome()
	m.cursor = 1 // Apply

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm, ok := next.(RunModel)
	if !ok {
		t.Fatalf("got %T, want RunModel", next)
	}
	if rm.kind != RunApply {
		t.Fatalf("kind = %q, want apply", rm.kind)
	}
	if rm.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", rm.configPath, m.configPath)
	}
	if cmd == nil {
		t.Fatal("expected Init command when opening apply run")
	}
}

func TestHomeUpdate_EnterOnUpgradeOpensRunView(t *testing.T) {
	m := newTestHome()
	m.cursor = 2 // Upgrade

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rm, ok := next.(RunModel)
	if !ok {
		t.Fatalf("got %T, want RunModel", next)
	}
	if rm.kind != RunUpgrade {
		t.Fatalf("kind = %q, want upgrade", rm.kind)
	}
	if cmd == nil {
		t.Fatal("expected Init command when opening upgrade run")
	}
}

func TestPlanUpdate_AStartsApplyRun(t *testing.T) {
	home := newTestHome()
	m := PlanModel{
		configPath: home.configPath,
		home:       home,
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	rm, ok := next.(RunModel)
	if !ok {
		t.Fatalf("got %T, want RunModel", next)
	}
	if rm.kind != RunApply {
		t.Fatalf("kind = %q, want apply", rm.kind)
	}
	if cmd == nil {
		t.Fatal("expected Init command for apply run")
	}
}
