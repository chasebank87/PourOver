package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configimport"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/spf13/cobra"
)

// NewImportCmd returns the import subcommand.
func NewImportCmd() *cobra.Command {
	var (
		doPackages bool
		doFiles    bool
		dryRun     bool
		force      bool
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import existing brew packages and files into PourOver config",
		Long: `Import discovers installed Homebrew packages and common config/dotfile
paths, then writes packages.lua and files.links under ~/.pourover.

By default a re-import merges: newly discovered packages and file targets are
added; existing declarations are kept (nothing is removed from config).
Use --force to replace packages/links with the discovered set only.
Use --dry-run to preview.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(cmd, importFlags{
				packages: doPackages,
				files:    doFiles,
				dryRun:   dryRun,
				force:    force,
			})
		},
	}
	cmd.Flags().BoolVar(&doPackages, "packages", true, "import installed brew formulae and casks into packages.lua")
	cmd.Flags().BoolVar(&doFiles, "files", true, "import existing config/dotfile paths into files.links")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview import without writing config or retargeting files")
	cmd.Flags().BoolVar(&force, "force", false, "replace packages/links with the discovered set (default: merge/add-only)")
	return cmd
}

type importFlags struct {
	packages bool
	files    bool
	dryRun   bool
	force    bool
}

func runImport(cmd *cobra.Command, flags importFlags) error {
	if !flags.packages && !flags.files {
		return fmt.Errorf("nothing to import: enable --packages and/or --files")
	}

	configFlag, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return err
	}
	verbose, err := cmd.Root().PersistentFlags().GetBool("verbose")
	if err != nil {
		return err
	}

	var cfgDir, configPath string
	if configFlag != "" {
		configPath = configFlag
		cfgDir = filepath.Dir(configPath)
	} else {
		cfgDir, err = paths.DefaultConfigDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(cfgDir, paths.ConfigFileName)
	}

	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "import into %s\n", cfgDir)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if flags.dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "would scaffold config in %s\n", cfgDir)
		} else if err := InitConfigDir(cfgDir, false); err != nil {
			return err
		}
	}

	policy := config.Policy{UninstallMode: config.UninstallModeSafe}
	backup := config.Backup{}
	existingLinks := []config.FileLink{}
	var existingFormulae, existingCasks []string
	if _, err := os.Stat(configPath); err == nil {
		if m, loadErr := config.LoadManifest(configPath); loadErr == nil {
			policy = m.Policy
			backup = m.Backup
			existingLinks = append([]config.FileLink(nil), m.Files.Links...)
			existingFormulae = append([]string(nil), m.Packages.Formulae...)
			existingCasks = append([]string(nil), m.Packages.Casks...)
		}
	}

	out := cmd.OutOrStdout()
	links := existingLinks
	filesChanged := false

	if flags.packages {
		runner := discovery.NewExecRunner()
		state, err := discovery.DiscoverBrew(cmd.Context(), runner)
		if err != nil {
			return fmt.Errorf("discover brew: %w", err)
		}
		var formulae, casks []string
		var addedF, addedC []string
		if flags.force {
			formulae = append([]string(nil), state.FormulaeRequested...)
			casks = append([]string(nil), state.Casks...)
			fmt.Fprintf(out, "packages: replace with %d formulae, %d casks -> %s\n",
				len(formulae), len(casks), filepath.Join(cfgDir, "packages.lua"))
		} else {
			formulae, addedF = configimport.MergePackageLists(existingFormulae, state.FormulaeRequested)
			casks, addedC = configimport.MergePackageLists(existingCasks, state.Casks)
			fmt.Fprintf(out, "packages: +%d formulae, +%d casks (total %d formulae, %d casks) -> %s\n",
				len(addedF), len(addedC), len(formulae), len(casks), filepath.Join(cfgDir, "packages.lua"))
			for _, name := range addedF {
				fmt.Fprintf(out, "  + formula %s\n", name)
			}
			for _, name := range addedC {
				fmt.Fprintf(out, "  + cask %s\n", name)
			}
		}
		pkgPath := filepath.Join(cfgDir, "packages.lua")
		body := configimport.FormatPackagesLua(formulae, casks)
		if !flags.dryRun {
			if err := os.WriteFile(pkgPath, []byte(body), 0o644); err != nil {
				return err
			}
		}
	}

	if flags.files {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		candidates, err := configimport.ExistingImportable(configimport.DefaultHomeCandidates(home))
		if err != nil {
			return err
		}
		declared := configimport.LinkTargets(existingLinks)
		var imported []config.FileLink
		for _, c := range candidates {
			if !flags.force {
				if _, ok := declared[c.TargetDecl]; ok {
					if verbose {
						fmt.Fprintf(cmd.ErrOrStderr(), "skip existing link %s\n", c.TargetDecl)
					}
					continue
				}
			}
			fmt.Fprintf(out, "file: %s -> %s\n", c.TargetDecl, c.RelSource)
			if flags.dryRun {
				imported = append(imported, config.FileLink{Source: c.RelSource, Target: c.TargetDecl})
				continue
			}
			link, err := configimport.ImportFile(cfgDir, c, true)
			if err != nil {
				return fmt.Errorf("import %s: %w", c.TargetPath, err)
			}
			imported = append(imported, link)
		}
		if flags.force {
			links = imported
			fmt.Fprintf(out, "files: replace with %d link(s)\n", len(links))
		} else {
			var added []config.FileLink
			links, added = configimport.MergeFileLinks(existingLinks, imported)
			fmt.Fprintf(out, "files: +%d link(s) (total %d)\n", len(added), len(links))
		}
		filesChanged = true
	}

	if filesChanged {
		rootBody := configimport.FormatRootLua(links, policy, backup)
		fmt.Fprintf(out, "writing %s\n", configPath)
		if !flags.dryRun {
			if err := os.WriteFile(configPath, []byte(rootBody), 0o644); err != nil {
				return err
			}
		}
	}

	if flags.dryRun {
		fmt.Fprintln(out, "Dry run only; no files were modified.")
	} else {
		fmt.Fprintln(out, "Import complete. Run `pourover plan` to review.")
		if m, err := config.LoadManifest(configPath); err == nil {
			maybeAutoPushConfig(cmd, configPath, m)
		}
	}
	return nil
}
