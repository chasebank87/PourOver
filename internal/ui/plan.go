package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/chasebank87/PourOver/internal/plan"
)

// WritePlan prints an apply-like plan header and action list to w.
// Pending actions are muted; advisory rename lines use Warning.
func WritePlan(w io.Writer, p plan.Plan) {
	fmt.Fprintln(w, Brand().Render("☕ PourOver")+"  "+Accent().Render("plan"))
	fmt.Fprintln(w, Muted().Render(strings.Repeat("─", 40)))
	if len(p.Actions) == 0 {
		fmt.Fprintln(w, Muted().Render("☕ No changes."))
		return
	}
	for _, a := range p.Actions {
		line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
		if line == "" {
			continue
		}
		fmt.Fprintln(w, PlanActionStyle(a).Render(line))
	}
}

// PlanActionStyle returns the style for a plan/dry-run action line.
// Outcomes have not happened yet: pending is muted, advisories are warning.
func PlanActionStyle(a plan.Action) lipgloss.Style {
	switch a.Type {
	case plan.ActionCaskRename:
		return Warning()
	default:
		return Muted()
	}
}
