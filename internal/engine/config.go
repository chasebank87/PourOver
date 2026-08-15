package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/backup"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
)

// ConfigStatus summarizes iCloud and git sync settings for TUI/CLI display.
type ConfigStatus struct {
	ICloudEnabled   bool
	ICloudPath      string
	ICloudAvailable bool

	GitEnabled  bool
	GitRemote   string
	GitBranch   string
	GitRepo     bool
	GitDirty    bool
	GitSetupTip string
}

// PushConfigResult describes a config git push attempt.
type PushConfigResult struct {
	Pushed bool
	Remote string
}

const gitSetupTip = `use pourover config git setup <url> for first-time setup`

// LoadConfigStatus reads pourover.lua plus config-dir git state.
func LoadConfigStatus(configPath string) (ConfigStatus, error) {
	if err := requireConfigFile(configPath); err != nil {
		return ConfigStatus{}, err
	}
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return ConfigStatus{}, err
	}
	return statusFromManifest(configPath, manifest)
}

func statusFromManifest(configPath string, manifest config.Manifest) (ConfigStatus, error) {
	st := ConfigStatus{
		ICloudEnabled: manifest.Backup.ICloud.Enabled,
		GitEnabled:    manifest.Backup.Git.Enabled,
		GitRemote:     manifest.Backup.Git.Remote,
		GitBranch:     manifest.Backup.Git.Branch,
	}
	if st.GitBranch == "" {
		st.GitBranch = "main"
	}

	if st.ICloudEnabled {
		path, available, err := backup.ResolveICloudDir(manifest)
		if err != nil {
			return ConfigStatus{}, err
		}
		st.ICloudPath = path
		st.ICloudAvailable = available
		if path == "" {
			st.ICloudPath = strings.TrimSpace(manifest.Backup.ICloud.Path)
		}
	} else if p := strings.TrimSpace(manifest.Backup.ICloud.Path); p != "" {
		st.ICloudPath = p
	}

	cfgDir := filepath.Dir(configPath)
	st.GitRepo = configgit.IsRepo(cfgDir)
	if st.GitRepo {
		if remote, err := configgit.RemoteURL(cfgDir); err == nil && remote != "" {
			st.GitRemote = remote
		}
		if dirty, err := configgit.StatusDirty(cfgDir); err == nil {
			st.GitDirty = dirty
		}
	} else {
		st.GitSetupTip = gitSetupTip
	}
	return st, nil
}

// EnableICloud patches pourover.lua and returns updated status.
func EnableICloud(configPath, icloudPath string, setPath bool) (ConfigStatus, error) {
	if err := requireConfigFile(configPath); err != nil {
		return ConfigStatus{}, err
	}
	if err := config.PatchICloudFile(configPath, true, icloudPath, setPath); err != nil {
		return ConfigStatus{}, err
	}
	return LoadConfigStatus(configPath)
}

// DisableICloud patches pourover.lua to disable iCloud mirroring.
func DisableICloud(configPath string) error {
	if err := requireConfigFile(configPath); err != nil {
		return err
	}
	return config.PatchICloudFile(configPath, false, "", false)
}

// PushConfig commits dirty changes when needed and pushes to the configured remote.
func PushConfig(configPath string) (PushConfigResult, error) {
	if err := requireConfigFile(configPath); err != nil {
		return PushConfigResult{}, err
	}
	cfgDir := filepath.Dir(configPath)
	if !configgit.IsRepo(cfgDir) {
		return PushConfigResult{}, fmt.Errorf("%s is not a git repo (run `pourover config git setup <url>`)", cfgDir)
	}
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return PushConfigResult{}, err
	}
	branch := manifest.Backup.Git.Branch
	if branch == "" {
		branch = "main"
	}
	pushed, err := configgit.CommitAndPushIfDirty(cfgDir, branch, time.Now())
	if err != nil {
		return PushConfigResult{}, err
	}
	remote, _ := configgit.RemoteURL(cfgDir)
	if remote == "" {
		remote = cfgDir
	}
	return PushConfigResult{Pushed: pushed, Remote: remote}, nil
}

// PullConfig pulls config-dir updates from the git remote (ff-only).
func PullConfig(configPath string) error {
	if err := requireConfigFile(configPath); err != nil {
		return err
	}
	cfgDir := filepath.Dir(configPath)
	if !configgit.IsRepo(cfgDir) {
		return fmt.Errorf("%s is not a git repo (run `pourover config git setup <url>`)", cfgDir)
	}
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return err
	}
	branch := manifest.Backup.Git.Branch
	if branch == "" {
		branch = "main"
	}
	return configgit.Pull(cfgDir, branch)
}

func requireConfigFile(configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s (run `pourover init` first)", configPath)
		}
		return err
	}
	return nil
}
