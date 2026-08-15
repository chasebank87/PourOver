package plan

import (
	"context"
	"sort"

	"github.com/chasebank87/PourOver/internal/discovery"
)

// AdviseCaskRenames replaces cask_install actions for retired Homebrew tokens
// with cask_rename advice when the canonical cask is already installed, and
// drops removes of those canonical casks so apply does not uninstall them.
func AdviseCaskRenames(ctx context.Context, runner discovery.Runner, p Plan, installedCasks []string) (Plan, error) {
	var pending []string
	for _, a := range p.Actions {
		if a.Type == ActionCaskInstall {
			pending = append(pending, a.Name)
		}
	}
	if len(pending) == 0 {
		return p, nil
	}
	renames, err := discovery.DetectCaskRenames(ctx, runner, pending, installedCasks)
	if err != nil {
		return p, err
	}
	if len(renames) == 0 {
		return p, nil
	}
	byFrom := map[string]string{}
	keepCanonical := map[string]struct{}{}
	for _, r := range renames {
		byFrom[r.From] = r.To
		keepCanonical[r.To] = struct{}{}
	}

	var out []Action
	var renameActions []Action
	for _, a := range p.Actions {
		switch a.Type {
		case ActionCaskInstall:
			if to, ok := byFrom[a.Name]; ok {
				renameActions = append(renameActions, Action{
					Type:  ActionCaskRename,
					Name:  a.Name,
					Value: to,
				})
				continue
			}
			out = append(out, a)
		case ActionCaskRemove:
			if _, keep := keepCanonical[a.Name]; keep {
				continue
			}
			out = append(out, a)
		default:
			out = append(out, a)
		}
	}
	sort.Slice(renameActions, func(i, j int) bool {
		return renameActions[i].Name < renameActions[j].Name
	})
	// Renames first so they are obvious in plan/apply output.
	return Plan{Actions: append(renameActions, out...)}, nil
}
