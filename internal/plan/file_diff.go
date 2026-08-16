package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/paths"
)

// BuildGenerationFilePlan emits file actions from generation blob vs live status.
// Value holds the content hash so apply can read the generation blob.
// Links use link_*; managed use managed_copy; templates use template_write.
// Symlink targets appear as Differ and become update/create writes (not symlinks).
func BuildGenerationFilePlan(statuses []generation.FileStatus, replaceMode config.FileReplaceMode) (Plan, error) {
	var actions []Action
	for _, st := range statuses {
		switch st.Kind {
		case generation.FileStatusSame:
			continue
		case generation.FileStatusMissing, generation.FileStatusDiffer:
			actions = append(actions, genFileAction(st, false))
		case generation.FileStatusBlocked:
			if replaceMode == config.FileReplaceBackup {
				a := genFileAction(st, true)
				actions = append(actions, a)
				continue
			}
			return Plan{}, fmt.Errorf(
				"target %q exists and is not a replaceable file (source %q)",
				st.Entry.Target, st.Entry.Source,
			)
		default:
			return Plan{}, fmt.Errorf("unknown generation file status %q for target %q", st.Kind, st.Entry.Target)
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Type != actions[j].Type {
			return actions[i].Type < actions[j].Type
		}
		return actions[i].Name < actions[j].Name
	})
	return Plan{Actions: actions}, nil
}

func genFileAction(st generation.FileStatus, backup bool) Action {
	e := st.Entry
	a := Action{
		Name:   e.Target,
		Source: e.Source,
		Value:  e.Hash,
		Kind:   e.Mode,
	}
	if backup {
		a.Kind = "backup"
	}
	switch e.Kind {
	case generation.FileKindManaged:
		a.Type = ActionManagedCopy
	case generation.FileKindTemplate:
		a.Type = ActionTemplateWrite
	default:
		switch st.Kind {
		case generation.FileStatusMissing:
			a.Type = ActionLinkCreate
		case generation.FileStatusBlocked:
			a.Type = ActionLinkReplace
		default:
			a.Type = ActionLinkUpdate
		}
	}
	return a
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
		if paths.SkipOwnedPath(abs) {
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
