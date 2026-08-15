package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// BuildFilePlan computes link create/update/replace actions from discovered link status.
// When replaceMode is backup, blocked targets (existing regular files) become link_replace.
// Otherwise blocked targets are a plan error (default / error mode).
func BuildFilePlan(statuses []discovery.FileLinkStatus, replaceMode config.FileReplaceMode) (Plan, error) {
	var creates, updates, replaces []Action

	for _, st := range statuses {
		switch st.Kind {
		case discovery.LinkStatusMissing:
			creates = append(creates, linkAction(ActionLinkCreate, st))
		case discovery.LinkStatusWrong:
			updates = append(updates, linkAction(ActionLinkUpdate, st))
		case discovery.LinkStatusCorrect:
			continue
		case discovery.LinkStatusBlocked:
			if replaceMode == config.FileReplaceBackup {
				replaces = append(replaces, linkAction(ActionLinkReplace, st))
				continue
			}
			return Plan{}, fmt.Errorf(
				"target %q exists and is not a symlink (source %q)",
				st.Link.Target, st.Link.Source,
			)
		default:
			return Plan{}, fmt.Errorf("unknown link status %q for target %q", st.Kind, st.Link.Target)
		}
	}

	sort.Slice(creates, func(i, j int) bool { return creates[i].Name < creates[j].Name })
	sort.Slice(updates, func(i, j int) bool { return updates[i].Name < updates[j].Name })
	sort.Slice(replaces, func(i, j int) bool { return replaces[i].Name < replaces[j].Name })

	actions := make([]Action, 0, len(creates)+len(updates)+len(replaces))
	actions = append(actions, creates...)
	actions = append(actions, updates...)
	actions = append(actions, replaces...)
	return Plan{Actions: actions}, nil
}

func linkAction(typ ActionType, st discovery.FileLinkStatus) Action {
	return Action{
		Type:   typ,
		Name:   st.Link.Target,
		Source: st.Link.Source,
	}
}

// BuildManagedPlan computes managed_copy actions from discovered managed status.
// Missing and content-differ both emit managed_copy; same is a noop.
// Blocked (unexpected target type) errors unless replaceMode is backup.
func BuildManagedPlan(statuses []discovery.ManagedStatus, replaceMode config.FileReplaceMode) (Plan, error) {
	var actions []Action
	for _, st := range statuses {
		switch st.Kind {
		case discovery.ManagedStatusMissing, discovery.ManagedStatusDiffer:
			actions = append(actions, Action{
				Type:   ActionManagedCopy,
				Name:   st.File.Target,
				Source: st.File.Source,
			})
		case discovery.ManagedStatusSame:
			continue
		case discovery.ManagedStatusBlocked:
			if replaceMode == config.FileReplaceBackup {
				actions = append(actions, Action{
					Type:   ActionManagedCopy,
					Name:   st.File.Target,
					Source: st.File.Source,
					Kind:   "backup",
				})
				continue
			}
			return Plan{}, fmt.Errorf(
				"target %q exists and is not a replaceable file (source %q)",
				st.File.Target, st.File.Source,
			)
		default:
			return Plan{}, fmt.Errorf("unknown managed status %q for target %q", st.Kind, st.File.Target)
		}
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return Plan{Actions: actions}, nil
}

// BuildUnlinkPlan computes file_unlink actions from discovered unlink status.
func BuildUnlinkPlan(statuses []discovery.UnlinkStatus) (Plan, error) {
	var actions []Action
	for _, st := range statuses {
		switch st.Kind {
		case discovery.UnlinkStatusRemove:
			actions = append(actions, Action{
				Type: ActionFileUnlink,
				Name: st.Path,
			})
		case discovery.UnlinkStatusMissing:
			continue
		default:
			return Plan{}, fmt.Errorf("unknown unlink status %q for path %q", st.Kind, st.Path)
		}
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return Plan{Actions: actions}, nil
}

// BuildFilePrunePlan emits file_prune for owned paths not in declaredTargets.
// declaredTargets should be links.targets ∪ managed.targets ∪ templates.targets ∪ unlink paths
// (explicit unlink already has its own action — do not also prune).
// Paths are expanded/normalized to absolute for comparison. Only candidates that
// still exist on disk are emitted. non_destructive yields an empty plan.
// Never prunes paths outside owned.
func BuildFilePrunePlan(owned []string, declaredTargets []string, mode config.FilesMode) Plan {
	switch mode {
	case config.FilesModeNonDestructive:
		return Plan{}
	case config.FilesModeSafe, config.FilesModeStrict, "":
		// safe/strict (and empty→safe) both show prune in the plan; apply differs later.
	default:
		// Unknown modes are treated like safe for planning after validation.
	}
	if len(owned) == 0 {
		return Plan{}
	}

	declared := make(map[string]struct{}, len(declaredTargets))
	for _, target := range declaredTargets {
		abs, err := normalizeFilePath(target)
		if err != nil || abs == "" {
			continue
		}
		declared[abs] = struct{}{}
	}

	var actions []Action
	for _, path := range owned {
		abs, err := normalizeFilePath(path)
		if err != nil || abs == "" {
			continue
		}
		if _, ok := declared[abs]; ok {
			continue
		}
		if _, err := os.Lstat(abs); err != nil {
			continue
		}
		actions = append(actions, Action{Type: ActionFilePrune, Name: abs})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].Name < actions[j].Name })
	return Plan{Actions: actions}
}

func normalizeFilePath(path string) (string, error) {
	expanded, err := expandHomePath(path)
	if err != nil {
		return "", err
	}
	if expanded == "" {
		return "", nil
	}
	if !filepath.IsAbs(expanded) {
		abs, err := filepath.Abs(expanded)
		if err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	return filepath.Clean(expanded), nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// MergePlans concatenates plans in order (brew first, then files).
func MergePlans(plans ...Plan) Plan {
	var actions []Action
	for _, p := range plans {
		actions = append(actions, p.Actions...)
	}
	return Plan{Actions: actions}
}
