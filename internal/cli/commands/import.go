package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
		Short: "Import existing brew/mas packages and files into PourOver config",
		Long: `Import discovers installed Homebrew packages, Mac App Store apps
(via mas list), and common config/dotfile paths, then writes packages.lua
and files.links under ~/.pourover.

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
	cmd.Flags().BoolVar(&doPackages, "packages", true, "import installed brew formulae/casks and App Store apps into packages.lua")
	cmd.Flags().BoolVar(&doFiles, "files", true, "import existing config/dotfile paths into files.links")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview import without writing config or retargeting files")
	cmd.Flags().BoolVar(&force, "force", false, "replace packages/links with the discovered set (default: merge/add-only)")
	cmd.AddCommand(newImportMacOSCmd())
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
			fmt.Fprintf(out, "packages: replace with %d taps, %d formulae, %d casks, %d mas -> %s\n",
				len(result.Taps), len(result.Formulae), len(result.Casks), len(result.Mas), result.PackagesPath)
		} else {
			fmt.Fprintf(out, "packages: +%d taps, +%d formulae, +%d casks, +%d mas (total %d taps, %d formulae, %d casks, %d mas) -> %s\n",
				len(result.AddedTaps), len(result.AddedFormulae), len(result.AddedCasks), len(result.AddedMas),
				len(result.Taps), len(result.Formulae), len(result.Casks), len(result.Mas), result.PackagesPath)
			for _, name := range result.AddedTaps {
				fmt.Fprintf(out, "  + tap %s\n", name)
			}
			for _, name := range result.AddedFormulae {
				fmt.Fprintf(out, "  + formula %s\n", name)
			}
			for _, name := range result.AddedCasks {
				fmt.Fprintf(out, "  + cask %s\n", name)
			}
			for _, app := range result.AddedMas {
				fmt.Fprintf(out, "  + mas %s (%d)\n", app.Name, app.ID)
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

type importMacOSFlags struct {
	dryRun bool
	force  bool
}

func newImportMacOSCmd() *cobra.Command {
	var dryRun, force bool
	cmd := &cobra.Command{
		Use:   "macos",
		Short: "Import curated macOS defaults into macos.lua",
		Long: `Snapshot readable curated macOS defaults keys from the catalog into macos.lua.

By default merges newly discovered keys (add-only). Use --force to replace curated
macos.defaults sections with the snapshot. Use --dry-run to preview without writing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportMacOS(cmd, importMacOSFlags{dryRun: dryRun, force: force}, discovery.NewExecDefaultsRunner())
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview import without writing macos.lua or pourover.lua")
	cmd.Flags().BoolVar(&force, "force", false, "replace curated macos.defaults with the discovered set (default: merge/add-only)")
	return cmd
}

func runImportMacOS(cmd *cobra.Command, flags importMacOSFlags, runner discovery.DefaultsRunner) error {
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
		fmt.Fprintf(cmd.ErrOrStderr(), "import macos into %s\n", cfgDir)
	}

	result, err := engine.ImportMacOS(cmd.Context(), engine.ImportMacOSOptions{
		ConfigDir:  cfgDir,
		ConfigPath: configPath,
		DryRun:     flags.dryRun,
		Force:      flags.force,
		Runner:     runner,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	printImportMacOSResult(out, cmd.ErrOrStderr(), verbose, result)

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

const importMacOSPreviewLines = 40

func printImportMacOSResult(out, errOut io.Writer, verbose bool, result engine.ImportMacOSResult) {
	mode := "merge"
	if result.Force {
		mode = "replace"
	}
	if result.DryRun {
		fmt.Fprintf(out, "macos: read %d key(s), +%d (%s); would write %s\n",
			result.ReadCount, result.Added, mode, result.MacOSPath)
	} else {
		fmt.Fprintf(out, "macos: read %d key(s), +%d (%s) -> %s\n",
			result.ReadCount, result.Added, mode, result.MacOSPath)
	}
	if result.HasSystemScope && result.AdminNote != "" {
		fmt.Fprintln(out, result.AdminNote)
	}
	if result.EnsuredRequire {
		fmt.Fprintln(out, `updated pourover.lua to require("macos")`)
	}
	if verbose {
		for _, w := range result.Warnings {
			fmt.Fprintf(errOut, "warning: %s\n", w)
		}
		if result.DryRun && result.Lua != "" {
			printLuaPreview(out, result.Lua, importMacOSPreviewLines)
		}
	}
}

func printLuaPreview(out io.Writer, lua string, maxLines int) {
	lines := strings.Split(lua, "\n")
	n := len(lines)
	truncated := false
	if n > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}
	fmt.Fprintln(out, "--- macos.lua preview ---")
	for _, line := range lines {
		fmt.Fprintln(out, line)
	}
	if truncated {
		fmt.Fprintf(out, "... (%d more lines)\n", n-maxLines)
	}
}
