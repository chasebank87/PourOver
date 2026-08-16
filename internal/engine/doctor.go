package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/pam"
)

// DoctorCheck is one health check result.
type DoctorCheck struct {
	Name   string
	OK     bool
	Warn   bool // advisory; does not fail DoctorReport.OK()
	Detail string
}

// DoctorReport aggregates doctor checks.
type DoctorReport struct {
	Checks []DoctorCheck
}

// OK reports whether every non-warn check passed.
func (r DoctorReport) OK() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// DoctorInputs supplies paths and pre-probed PATH / brew status for Doctor.
// Frontends probe brew/pourover on PATH; engine runs filesystem and config checks.
type DoctorInputs struct {
	ConfigPath  string
	StateDir    string
	Manifest    config.Manifest
	BrewOK      bool
	BrewErr     string
	PouroverOK  bool
	PouroverErr string
}

// Doctor runs environment health checks and returns structured results.
func Doctor(in DoctorInputs) (DoctorReport, error) {
	var checks []DoctorCheck

	if in.PouroverOK {
		checks = append(checks, DoctorCheck{Name: "pourover", OK: true, Detail: "on PATH"})
	} else {
		detail := in.PouroverErr
		if detail == "" {
			detail = "pourover not on PATH"
		}
		checks = append(checks, DoctorCheck{Name: "pourover", OK: false, Detail: detail})
	}

	if in.BrewOK {
		checks = append(checks, DoctorCheck{Name: "brew", OK: true, Detail: "available"})
	} else {
		detail := in.BrewErr
		if detail == "" {
			detail = "brew not available"
		}
		checks = append(checks, DoctorCheck{Name: "brew", OK: false, Detail: detail})
	}

	if _, err := os.Stat(in.ConfigPath); err != nil {
		if os.IsNotExist(err) {
			checks = append(checks, DoctorCheck{
				Name:   "config",
				OK:     false,
				Detail: fmt.Sprintf("not found at %s (run pourover init)", in.ConfigPath),
			})
		} else {
			checks = append(checks, DoctorCheck{Name: "config", OK: false, Detail: err.Error()})
		}
	} else if _, err := config.LoadManifest(in.ConfigPath); err != nil {
		checks = append(checks, DoctorCheck{Name: "config", OK: false, Detail: err.Error()})
	} else {
		checks = append(checks, DoctorCheck{Name: "config", OK: true, Detail: in.ConfigPath})
	}

	checks = append(checks, packagesDoctorCheck(in.ConfigPath)...)
	checks = append(checks, masDoctorChecks(in.Manifest)...)
	checks = append(checks, pamWatchIDDoctorCheck(in.Manifest)...)

	if err := os.MkdirAll(in.StateDir, 0o755); err != nil {
		checks = append(checks, DoctorCheck{Name: "state", OK: false, Detail: err.Error()})
	} else {
		probe := filepath.Join(in.StateDir, ".pourover-doctor-write")
		if err := os.WriteFile(probe, []byte("ok\n"), 0o644); err != nil {
			checks = append(checks, DoctorCheck{Name: "state", OK: false, Detail: "not writable: " + err.Error()})
		} else {
			_ = os.Remove(probe)
			checks = append(checks, DoctorCheck{Name: "state", OK: true, Detail: in.StateDir})
		}
	}

	if in.Manifest.Backup.ICloud.Enabled {
		path, enabled, err := backup.ResolveICloudDir(in.Manifest)
		if err != nil {
			checks = append(checks, DoctorCheck{Name: "icloud", OK: false, Detail: err.Error()})
		} else if !enabled {
			checks = append(checks, DoctorCheck{
				Name:   "icloud",
				OK:     false,
				Detail: "enabled in config but path unavailable (is iCloud Drive signed in?)",
			})
		} else {
			checks = append(checks, DoctorCheck{Name: "icloud", OK: true, Detail: path})
		}
	} else {
		checks = append(checks, DoctorCheck{Name: "icloud", OK: true, Detail: "disabled"})
	}

	checks = append(checks, gitDoctorCheck(in.ConfigPath, in.Manifest)...)

	return DoctorReport{Checks: checks}, nil
}

func packagesDoctorCheck(configPath string) []DoctorCheck {
	if _, err := os.Stat(configPath); err != nil {
		return nil // config check already covers missing file
	}
	m, err := config.LoadManifest(configPath)
	if err != nil {
		if isPackageNameHealthError(err) {
			return []DoctorCheck{{Name: "packages", OK: false, Detail: err.Error()}}
		}
		return []DoctorCheck{{Name: "packages", OK: true, Detail: "skipped (config invalid)"}}
	}
	n := len(m.Packages.Formulae) + len(m.Packages.Casks)
	detail := fmt.Sprintf("%d brew package(s); Homebrew tokens are lowercase", n)
	if m.Packages.MasConfigured {
		detail = fmt.Sprintf("%d brew + %d mas app(s); Homebrew tokens are lowercase", n, len(m.Packages.Mas))
	}
	return []DoctorCheck{{
		Name:   "packages",
		OK:     true,
		Detail: detail,
	}}
}

func masDoctorChecks(m config.Manifest) []DoctorCheck {
	if !m.Packages.MasConfigured {
		return nil
	}
	if _, err := exec.LookPath("mas"); err != nil {
		return []DoctorCheck{{
			Name:   "mas",
			OK:     true,
			Warn:   true,
			Detail: "packages.mas configured but mas not on PATH (will install via brew on apply)",
		}}
	}
	ctx := context.Background()
	state, err := discovery.DiscoverMas(ctx, discovery.NewExecMasRunner())
	if err != nil {
		return []DoctorCheck{{
			Name:   "mas",
			OK:     true,
			Warn:   true,
			Detail: "mas list failed: " + err.Error(),
		}}
	}
	listed := make(map[int64]struct{}, len(state.Apps))
	for _, app := range state.Apps {
		listed[app.ID] = struct{}{}
	}
	probe := discovery.DefaultMasDiskProbe
	if probe == nil {
		probe = discovery.HostMasDiskProbe{}
	}
	var lag []string
	for _, want := range m.Packages.Mas {
		if _, ok := listed[want.ID]; ok {
			continue
		}
		exists, adamID := probe.FindApp(want.Name)
		if !exists {
			continue
		}
		if adamID != 0 && adamID != want.ID {
			continue
		}
		lag = append(lag, discovery.FormatMasDiskHint(want.Name, want.ID))
	}
	detail := fmt.Sprintf("mas list reports %d app(s); %d declared", len(state.Apps), len(m.Packages.Mas))
	if len(lag) > 0 {
		return []DoctorCheck{{
			Name:   "mas",
			OK:     true,
			Warn:   true,
			Detail: detail + "; Spotlight lag: " + strings.Join(lag, "; "),
		}}
	}
	return []DoctorCheck{{Name: "mas", OK: true, Detail: detail}}
}

func pamWatchIDDoctorCheck(m config.Manifest) []DoctorCheck {
	sl := m.MacOS.Security.PAM.SudoLocal
	if !sl.Configured || !sl.Enable || !sl.WatchIDAuth {
		return nil
	}
	candidates := pam.DefaultWatchIDSearchPaths("")
	if _, ok := pam.FindModule(candidates); ok {
		return []DoctorCheck{{Name: "pam-watchid", OK: true, Detail: "pam_watchid.so found"}}
	}
	return []DoctorCheck{{
		Name:   "pam-watchid",
		OK:     true,
		Warn:   true,
		Detail: "watch_id_auth enabled but pam_watchid.so not found; install pam-watchid manually (not a Homebrew core formula)",
	}}
}

func isPackageNameHealthError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "lowercase Homebrew token")
}

func gitDoctorCheck(configPath string, manifest config.Manifest) []DoctorCheck {
	if !manifest.Backup.Git.Enabled {
		return []DoctorCheck{{Name: "git", OK: true, Detail: "disabled"}}
	}
	cfgDir := filepath.Dir(configPath)
	if !configgit.IsRepo(cfgDir) {
		return []DoctorCheck{{
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
		return []DoctorCheck{{Name: "git", OK: false, Detail: err.Error()}}
	}
	state := "clean"
	if dirty {
		state = "dirty"
	}
	detail := fmt.Sprintf("%s (%s)", remote, state)
	if remote == "" {
		detail = "repo local only; no remote configured (" + state + ")"
	}
	return []DoctorCheck{{Name: "git", OK: true, Detail: detail}}
}
