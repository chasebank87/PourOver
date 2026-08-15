package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configimport"
	"github.com/chasebank87/PourOver/internal/discovery"
)

// ImportOptions configures importing brew packages and/or file links into config.
type ImportOptions struct {
	ConfigDir  string
	ConfigPath string
	Packages   bool
	Files      bool
	DryRun     bool
	Force      bool
	Home       string // optional; defaults to os.UserHomeDir
}

// ImportFileLine is one discovered file candidate considered for import.
type ImportFileLine struct {
	TargetDecl string
	RelSource  string
}

// ImportResult is a structured summary for frontends to print.
type ImportResult struct {
	DryRun     bool
	ConfigDir  string
	ConfigPath string

	PackagesDone  bool
	ForceReplace  bool
	PackagesPath  string
	Taps          []string
	Formulae      []string
	Casks         []string
	AddedTaps     []string
	AddedFormulae []string
	AddedCasks    []string

	FilesDone    bool
	FileLines    []ImportFileLine
	SkippedLinks []string // target decls skipped because already declared (merge mode)
	Links        []config.FileLink
	AddedLinks   []config.FileLink
	WroteRoot    bool
}

// Import discovers brew packages and/or home config files and writes PourOver config.
// Callers should scaffold a missing config directory before Import when not dry-running.
func Import(ctx context.Context, runner discovery.Runner, opts ImportOptions) (ImportResult, error) {
	if !opts.Packages && !opts.Files {
		return ImportResult{}, fmt.Errorf("nothing to import: enable packages and/or files")
	}
	if opts.ConfigDir == "" || opts.ConfigPath == "" {
		return ImportResult{}, fmt.Errorf("config dir and path are required")
	}

	result := ImportResult{
		DryRun:       opts.DryRun,
		ConfigDir:    opts.ConfigDir,
		ConfigPath:   opts.ConfigPath,
		ForceReplace: opts.Force,
	}

	policy := config.Policy{UninstallMode: config.UninstallModeSafe}
	backupCfg := config.Backup{}
	existingLinks := []config.FileLink{}
	var existingTaps, existingFormulae, existingCasks []string
	if _, err := os.Stat(opts.ConfigPath); err == nil {
		if m, loadErr := config.LoadManifest(opts.ConfigPath); loadErr == nil {
			policy = m.Policy
			backupCfg = m.Backup
			existingLinks = append([]config.FileLink(nil), m.Files.Links...)
			existingTaps = append([]string(nil), m.Packages.Taps...)
			existingFormulae = append([]string(nil), m.Packages.Formulae...)
			existingCasks = append([]string(nil), m.Packages.Casks...)
		}
	}

	links := existingLinks
	filesChanged := false

	if opts.Packages {
		if runner == nil {
			return ImportResult{}, fmt.Errorf("brew runner is required for package import")
		}
		state, err := discovery.DiscoverBrew(ctx, runner)
		if err != nil {
			return ImportResult{}, fmt.Errorf("discover brew: %w", err)
		}
		discoveredTaps := discovery.DeclarableTaps(state.Taps)
		var taps, formulae, casks []string
		var addedT, addedF, addedC []string
		if opts.Force {
			taps = append([]string(nil), discoveredTaps...)
			formulae = append([]string(nil), state.FormulaeRequested...)
			casks = append([]string(nil), state.Casks...)
		} else {
			taps, addedT = configimport.MergePackageLists(existingTaps, discoveredTaps)
			formulae, addedF = configimport.MergePackageLists(existingFormulae, state.FormulaeRequested)
			casks, addedC = configimport.MergePackageLists(existingCasks, state.Casks)
		}
		pkgPath := filepath.Join(opts.ConfigDir, "packages.lua")
		body := configimport.FormatPackagesLuaFull(taps, formulae, casks)
		if !opts.DryRun {
			if err := os.WriteFile(pkgPath, []byte(body), 0o644); err != nil {
				return ImportResult{}, err
			}
		}
		result.PackagesDone = true
		result.PackagesPath = pkgPath
		result.Taps = taps
		result.Formulae = formulae
		result.Casks = casks
		result.AddedTaps = addedT
		result.AddedFormulae = addedF
		result.AddedCasks = addedC
	}

	if opts.Files {
		home := opts.Home
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return ImportResult{}, err
			}
		}
		candidates, err := configimport.ExistingImportable(configimport.DefaultHomeCandidates(home))
		if err != nil {
			return ImportResult{}, err
		}
		declared := configimport.LinkTargets(existingLinks)
		var imported []config.FileLink
		var lines []ImportFileLine
		var skipped []string
		for _, c := range candidates {
			if !opts.Force {
				if _, ok := declared[c.TargetDecl]; ok {
					skipped = append(skipped, c.TargetDecl)
					continue
				}
			}
			lines = append(lines, ImportFileLine{TargetDecl: c.TargetDecl, RelSource: c.RelSource})
			if opts.DryRun {
				imported = append(imported, config.FileLink{Source: c.RelSource, Target: c.TargetDecl})
				continue
			}
			link, err := configimport.ImportFile(opts.ConfigDir, c, true)
			if err != nil {
				return ImportResult{}, fmt.Errorf("import %s: %w", c.TargetPath, err)
			}
			imported = append(imported, link)
		}
		var added []config.FileLink
		if opts.Force {
			links = imported
		} else {
			links, added = configimport.MergeFileLinks(existingLinks, imported)
		}
		result.FilesDone = true
		result.FileLines = lines
		result.SkippedLinks = skipped
		result.Links = links
		result.AddedLinks = added
		filesChanged = true
	}

	if filesChanged {
		rootBody := configimport.FormatRootLua(links, policy, backupCfg)
		result.WroteRoot = true
		if !opts.DryRun {
			if err := os.WriteFile(opts.ConfigPath, []byte(rootBody), 0o644); err != nil {
				return ImportResult{}, err
			}
		}
	}

	return result, nil
}
