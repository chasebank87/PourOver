package plan

import (
	"fmt"
	"strconv"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildDefaultsPlan emits defaults_write actions for drifted settings.
func BuildDefaultsPlan(statuses []discovery.SettingStatus) Plan {
	var actions []Action
	for _, st := range statuses {
		if !st.Drift {
			continue
		}
		d := st.Desired
		actions = append(actions, Action{
			Type:   ActionDefaultsWrite,
			Name:   d.Domain + " " + d.Key,
			Domain: d.Domain,
			Key:    d.Key,
			Value:  formatSettingValue(d.Value),
			Kind:   string(d.Value.Kind),
		})
	}
	return Plan{Actions: actions}
}

func formatSettingValue(v config.SettingValue) string {
	switch v.Kind {
	case config.SettingBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case config.SettingInt:
		return strconv.FormatInt(v.Int, 10)
	case config.SettingFloat:
		return strconv.FormatFloat(v.Float, 'f', -1, 64)
	case config.SettingString:
		return v.String
	default:
		return fmt.Sprintf("%v", v)
	}
}
