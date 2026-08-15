package plan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// RenderText returns a stable, human-readable plan summary.
func RenderText(p Plan) string {
	if len(p.Actions) == 0 {
		return "No changes.\n"
	}
	var b strings.Builder
	for _, a := range p.Actions {
		if line := formatActionLine(a); line != "" {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// RenderJSON returns indented JSON for the plan (stable field names for tooling).
func RenderJSON(p Plan) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatActionLine(a Action) string {
	switch a.Type {
	case ActionTapAdd:
		return fmt.Sprintf("tap %s", a.Name)
	case ActionTapTrust:
		return fmt.Sprintf("trust tap %s", a.Name)
	case ActionTapRemove:
		return fmt.Sprintf("untap %s", a.Name)
	case ActionFormulaInstall:
		return fmt.Sprintf("install formula %s", a.Name)
	case ActionFormulaRemove:
		return fmt.Sprintf("remove formula %s", a.Name)
	case ActionFormulaUpgrade:
		return fmt.Sprintf("upgrade formula %s", a.Name)
	case ActionCaskInstall:
		return fmt.Sprintf("install cask %s", a.Name)
	case ActionCaskRemove:
		return fmt.Sprintf("remove cask %s", a.Name)
	case ActionCaskUpgrade:
		return fmt.Sprintf("upgrade cask %s", a.Name)
	case ActionLinkCreate:
		return fmt.Sprintf("link create %s <- %s", a.Name, a.Source)
	case ActionLinkUpdate:
		return fmt.Sprintf("link update %s <- %s", a.Name, a.Source)
	case ActionDefaultsWrite:
		return fmt.Sprintf("defaults write %s %s = %s", a.Domain, a.Key, a.Value)
	default:
		return fmt.Sprintf("unknown %s %s", a.Type, a.Name)
	}
}
