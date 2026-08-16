package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
	"github.com/chasebank87/PourOver/internal/ui"
	"github.com/spf13/cobra"
)

// NewDoctorCmd returns the doctor subcommand.
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites and environment health",
		Long: `Verify Homebrew is available, config is readable, package names are valid
lowercase Homebrew tokens, the state directory is writable, and (when enabled)
iCloud / git sync status.`,
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	configPath, verbose, _, err := planDisplayOptions(cmd)
	if err != nil {
		return err
	}
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return err
	}

	var manifest config.Manifest
	if _, err := os.Stat(configPath); err == nil {
		manifest, err = config.LoadManifest(configPath)
		if err != nil {
			// still run other checks; config check will fail via load attempt below
			manifest = config.Manifest{}
			_ = err
		}
	}

	brewOK := true
	brewErr := ""
	if _, err := exec.LookPath("brew"); err != nil {
		brewOK = false
		brewErr = "brew not found on PATH"
	} else if out, err := exec.Command("brew", "--version").CombinedOutput(); err != nil {
		brewOK = false
		brewErr = fmt.Sprintf("brew --version failed: %v (%s)", err, stringsTrim(out))
	}

	pouroverOK := true
	pouroverErr := ""
	if _, err := exec.LookPath("pourover"); err != nil {
		pouroverOK = false
		pouroverErr = "pourover not found on PATH"
	}

	report, err := engine.Doctor(engine.DoctorInputs{
		ConfigPath:  configPath,
		StateDir:    stateDir,
		Manifest:    manifest,
		BrewOK:      brewOK,
		BrewErr:     brewErr,
		PouroverOK:  pouroverOK,
		PouroverErr: pouroverErr,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fancy := ui.Enabled(out, false)
	for _, c := range report.Checks {
		status := "ok"
		if !c.OK {
			status = "FAIL"
		}
		line := fmt.Sprintf("[%s] %s: %s", status, c.Name, c.Detail)
		if fancy {
			if c.OK {
				line = ui.Success().Render(line)
			} else {
				line = ui.Fail().Render(line)
			}
		}
		fmt.Fprintln(out, line)
	}
	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "config=%s state=%s\n", configPath, stateDir)
	}
	if !report.OK() {
		return fmt.Errorf("doctor found problems")
	}
	done := "All checks passed."
	if fancy {
		done = ui.Success().Render(done)
	}
	fmt.Fprintln(out, done)
	return nil
}

func stringsTrim(b []byte) string {
	s := string(b)
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}
