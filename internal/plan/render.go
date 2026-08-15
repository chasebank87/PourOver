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
	case ActionCaskRename:
		return fmt.Sprintf("cask renamed: %s → %s (update packages.lua)", a.Name, a.Value)
	case ActionLinkCreate:
		return fmt.Sprintf("link create %s <- %s", a.Name, a.Source)
	case ActionLinkUpdate:
		return fmt.Sprintf("link update %s <- %s", a.Name, a.Source)
	case ActionLinkReplace:
		return fmt.Sprintf("link replace %s <- %s (backup)", a.Name, a.Source)
	case ActionManagedCopy:
		if a.Kind == "backup" {
			return fmt.Sprintf("managed copy %s <- %s (backup)", a.Name, a.Source)
		}
		return fmt.Sprintf("managed copy %s <- %s", a.Name, a.Source)
	case ActionTemplateWrite:
		if a.Kind == "backup" {
			return fmt.Sprintf("template write %s <- %s (backup)", a.Name, a.Source)
		}
		return fmt.Sprintf("template write %s <- %s", a.Name, a.Source)
	case ActionFileUnlink:
		return fmt.Sprintf("unlink %s", a.Name)
	case ActionFilePrune:
		return fmt.Sprintf("prune file %s", a.Name)
	case ActionDefaultsWrite:
		return fmt.Sprintf("defaults write %s %s = %s", a.Domain, a.Key, a.Value)
	case ActionPAMSudoLocalWrite:
		return fmt.Sprintf("pam write %s", a.Name)
	case ActionPAMSudoLocalRemove:
		return fmt.Sprintf("pam disable stub %s", a.Name)
	case ActionPAMSudoInclude:
		return fmt.Sprintf("pam include sudo_local in %s", a.Name)
	default:
		return fmt.Sprintf("unknown %s %s", a.Type, a.Name)
	}
}
