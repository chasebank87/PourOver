package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/configgit"
	"github.com/spf13/cobra"
)

func TestConfigICloudEnableDisable(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, ".pourover")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "pourover.lua")
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = { icloud = { enabled = false } },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "pourover"}
	rootCmd.PersistentFlags().String("config", "", "")
	_ = rootCmd.PersistentFlags().Set("config", configPath)
	rootCmd.AddCommand(NewConfigCmd())

	rootCmd.SetArgs([]string{"config", "icloud", "enable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Backup.ICloud.Enabled {
		t.Fatal("expected icloud enabled")
	}

	rootCmd.SetArgs([]string{"config", "icloud", "disable"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	m, err = config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if m.Backup.ICloud.Enabled {
		t.Fatal("expected icloud disabled")
	}
}

func TestConfigGitRestoreRequiresForce(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}

	rootCmd := &cobra.Command{Use: "pourover"}
	rootCmd.PersistentFlags().String("config", "", "")
	_ = rootCmd.PersistentFlags().Set("config", configPath)
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.SetArgs([]string{"config", "git", "restore", "git@github.com:example/x.git"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("err = %v, want not empty", err)
	}
}

func TestMaybeAutoPushConfig_SoftFail(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pourover.lua")
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = {
    git = {
      enabled = true,
      auto_push = true,
      remote = "git@github.com:example/does-not-exist-pourover.git",
      branch = "main",
    },
  },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := configgit.Init(dir); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.email", "test@example.com").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatal(err)
	}
	if err := configgit.EnsureRemote(dir, "git@github.com:example/does-not-exist-pourover.git"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.lua"), []byte("-- x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	m, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}
	maybeAutoPushConfig(cmd, configPath, m)
	if !strings.Contains(stderr.String(), "warning: config git auto-push failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDoctorGitDisabled(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return { policy = { uninstall_mode = "safe" } }`), 0o644); err != nil {
		t.Fatal(err)
	}
	checks := gitDoctorCheck(configPath, mustLoadDoctor(t, configPath))
	if len(checks) != 1 || checks[0].Name != "git" || !checks[0].OK || checks[0].Detail != "disabled" {
		t.Fatalf("%+v", checks)
	}
}
