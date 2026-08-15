package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/scaffold"
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
		} else if err := scaffold.InitConfigDir(cfgDir, false); err != nil {
			return err
		}
	}

	var runner discovery.Runner
	if flags.packages {
		runner = discovery.NewExecRunner()
	}
	result, err := engine.Import(cmd.Context(), runner, engine.ImportOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		Packages:   flags.packages,
		Files:      flags.files,
		DryRun:     flags.dryRun,
		Force:      flags.force,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	printImportResult(out, cmd.ErrOrStderr(), verbose, result)

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

func printImportResult(out, errOut io.Writer, verbose bool, result engine.ImportResult) {
	if result.PackagesDone {
		if result.ForceReplace {
			fmt.Fprintf(out, "packages: replace with %d taps, %d formulae, %d casks -> %s\n",
				len(result.Taps), len(result.Formulae), len(result.Casks), result.PackagesPath)
		} else {
			fmt.Fprintf(out, "packages: +%d taps, +%d formulae, +%d casks (total %d taps, %d formulae, %d casks) -> %s\n",
				len(result.AddedTaps), len(result.AddedFormulae), len(result.AddedCasks),
				len(result.Taps), len(result.Formulae), len(result.Casks), result.PackagesPath)
			for _, name := range result.AddedTaps {
				fmt.Fprintf(out, "  + tap %s\n", name)
			}
			for _, name := range result.AddedFormulae {
				fmt.Fprintf(out, "  + formula %s\n", name)
			}
			for _, name := range result.AddedCasks {
				fmt.Fprintf(out, "  + cask %s\n", name)
			}
		}
	}
	if result.FilesDone {
		if verbose {
			for _, target := range result.SkippedLinks {
				fmt.Fprintf(errOut, "skip existing link %s\n", target)
			}
		}
		for _, line := range result.FileLines {
			fmt.Fprintf(out, "file: %s -> %s\n", line.TargetDecl, line.RelSource)
		}
		if result.ForceReplace {
			fmt.Fprintf(out, "files: replace with %d link(s)\n", len(result.Links))
		} else {
			fmt.Fprintf(out, "files: +%d link(s) (total %d)\n", len(result.AddedLinks), len(result.Links))
		}
	}
	if result.WroteRoot {
		fmt.Fprintf(out, "writing %s\n", result.ConfigPath)
	}
}
