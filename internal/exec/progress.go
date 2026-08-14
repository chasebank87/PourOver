package exec

import (
	"strings"

	"github.com/chasebank87/PourOver/internal/plan"
)

// Progress reports a human-readable line for an action about to run.
// May be nil to suppress per-action progress.
type Progress func(line string)

func report(progress Progress, a plan.Action) {
	if progress == nil {
		return
	}
	line := strings.TrimSuffix(plan.RenderText(plan.Plan{Actions: []plan.Action{a}}), "\n")
	if line != "" && line != "No changes." {
		progress(line)
	}
}
