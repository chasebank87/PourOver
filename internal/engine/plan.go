package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/pam"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/policy"
	"github.com/chasebank87/PourOver/internal/state"
)

// PlanResult is a reconcile plan plus the generation it was built from.
type PlanResult struct {
	Plan         plan.Plan
	Manifest     config.Manifest
	GenerationID string
	Generation   generation.Manifest
}

// BuildPlan loads config at configPath, discovers current state via runner, and
// returns the merged reconcile plan. Defaults discovery uses the system
// defaults runner (same as the former CLI buildPlan). State is read from
// paths.DefaultStateDir for owned-file prune. MAS discovery uses
// discovery.NewExecMasRunner when packages.mas is configured.
func BuildPlan(ctx context.Context, configPath string, runner discovery.Runner) (plan.Plan, error) {
	res, err := BuildPlanResult(ctx, configPath, runner, discovery.NewExecDefaultsRunner(), nil, "", time.Now())
	if err != nil {
		return plan.Plan{}, err
	}
	return res.Plan, nil
}

// BuildPlanWith is like BuildPlan but accepts explicit DefaultsRunner and
// MasRunner (useful for tests) and optional stateDir.
// When stateDir is empty, paths.DefaultStateDir is used for LoadLock / prune.
// When masRunner is nil and packages.mas is configured, NewExecMasRunner is used.
// When packages.mas is not configured, DiscoverMas is skipped.
// When DiscoverMas fails because mas is not on PATH, planning continues with
// an empty MasState so the implied formula_install mas can bootstrap.
func BuildPlanWith(ctx context.Context, configPath string, runner discovery.Runner, defaultsRunner discovery.DefaultsRunner, masRunner discovery.MasRunner, stateDir string) (plan.Plan, error) {
	res, err := BuildPlanResult(ctx, configPath, runner, defaultsRunner, masRunner, stateDir, time.Now())
	if err != nil {
		return plan.Plan{}, err
	}
	return res.Plan, nil
}

// BuildGeneration evaluates Lua and writes a new activation generation (no live writes).
func BuildGeneration(configPath, stateDir string, at time.Time) (generation.BuildResult, config.Manifest, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return generation.BuildResult{}, config.Manifest{}, fmt.Errorf("load config: %w", err)
	}
	if stateDir == "" {
		stateDir, err = paths.DefaultStateDir()
		if err != nil {
			return generation.BuildResult{}, config.Manifest{}, fmt.Errorf("state directory: %w", err)
		}
	}
	if at.IsZero() {
		at = time.Now()
	}
	res, err := generation.Build(stateDir, filepath.Dir(configPath), manifest, at)
	if err != nil {
		return generation.BuildResult{}, config.Manifest{}, err
	}
	return res, manifest, nil
}

// BuildPlanResult builds a generation then plans live state against it.
func BuildPlanResult(ctx context.Context, configPath string, runner discovery.Runner, defaultsRunner discovery.DefaultsRunner, masRunner discovery.MasRunner, stateDir string, at time.Time) (PlanResult, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return PlanResult{}, fmt.Errorf("load config: %w", err)
	}

	if stateDir == "" {
		stateDir, err = paths.DefaultStateDir()
		if err != nil {
			return PlanResult{}, fmt.Errorf("state directory: %w", err)
		}
	}
	if at.IsZero() {
		at = time.Now()
	}
	configDir := filepath.Dir(configPath)
	genRes, err := generation.Build(stateDir, configDir, manifest, at)
	if err != nil {
		return PlanResult{}, fmt.Errorf("build generation: %w", err)
	}
	gen := genRes.Manifest

	brewState, err := discovery.DiscoverBrew(ctx, runner)
	if err != nil {
		return PlanResult{}, fmt.Errorf("discover brew: %w", err)
	}
	pamCfg := gen.MacOS.Security.PAM.SudoLocal
	packages := plan.ExpandPAMFormulae(gen.Packages, pamCfg)
	packages = plan.ExpandMasFormulae(packages)
	deps, err := discovery.FormulaDependencyClosure(ctx, runner, packages.Formulae)
	if err != nil {
		return PlanResult{}, fmt.Errorf("formula deps: %w", err)
	}
	brewState.ProtectedFormulae = deps
	brewPlan := plan.BuildBrewPlan(packages, brewState)
	brewPlan, err = plan.AdviseCaskRenames(ctx, runner, brewPlan, brewState.Casks)
	if err != nil {
		return PlanResult{}, fmt.Errorf("detect cask renames: %w", err)
	}

	var masPlan plan.Plan
	if packages.MasConfigured {
		if masRunner == nil {
			masRunner = discovery.NewExecMasRunner()
		}
		masState, err := discovery.DiscoverMas(ctx, masRunner)
		if err != nil {
			if discovery.IsMasNotFound(err) {
				masState = discovery.MasState{}
			} else {
				return PlanResult{}, fmt.Errorf("discover mas: %w", err)
			}
		}
		masPlan = plan.BuildMasPlan(packages, masState)
	}

	pamPlan, err := buildPAMPlan(ctx, runner, pamCfg, plan.DefaultPAMSudoLocalPath, plan.DefaultPAMSudoPath)
	if err != nil {
		return PlanResult{}, fmt.Errorf("plan pam: %w", err)
	}

	desired := config.FlattenDefaults(gen.MacOS.Defaults)
	statuses, err := discovery.DiscoverDefaults(ctx, defaultsRunner, desired)
	if err != nil {
		return PlanResult{}, fmt.Errorf("discover macos defaults: %w", err)
	}
	defaultsPlan := plan.BuildDefaultsPlan(statuses)

	replaceMode := policy.ResolveFileReplace(string(gen.Policy.FileReplace))
	genStatuses, err := generation.DiscoverFiles(gen.Files)
	if err != nil {
		return PlanResult{}, fmt.Errorf("discover generation files: %w", err)
	}
	filePlan, err := plan.BuildGenerationFilePlan(genStatuses, replaceMode)
	if err != nil {
		return PlanResult{}, err
	}

	unlinkStatuses, err := discovery.DiscoverUnlinkPaths(gen.Unlink, configDir)
	if err != nil {
		return PlanResult{}, fmt.Errorf("discover unlink paths: %w", err)
	}
	unlinkPlan, err := plan.BuildUnlinkPlan(unlinkStatuses)
	if err != nil {
		return PlanResult{}, err
	}

	lock, err := state.LoadLock(stateDir)
	if err != nil {
		return PlanResult{}, fmt.Errorf("load lock: %w", err)
	}
	filesMode := policy.ResolveFilesMode(string(gen.Policy.FilesMode))
	prunePlan := plan.BuildFilePrunePlan(lock.OwnedFiles, generation.DeclaredTargets(gen), filesMode)

	// brew → mas → pam → macos defaults → generation files → unlinks → prune
	merged := plan.MergePlans(brewPlan, masPlan, pamPlan, defaultsPlan, filePlan, unlinkPlan, prunePlan)
	return PlanResult{
		Plan:         merged,
		Manifest:     manifest,
		GenerationID: gen.ID,
		Generation:   gen,
	}, nil
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
