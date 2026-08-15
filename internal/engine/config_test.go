package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
)

func TestEnableDisableICloud_PatchesManifest(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	icloudDir := filepath.Join(root, "icloud")
	if err := os.MkdirAll(icloudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = {
    icloud = {
      enabled = false,
      path = "` + icloudDir + `",
    },
  },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := EnableICloud(configPath, "", false)
	if err != nil {
		t.Fatalf("EnableICloud: %v", err)
	}
	if !st.ICloudEnabled {
		t.Fatal("expected ICloudEnabled")
	}
	if st.ICloudPath != icloudDir {
		t.Fatalf("ICloudPath = %q, want %q", st.ICloudPath, icloudDir)
	}
	if !st.ICloudAvailable {
		t.Fatal("expected ICloudAvailable")
	}

	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Backup.ICloud.Enabled {
		t.Fatalf("manifest icloud = %+v", m.Backup.ICloud)
	}

	if err := DisableICloud(configPath); err != nil {
		t.Fatalf("DisableICloud: %v", err)
	}
	m, err = config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if m.Backup.ICloud.Enabled {
		t.Fatal("expected icloud disabled")
	}
}

func TestLoadConfigStatus_ICloudAndGit(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	icloudDir := filepath.Join(root, "icloud")
	if err := os.MkdirAll(icloudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = {
    icloud = { enabled = true, path = "` + icloudDir + `" },
    git = {
      enabled = true,
      remote = "git@github.com:example/pourover.git",
      branch = "main",
      auto_push = true,
    },
  },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := configgit.Init(root); err != nil {
		t.Fatal(err)
	}
	gitConfig(t, root)
	if err := configgit.EnsureRemote(root, "git@github.com:example/pourover.git"); err != nil {
		t.Fatal(err)
	}
	// Dirty working tree
	if err := os.WriteFile(filepath.Join(root, "extra.lua"), []byte("-- x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := LoadConfigStatus(configPath)
	if err != nil {
		t.Fatalf("LoadConfigStatus: %v", err)
	}
	if !st.ICloudEnabled || !st.ICloudAvailable || st.ICloudPath != icloudDir {
		t.Fatalf("icloud status = %+v", st)
	}
	if !st.GitEnabled || !st.GitRepo || !st.GitDirty {
		t.Fatalf("git status = %+v", st)
	}
	if st.GitRemote != "git@github.com:example/pourover.git" {
		t.Fatalf("GitRemote = %q", st.GitRemote)
	}
	if st.GitBranch != "main" {
		t.Fatalf("GitBranch = %q", st.GitBranch)
	}
}

func TestLoadConfigStatus_GitSetupTipWhenNotRepo(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = { git = { enabled = false } },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := LoadConfigStatus(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.GitRepo {
		t.Fatal("expected not a repo")
	}
	if !strings.Contains(st.GitSetupTip, "pourover config git setup") {
		t.Fatalf("GitSetupTip = %q, want setup tip", st.GitSetupTip)
	}
}

func TestPushConfig_NothingToPush(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = {
    git = {
      enabled = true,
      remote = "git@github.com:example/x.git",
      branch = "main",
      auto_push = true,
    },
  },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := configgit.Init(root); err != nil {
		t.Fatal(err)
	}
	gitConfig(t, root)
	if err := configgit.AddAll(root); err != nil {
		t.Fatal(err)
	}
	if err := configgit.Commit(root, "initial"); err != nil {
		t.Fatal(err)
	}
	// No remote configured for push — but clean tree should report nothing to push
	// before attempting network. CommitAndPushIfDirty returns false when clean+not ahead.
	result, err := PushConfig(configPath)
	if err != nil {
		t.Fatalf("PushConfig: %v", err)
	}
	if result.Pushed {
		t.Fatal("expected nothing to push")
	}
}

func TestPushConfig_RequiresRepo(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := PushConfig(configPath)
	if err == nil || !strings.Contains(err.Error(), "not a git repo") {
		t.Fatalf("err = %v, want not a git repo", err)
	}
}

func TestPullConfig_RequiresRepo(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PullConfig(configPath)
	if err == nil || !strings.Contains(err.Error(), "not a git repo") {
		t.Fatalf("err = %v, want not a git repo", err)
	}
}

func gitConfig(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}
}
