package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Home       string              // optional; defaults to os.UserHomeDir
	MasRunner  discovery.MasRunner // nil → NewExecMasRunner for package import

	// FileTargets limits file import to these TargetDecls (or absolute paths).
	// When set, interactive FileSelect and FilesAll are ignored for filtering.
	FileTargets []string
	// FilesAll imports every existing candidate (CI / --files-all).
	FilesAll bool
	// FileSelect optionally filters candidates after discovery (interactive picker).
	// Receives candidates not already skipped as managed (unless Force).
	FileSelect func(candidates []configimport.FileCandidate) ([]configimport.FileCandidate, error)
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
	Mas           []config.MasApp
	AddedTaps     []string
	AddedFormulae []string
	AddedCasks    []string
	AddedMas      []config.MasApp

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

	existingLinks := []config.FileLink{}
	var existingTaps, existingFormulae, existingCasks []string
	var existingMas []config.MasApp
	masConfigured := false
	if _, err := os.Stat(opts.ConfigPath); err == nil {
		if m, loadErr := config.LoadManifest(opts.ConfigPath); loadErr == nil {
			existingLinks = append([]config.FileLink(nil), m.Files.Links...)
			existingTaps = append([]string(nil), m.Packages.TapNames()...)
			existingFormulae = append([]string(nil), m.Packages.Formulae...)
			existingCasks = append([]string(nil), m.Packages.Casks...)
			existingMas = append([]config.MasApp(nil), m.Packages.Mas...)
			masConfigured = m.Packages.MasConfigured
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

		mas, addedMas, writeMas, err := importMasApps(ctx, opts, existingMas, masConfigured)
		if err != nil {
			return ImportResult{}, err
		}
		var masForFormat []config.MasApp // nil omits mas key
		if writeMas {
			masForFormat = mas
			if masForFormat == nil {
				masForFormat = []config.MasApp{}
			}
		}

		pkgPath := filepath.Join(opts.ConfigDir, "packages.lua")
		body := configimport.FormatPackagesLuaFull(taps, formulae, casks, masForFormat)
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
		result.Mas = mas
		result.AddedTaps = addedT
		result.AddedFormulae = addedF
		result.AddedCasks = addedC
		result.AddedMas = addedMas
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

		var selectable []configimport.FileCandidate
		var skipped []string
		for _, c := range candidates {
			if !opts.Force {
				if _, ok := declared[c.TargetDecl]; ok {
					skipped = append(skipped, c.TargetDecl)
					continue
				}
			}
			selectable = append(selectable, c)
		}

		chosen, err := selectFileCandidates(opts, selectable)
		if err != nil {
			return ImportResult{}, err
		}

		var imported []config.FileLink
		var lines []ImportFileLine
		for _, c := range chosen {
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
			// Force replaces the full link set with newly imported plus any
			// previously managed targets that were not re-selected only when
			// FilesAll/explicit selection covers the intended set. Callers that
			// pass FileTargets or FileSelect under --force update only those
			// targets and keep other existing links.
			if opts.FilesAll && len(opts.FileTargets) == 0 && opts.FileSelect == nil {
				links = imported
			} else {
				links, added = mergeForceSelectedLinks(existingLinks, imported)
			}
		} else {
			links, added = configimport.MergeFileLinks(existingLinks, imported)
		}
		result.FilesDone = true
		result.FileLines = lines
		result.SkippedLinks = skipped
		result.Links = links
		result.AddedLinks = added
		filesChanged = len(imported) > 0 || (opts.Force && opts.FilesAll && len(opts.FileTargets) == 0 && opts.FileSelect == nil)
	}

	if filesChanged {
		result.WroteRoot = true
		if !opts.DryRun {
			if err := config.PatchFilesLinksFile(opts.ConfigPath, links); err != nil {
				return ImportResult{}, err
			}
		}
	}

	return result, nil
}

func selectFileCandidates(opts ImportOptions, selectable []configimport.FileCandidate) ([]configimport.FileCandidate, error) {
	if len(opts.FileTargets) > 0 {
		chosen := configimport.FilterCandidatesByTargets(selectable, opts.FileTargets)
		if len(chosen) == 0 {
			return nil, fmt.Errorf("no matching file targets among %d candidate(s)", len(selectable))
		}
		return chosen, nil
	}
	if opts.FileSelect != nil {
		return opts.FileSelect(selectable)
	}
	if opts.FilesAll {
		return selectable, nil
	}
	// No selection strategy: import nothing rather than silently owning ~/.config.
	if len(selectable) == 0 {
		return nil, nil
	}
	return nil, fmt.Errorf("file import requires --target, --files-all, or an interactive selection")
}

// mergeForceSelectedLinks updates or inserts imported links while keeping other
// existing declarations (used when --force applies to a subset).
func mergeForceSelectedLinks(existing, imported []config.FileLink) (all, added []config.FileLink) {
	byTarget := make(map[string]config.FileLink, len(existing))
	order := make([]string, 0, len(existing)+len(imported))
	for _, l := range existing {
		byTarget[l.Target] = l
		order = append(order, l.Target)
	}
	for _, l := range imported {
		if _, ok := byTarget[l.Target]; !ok {
			order = append(order, l.Target)
			added = append(added, l)
		}
		byTarget[l.Target] = l
	}
	all = make([]config.FileLink, 0, len(order))
	for _, t := range order {
		all = append(all, byTarget[t])
	}
	return all, added
}

// importMasApps discovers installed Mac App Store apps and merges or replaces
// them per Force. When mas is missing from PATH, MAS is soft-skipped: brew
// packages still import, and any existing configured mas list is preserved.
func importMasApps(ctx context.Context, opts ImportOptions, existing []config.MasApp, configured bool) (mas, added []config.MasApp, writeMas bool, err error) {
	masRunner := opts.MasRunner
	if masRunner == nil {
		masRunner = discovery.NewExecMasRunner()
	}
	state, err := discovery.DiscoverMas(ctx, masRunner)
	if err != nil {
		if discovery.IsMasNotFound(err) {
			if configured {
				return existing, nil, true, nil
			}
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("discover mas: %w", err)
	}

	discovered := make([]config.MasApp, 0, len(state.Apps))
	for _, app := range state.Apps {
		discovered = append(discovered, config.MasApp{Name: app.Name, ID: app.ID})
	}

	if opts.Force {
		mas = append([]config.MasApp(nil), discovered...)
		writeMas = configured || len(mas) > 0
		return mas, nil, writeMas, nil
	}

	mas, added = configimport.MergeMasApps(existing, discovered)
	writeMas = configured || len(mas) > 0
	return mas, added, writeMas, nil
}

// UnmanageFilesOptions configures dropping file links without deleting live paths.
type UnmanageFilesOptions struct {
	ConfigPath string
	StateDir   string
	Targets    []string // TargetDecl or absolute paths
}

// UnmanageFilesResult summarizes what was removed from config/ownership.
type UnmanageFilesResult struct {
	RemovedLinks []config.FileLink
	ClearedOwned []string
}

// UnmanageFiles removes files.links entries for the given targets, clears matching
// owned_files entries (including paths under a directory link), and leaves live
// disk and ~/.pourover source copies untouched.
func UnmanageFiles(opts UnmanageFilesOptions) (UnmanageFilesResult, error) {
	var out UnmanageFilesResult
	if opts.ConfigPath == "" {
		return out, fmt.Errorf("config path is required")
	}
	if len(opts.Targets) == 0 {
		return out, fmt.Errorf("at least one target is required")
	}
	m, err := config.LoadManifest(opts.ConfigPath)
	if err != nil {
		return out, err
	}

	want := make(map[string]struct{}, len(opts.Targets))
	for _, t := range opts.Targets {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		want[configimport.NormalizeTargetDecl(t)] = struct{}{}
	}

	var kept []config.FileLink
	var removed []config.FileLink
	for _, link := range m.Files.Links {
		decl := configimport.NormalizeTargetDecl(link.Target)
		if _, ok := want[decl]; ok {
			removed = append(removed, link)
			continue
		}
		kept = append(kept, link)
	}
	if len(removed) == 0 {
		return out, fmt.Errorf("no managed files.links matched the given targets")
	}
	if err := config.PatchFilesLinksFile(opts.ConfigPath, kept); err != nil {
		return out, err
	}
	out.RemovedLinks = removed

	if opts.StateDir == "" {
		return out, nil
	}
	cleared, err := clearOwnedForRemovedLinks(opts.StateDir, removed)
	if err != nil {
		return out, err
	}
	out.ClearedOwned = cleared
	return out, nil
}
