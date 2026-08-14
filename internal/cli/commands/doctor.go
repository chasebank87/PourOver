package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
	"github.com/chasebank87/PourOver/internal/paths"
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

type doctorCheck struct {
	Name   string
	OK     bool
	Detail string
}

type doctorReport struct {
	Checks []doctorCheck
}

func (r doctorReport) OK() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

type doctorInputs struct {
	configPath  string
	stateDir    string
	manifest    config.Manifest
	brewOK      bool
	brewErr     string
	pouroverOK  bool
	pouroverErr string
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

	report, err := runDoctorChecks(doctorInputs{
		configPath:  configPath,
		stateDir:    stateDir,
		manifest:    manifest,
		brewOK:      brewOK,
		brewErr:     brewErr,
		pouroverOK:  pouroverOK,
		pouroverErr: pouroverErr,
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	for _, c := range report.Checks {
		status := "ok"
		if !c.OK {
			status = "FAIL"
		}
		fmt.Fprintf(out, "[%s] %s: %s\n", status, c.Name, c.Detail)
	}
	if verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), "config=%s state=%s\n", configPath, stateDir)
	}
	if !report.OK() {
		return fmt.Errorf("doctor found problems")
	}
	fmt.Fprintln(out, "All checks passed.")
	return nil
}

func runDoctorChecks(in doctorInputs) (doctorReport, error) {
	var checks []doctorCheck

	if in.pouroverOK {
		checks = append(checks, doctorCheck{Name: "pourover", OK: true, Detail: "on PATH"})
	} else {
		detail := in.pouroverErr
		if detail == "" {
			detail = "pourover not on PATH"
		}
		checks = append(checks, doctorCheck{Name: "pourover", OK: false, Detail: detail})
	}

	if in.brewOK {
		checks = append(checks, doctorCheck{Name: "brew", OK: true, Detail: "available"})
	} else {
		detail := in.brewErr
		if detail == "" {
			detail = "brew not available"
		}
		checks = append(checks, doctorCheck{Name: "brew", OK: false, Detail: detail})
	}

	if _, err := os.Stat(in.configPath); err != nil {
		if os.IsNotExist(err) {
			checks = append(checks, doctorCheck{
				Name:   "config",
				OK:     false,
				Detail: fmt.Sprintf("not found at %s (run pourover init)", in.configPath),
			})
		} else {
			checks = append(checks, doctorCheck{Name: "config", OK: false, Detail: err.Error()})
		}
	} else if _, err := config.LoadManifest(in.configPath); err != nil {
		checks = append(checks, doctorCheck{Name: "config", OK: false, Detail: err.Error()})
	} else {
		checks = append(checks, doctorCheck{Name: "config", OK: true, Detail: in.configPath})
	}

	checks = append(checks, packagesDoctorCheck(in.configPath)...)

	if err := os.MkdirAll(in.stateDir, 0o755); err != nil {
		checks = append(checks, doctorCheck{Name: "state", OK: false, Detail: err.Error()})
	} else {
		probe := filepath.Join(in.stateDir, ".pourover-doctor-write")
		if err := os.WriteFile(probe, []byte("ok\n"), 0o644); err != nil {
			checks = append(checks, doctorCheck{Name: "state", OK: false, Detail: "not writable: " + err.Error()})
		} else {
			_ = os.Remove(probe)
			checks = append(checks, doctorCheck{Name: "state", OK: true, Detail: in.stateDir})
		}
	}

	if in.manifest.Backup.ICloud.Enabled {
		path, enabled, err := backup.ResolveICloudDir(in.manifest)
		if err != nil {
			checks = append(checks, doctorCheck{Name: "icloud", OK: false, Detail: err.Error()})
		} else if !enabled {
			checks = append(checks, doctorCheck{
				Name:   "icloud",
				OK:     false,
				Detail: "enabled in config but path unavailable (is iCloud Drive signed in?)",
			})
		} else {
			checks = append(checks, doctorCheck{Name: "icloud", OK: true, Detail: path})
		}
	} else {
		checks = append(checks, doctorCheck{Name: "icloud", OK: true, Detail: "disabled"})
	}

	checks = append(checks, gitDoctorCheck(in.configPath, in.manifest)...)

	return doctorReport{Checks: checks}, nil
}

func packagesDoctorCheck(configPath string) []doctorCheck {
	if _, err := os.Stat(configPath); err != nil {
		return nil // config check already covers missing file
	}
	m, err := config.LoadManifest(configPath)
	if err != nil {
		if isPackageNameHealthError(err) {
			return []doctorCheck{{Name: "packages", OK: false, Detail: err.Error()}}
		}
		return []doctorCheck{{Name: "packages", OK: true, Detail: "skipped (config invalid)"}}
	}
	n := len(m.Packages.Formulae) + len(m.Packages.Casks)
	return []doctorCheck{{
		Name:   "packages",
		OK:     true,
		Detail: fmt.Sprintf("%d declared package(s); Homebrew tokens are lowercase", n),
	}}
}

func isPackageNameHealthError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "lowercase Homebrew token")
}

func gitDoctorCheck(configPath string, manifest config.Manifest) []doctorCheck {
	if !manifest.Backup.Git.Enabled {
		return []doctorCheck{{Name: "git", OK: true, Detail: "disabled"}}
	}
	cfgDir := filepath.Dir(configPath)
	if !configgit.IsRepo(cfgDir) {
		return []doctorCheck{{
			Name:   "git",
			OK:     false,
			Detail: fmt.Sprintf("enabled but %s is not a git repo", cfgDir),
		}}
	}
	remote := manifest.Backup.Git.Remote
	if remote == "" {
		if u, err := configgit.RemoteURL(cfgDir); err == nil {
			remote = u
		}
	}
	dirty, err := configgit.StatusDirty(cfgDir)
	if err != nil {
		return []doctorCheck{{Name: "git", OK: false, Detail: err.Error()}}
	}
	state := "clean"
	if dirty {
		state = "dirty"
	}
	detail := fmt.Sprintf("%s (%s)", remote, state)
	if remote == "" {
		detail = "repo local only; no remote configured (" + state + ")"
	}
	return []doctorCheck{{Name: "git", OK: true, Detail: detail}}
}

func stringsTrim(b []byte) string {
	s := string(b)
	if len(s) > 120 {
		return s[:120] + "..."
	}
	return s
}
