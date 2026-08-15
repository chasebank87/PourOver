package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/pam"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/state"
	tmpl "github.com/chasebank87/PourOver/internal/template"
)

// BuildPlan loads config at configPath, discovers current state via runner, and
// returns the merged reconcile plan. Defaults discovery uses the system
// defaults runner (same as the former CLI buildPlan). State is read from
// paths.DefaultStateDir for owned-file prune.
func BuildPlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	return BuildPlanWith(ctx, configPath, runner, discovery.NewExecDefaultsRunner(), "")
}

// BuildPlanWith is like BuildPlan but accepts an explicit DefaultsRunner
// (useful for tests that stub macOS defaults) and optional stateDir.
// When stateDir is empty, paths.DefaultStateDir is used for LoadLock / prune.
func BuildPlanWith(ctx context.Context, configPath string, runner discovery.Runner, defaultsRunner discovery.DefaultsRunner, stateDir string) (plan.Plan, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("load config: %w", err)
	}

	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover brew: %w", err)
	}
	pamCfg := manifest.MacOS.Security.PAM.SudoLocal
	packages := plan.ExpandPAMFormulae(manifest.Packages, pamCfg)
	brewPlan := plan.BuildBrewPlan(packages, brewState)
	brewPlan, err = plan.AdviseCaskRenames(ctx, runner, brewPlan, brewState.Casks)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("detect cask renames: %w", err)
	}

	pamPlan, err := buildPAMPlan(ctx, runner, pamCfg, plan.DefaultPAMSudoLocalPath, plan.DefaultPAMSudoPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("plan pam: %w", err)
	}

	desired := config.FlattenDefaults(manifest.MacOS.Defaults)
	statuses, err := discovery.DiscoverDefaults(ctx, defaultsRunner, desired)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover macos defaults: %w", err)
	}
	defaultsPlan := plan.BuildDefaultsPlan(statuses)

	configDir := filepath.Dir(configPath)
	replaceMode := policy.ResolveFileReplaceFromManifest(manifest)
	linkStatuses, err := discovery.DiscoverFileLinks(manifest.Files.Links, configDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover files: %w", err)
	}
	filePlan, err := plan.BuildFilePlan(linkStatuses, replaceMode)
	if err != nil {
		return plan.Plan{}, err
	}

	managedStatuses, err := discovery.DiscoverManagedFiles(manifest.Files.Managed, configDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover managed files: %w", err)
	}
	managedPlan, err := plan.BuildManagedPlan(managedStatuses, replaceMode)
	if err != nil {
		return plan.Plan{}, err
	}

	tmplCtx, err := tmpl.DefaultContext()
	if err != nil {
		return plan.Plan{}, fmt.Errorf("template context: %w", err)
	}
	templateStatuses, err := discovery.DiscoverTemplateFiles(manifest.Files.Templates, configDir, tmplCtx)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover templates: %w", err)
	}
	templatePlan, err := plan.BuildTemplatePlan(templateStatuses, replaceMode)
	if err != nil {
		return plan.Plan{}, err
	}

	unlinkStatuses, err := discovery.DiscoverUnlinkPaths(manifest.Files.Unlink, configDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("discover unlink paths: %w", err)
	}
	unlinkPlan, err := plan.BuildUnlinkPlan(unlinkStatuses)
	if err != nil {
		return plan.Plan{}, err
	}

	if stateDir == "" {
		stateDir, err = paths.DefaultStateDir()
		if err != nil {
			return plan.Plan{}, fmt.Errorf("state directory: %w", err)
		}
	}
	lock, err := state.LoadLock(stateDir)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("load lock: %w", err)
	}
	filesMode := policy.ResolveFilesModeFromManifest(manifest)
	prunePlan := plan.BuildFilePrunePlan(lock.OwnedFiles, declaredFileTargets(manifest), filesMode)

	// brew → pam → macos defaults → file links → managed copies → templates → unlinks → prune
	return plan.MergePlans(brewPlan, pamPlan, defaultsPlan, filePlan, managedPlan, templatePlan, unlinkPlan, prunePlan), nil
}

func buildPAMPlan(ctx context.Context, runner discovery.Runner, cfg config.SudoLocalPAM, sudoLocalPath, sudoPath string) (plan.Plan, error) {
	if !cfg.Configured {
		return plan.Plan{}, nil
	}

	reattachPath, watchidPath, err := resolvePAMModulePaths(ctx, runner, cfg)
	if err != nil {
		return plan.Plan{}, err
	}

	sudoLocalContent, sudoLocalExists, err := readOptionalFile(sudoLocalPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("read %s: %w", sudoLocalPath, err)
	}
	sudoContent, _, err := readOptionalFile(sudoPath)
	if err != nil {
		return plan.Plan{}, fmt.Errorf("read %s: %w", sudoPath, err)
	}

	return plan.BuildPAMPlan(plan.PAMDiffInput{
		Config:           cfg,
		SudoLocalPath:    sudoLocalPath,
		SudoPath:         sudoPath,
		SudoLocalContent: sudoLocalContent,
		SudoLocalExists:  sudoLocalExists,
		SudoContent:      sudoContent,
		ReattachPath:     reattachPath,
		WatchIDPath:      watchidPath,
	}), nil
}

func resolvePAMModulePaths(ctx context.Context, runner discovery.Runner, cfg config.SudoLocalPAM) (reattach, watchid string, err error) {
	if !cfg.Enable {
		return "", "", nil
	}
	if cfg.Reattach {
		prefix, err := discovery.BrewPrefix(ctx, runner, "pam-reattach")
		if err != nil {
			return "", "", err
		}
		reattach = pam.ModulePath(prefix, "pam_reattach.so")
	}
	if cfg.WatchIDAuth {
		// pam-watchid is not a Homebrew core formula; search common install paths.
		brewRoot, _ := discovery.BrewPrefix(ctx, runner, "")
		candidates := pam.DefaultWatchIDSearchPaths(brewRoot)
		found, ok := pam.FindModule(candidates)
		if !ok {
			return "", "", fmt.Errorf("watch_id_auth is enabled but pam_watchid.so was not found (searched: %s); install pam-watchid manually (e.g. mostlygeek/pam-watchid) — it is not a Homebrew core formula", strings.Join(candidates, ", "))
		}
		watchid = found
	}
	return reattach, watchid, nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// declaredFileTargets returns paths that should not be pruned: current links,
// managed, and template targets, plus explicit unlink paths (those get file_unlink instead).
func declaredFileTargets(m config.Manifest) []string {
	n := len(m.Files.Links) + len(m.Files.Managed) + len(m.Files.Templates) + len(m.Files.Unlink)
	out := make([]string, 0, n)
	for _, link := range m.Files.Links {
		out = append(out, link.Target)
	}
	for _, file := range m.Files.Managed {
		out = append(out, file.Target)
	}
	for _, file := range m.Files.Templates {
		out = append(out, file.Target)
	}
	out = append(out, m.Files.Unlink...)
	return out
}
