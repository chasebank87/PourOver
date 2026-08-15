package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
)

func TestDoctorView_ShowsPassFailChecklist(t *testing.T) {
	t.Parallel()

	m := DoctorModel{
		configPath: "/tmp/pourover.lua",
		report: engine.DoctorReport{Checks: []engine.DoctorCheck{
			{Name: "brew", OK: true, Detail: "available"},
			{Name: "config", OK: false, Detail: "not found"},
		}},
	}
	view := m.View()
	if !strings.Contains(view, "PASS") || !strings.Contains(view, "brew") {
		t.Fatalf("View() = %q, want PASS brew line", view)
	}
	if !strings.Contains(view, "FAIL") || !strings.Contains(view, "config") {
		t.Fatalf("View() = %q, want FAIL config line", view)
	}
	if !strings.Contains(view, "not found") {
		t.Fatalf("View() = %q, want check detail", view)
	}
}

func TestDoctorView_LoadingAndError(t *testing.T) {
	t.Parallel()

	loading := DoctorModel{loading: true}
	if !strings.Contains(loading.View(), "loading") {
		t.Fatalf("loading View() = %q, want loading state", loading.View())
	}

	errored := DoctorModel{err: "boom"}
	if !strings.Contains(errored.View(), "boom") {
		t.Fatalf("error View() = %q, want error text", errored.View())
	}
}

func TestDoctorUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := DoctorModel{configPath: home.configPath, home: home}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestDoctorUpdate_QQuits(t *testing.T) {
	m := DoctorModel{configPath: "/tmp/pourover.lua"}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestDoctorUpdate_DoctorLoadedMsg(t *testing.T) {
	m := DoctorModel{loading: true}

	next, _ := m.Update(doctorLoadedMsg{
		report: engine.DoctorReport{Checks: []engine.DoctorCheck{
			{Name: "state", OK: true, Detail: "/tmp/state"},
		}},
	})
	dm := next.(DoctorModel)
	if dm.loading {
		t.Fatal("loading should clear after doctorLoadedMsg")
	}
	if len(dm.report.Checks) != 1 || dm.report.Checks[0].Name != "state" {
		t.Fatalf("report = %+v, want state check", dm.report)
	}
}

func TestHomeUpdate_EnterOnDoctorOpensDoctorView(t *testing.T) {
	m := newTestHome()
	m.cursor = 3 // Doctor

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	dm, ok := next.(DoctorModel)
	if !ok {
		t.Fatalf("got %T, want DoctorModel", next)
	}
	if dm.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", dm.configPath, m.configPath)
	}
	if cmd == nil {
		t.Fatal("expected Init/load command when opening doctor view")
	}
}
