package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/ui"
)

// PlanModel shows the current reconcile plan / drift list.
type PlanModel struct {
	configPath string
	home       HomeModel
	plan       plan.Plan
	err        string
	loading    bool
	status     string // transient status (e.g. apply stub)
	width      int
	height     int
}

// NewPlanModel constructs a plan view for configPath, returning to home on esc.
func NewPlanModel(configPath string, home HomeModel) PlanModel {
	return PlanModel{
		configPath: configPath,
		home:       home,
		loading:    true,
	}
}

type planLoadedMsg struct {
	plan plan.Plan
	err  error
}

func (m PlanModel) Init() tea.Cmd {
	return refreshPlan(m.configPath)
}

func refreshPlan(configPath string) tea.Cmd {
	return func() tea.Msg {
		p, err := engine.BuildPlan(context.Background(), configPath, discovery.NewExecRunner())
		return planLoadedMsg{plan: p, err: err}
	}
}

func (m PlanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case planLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = friendlyErr(msg.err)
			m.plan = plan.Plan{}
			return m, nil
		}
		m.err = ""
		m.plan = msg.plan
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "r":
			m.loading = true
			m.status = ""
			m.err = ""
			return m, refreshPlan(m.configPath)
		case "a":
			rm := NewRunModel(RunApply, m.configPath, m.home)
			return rm, rm.Init()
		}
	}
	return m, nil
}

func (m PlanModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")

	n := len(m.plan.Actions)
	header := fmt.Sprintf("Plan — %d action(s)", n)
	if n == 0 {
		b.WriteString(styleSuccess.Render(header))
	} else {
		b.WriteString(styleWarning.Render(header))
	}
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(styleMuted.Render("loading plan…"))
		b.WriteString("\n")
	case m.err != "":
		b.WriteString(styleFail.Render("error: " + m.err))
		b.WriteString("\n")
	case n == 0:
		b.WriteString(styleSuccess.Render("in sync."))
		b.WriteString("\n")
	default:
		for _, a := range m.plan.Actions {
			line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
			b.WriteString(ui.PlanActionStyle(a).Render(line))
			b.WriteString("\n")
		}
	}

	if m.status != "" {
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(m.status))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("r refresh · a apply · esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}
