package plan

import (
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildAutoUpdatePlan reconciles brew autoupdate with packages.auto_update.
// Omitted auto_update (Configured=false) yields an empty plan.
func BuildAutoUpdatePlan(desired config.AutoUpdate, current discovery.AutoupdateState) Plan {
	if !desired.Configured {
		return Plan{}
	}

	if !desired.Enable {
		if current.Installed || current.Running {
			return Plan{Actions: []Action{{Type: ActionBrewAutoupdateDelete, Name: discovery.AutoUpdateLabel}}}
		}
		return Plan{}
	}

	matches := current.Installed && desired.MatchesSchedule(current.Interval, current.Upgrade, current.Greedy, current.Cleanup, current.Immediate)
	if matches && current.Running {
		return Plan{}
	}

	var actions []Action
	if current.Installed && !matches {
		actions = append(actions, Action{Type: ActionBrewAutoupdateDelete, Name: discovery.AutoUpdateLabel})
	}
	actions = append(actions, autoUpdateStartAction(desired))
	return Plan{Actions: actions}
}

func autoUpdateStartAction(desired config.AutoUpdate) Action {
	var flags []string
	if desired.Upgrade {
		flags = append(flags, "--upgrade")
	}
	if desired.Greedy {
		flags = append(flags, "--greedy")
	}
	if desired.Cleanup {
		flags = append(flags, "--cleanup")
	}
	if desired.Immediate {
		flags = append(flags, "--immediate")
	}
	return Action{
		Type:  ActionBrewAutoupdateStart,
		Name:  desired.IntervalDisplay(),
		Value: strings.Join(flags, " "),
	}
}
