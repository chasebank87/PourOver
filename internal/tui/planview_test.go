package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestPlanView_EmptyShowsInSync(t *testing.T) {
	t.Parallel()

	m := PlanModel{
		configPath: "/tmp/pourover.lua",
		plan:       plan.Plan{},
	}
	view := m.View()
	if !strings.Contains(view, "in sync.") {
		t.Fatalf("View() = %q, want empty-state \"in sync.\"", view)
	}
	if !strings.Contains(view, "0 action") {
		t.Fatalf("View() = %q, want action count header", view)
	}
}

func TestPlanView_ListsActionsAndCount(t *testing.T) {
	t.Parallel()

	m := PlanModel{
		configPath: "/tmp/pourover.lua",
		plan: plan.Plan{Actions: []plan.Action{
			{Type: plan.ActionFormulaInstall, Name: "git"},
			{Type: plan.ActionCaskInstall, Name: "firefox"},
		}},
	}
	view := m.View()
	if !strings.Contains(view, "2 action") {
		t.Fatalf("View() = %q, want 2-action header", view)
	}
	if !strings.Contains(view, "install formula git") {
		t.Fatalf("View() = %q, want formula action line", view)
	}
	if !strings.Contains(view, "install cask firefox") {
		t.Fatalf("View() = %q, want cask action line", view)
	}
	if strings.Contains(view, "in sync.") {
		t.Fatalf("View() = %q, must not show empty state when actions exist", view)
	}
}

func TestPlanUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := PlanModel{
		configPath: home.configPath,
		home:       home,
		plan:       plan.Plan{},
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestPlanUpdate_QQuits(t *testing.T) {
	m := PlanModel{configPath: "/tmp/pourover.lua"}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestPlanUpdate_RRefreshes(t *testing.T) {
	m := PlanModel{configPath: "/tmp/pourover.lua", plan: plan.Plan{}}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	pm, ok := next.(PlanModel)
	if !ok {
		t.Fatalf("got %T, want PlanModel", next)
	}
	if !pm.loading {
		t.Fatal("expected loading=true after refresh")
	}
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
}

func TestPlanUpdate_PlanLoadedMsg(t *testing.T) {
	m := PlanModel{configPath: "/tmp/pourover.lua", loading: true}

	next, _ := m.Update(planLoadedMsg{
		plan: plan.Plan{Actions: []plan.Action{
			{Type: plan.ActionFormulaInstall, Name: "ripgrep"},
		}},
	})
	pm := next.(PlanModel)
	if pm.loading {
		t.Fatal("loading should clear after planLoadedMsg")
	}
	if len(pm.plan.Actions) != 1 || pm.plan.Actions[0].Name != "ripgrep" {
		t.Fatalf("plan = %+v, want ripgrep action", pm.plan)
	}
}

func TestPlanUpdate_AShowsApplyStub(t *testing.T) {
	m := PlanModel{configPath: "/tmp/pourover.lua", plan: plan.Plan{}}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	pm := next.(PlanModel)
	view := pm.View()
	if !strings.Contains(view, "Apply: coming in 1.5") {
		t.Fatalf("View() = %q, want apply stub message", view)
	}
}

func TestHomeUpdate_EnterOnPlanOpensPlanView(t *testing.T) {
	m := newTestHome()
	m.cursor = 0 // Plan

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm, ok := next.(PlanModel)
	if !ok {
		t.Fatalf("got %T, want PlanModel", next)
	}
	if pm.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", pm.configPath, m.configPath)
	}
	if cmd == nil {
		t.Fatal("expected Init/refresh command when opening plan view")
	}
}
