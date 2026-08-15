package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/chasebank87/PourOver/internal/state"
)

type stubBrewRunner struct {
	formulae         string
	casks            string
	outdatedFormulae string
	outdatedCasks    string
	outdatedSet      bool
}

func (s *stubBrewRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "--prefix" {
		return []byte("/opt/homebrew\n"), nil
	}
	if len(args) == 2 && args[0] == "--prefix" {
		return []byte("/opt/homebrew/opt/" + args[1] + "\n"), nil
	}
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if isBrewListArgs(args, "--formula") {
		return []byte(s.formulae), nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return []byte(s.formulae), nil
	}
	if isBrewListArgs(args, "--cask") {
		return []byte(s.casks), nil
	}
	if len(args) >= 2 && args[0] == "deps" && args[1] == "--union" {
		return []byte(""), nil
	}
	if len(args) == 3 && args[0] == "outdated" && args[1] == "--formula" && args[2] == "-q" {
		if s.outdatedSet {
			return []byte(s.outdatedFormulae), nil
		}
		return []byte(s.formulae), nil
	}
	if len(args) == 3 && args[0] == "outdated" && args[1] == "--cask" && args[2] == "-q" {
		if s.outdatedSet {
			return []byte(s.outdatedCasks), nil
		}
		return []byte(s.casks), nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func isBrewListArgs(args []string, kind string) bool {
	if len(args) == 2 && args[0] == "list" && args[1] == kind {
		return true
	}
	return len(args) == 3 && args[0] == "list" && args[1] == kind && args[2] == "-1"
}

func TestBuildPlan_FromFixture(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "..", "test", "fixtures", "config", "valid")
	data, err := os.ReadFile(filepath.Join(src, "pourover.lua"))
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Touch link source so file discovery succeeds.
	if err := os.WriteFile(filepath.Join(configDir, "config", "nvim", "init.lua"), []byte("--"), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{formulae: "git\n", casks: "raycast\n"}
	p, err := BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 1 || names[0] != "fzf" {
		t.Fatalf("formula installs = %v", names)
	}
}

func TestBuildPlan_ManagedAndUnlink(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	srcContent := []byte("managed-body\n")
	if err := os.WriteFile(filepath.Join(configDir, "config", "foo.conf"), srcContent, 0o644); err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	managedTarget := filepath.Join(home, ".config", "foo.conf")
	if err := os.MkdirAll(filepath.Dir(managedTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	unlinkPath := filepath.Join(home, ".old-dotfile")
	if err := os.WriteFile(unlinkPath, []byte("remove-me"), 0o644); err != nil {
		t.Fatal(err)
	}

	lua := `return {
  packages = { formulae = {}, casks = {} },
  files = {
    managed = {
      { source = "config/foo.conf", target = "~/.config/foo.conf" },
    },
    unlink = { "~/.old-dotfile" },
  },
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{}
	p, err := BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionManagedCopy); len(names) != 1 || names[0] != "~/.config/foo.conf" {
		t.Fatalf("managed copies = %v", names)
	}
	if names := plan.ActionNames(p, plan.ActionFileUnlink); len(names) != 1 || names[0] != "~/.old-dotfile" {
		t.Fatalf("unlinks = %v", names)
	}
}

func TestBuildPlan_FilePruneFromLock(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	stateDir := filepath.Join(root, "state")
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	extra := filepath.Join(home, ".extra")
	if err := os.WriteFile(extra, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := config.Manifest{
		Policy: config.Policy{UninstallMode: config.UninstallModeSafe, FilesMode: config.FilesModeSafe},
	}
	if err := state.PersistApplyState(stateDir, manifest, plan.Plan{}, time.Now().UTC(), []string{extra}); err != nil {
		t.Fatal(err)
	}

	lua := `return {
  packages = { formulae = {}, casks = {} },
  files = {},
  policy = { uninstall_mode = "safe", files_mode = "safe" },
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{}
	p, err := BuildPlanWith(context.Background(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, stateDir)
	if err != nil {
		t.Fatalf("BuildPlanWith: %v", err)
	}
	names := plan.ActionNames(p, plan.ActionFilePrune)
	if len(names) != 1 || names[0] != extra {
		t.Fatalf("prune = %v, want [%s]", names, extra)
	}

	// non_destructive: no prune even with owned extras
	luaND := `return {
  packages = { formulae = {}, casks = {} },
  files = {},
  policy = { uninstall_mode = "safe", files_mode = "non_destructive" },
}
`
	if err := os.WriteFile(configPath, []byte(luaND), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = BuildPlanWith(context.Background(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, stateDir)
	if err != nil {
		t.Fatalf("BuildPlanWith non_destructive: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFilePrune); len(names) != 0 {
		t.Fatalf("non_destructive prune = %v, want none", names)
	}

	// empty owned lock → no prune
	emptyState := filepath.Join(root, "empty-state")
	if err := state.PersistApplyState(emptyState, manifest, plan.Plan{}, time.Now().UTC(), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = BuildPlanWith(context.Background(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, emptyState)
	if err != nil {
		t.Fatalf("BuildPlanWith empty owned: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFilePrune); len(names) != 0 {
		t.Fatalf("empty owned prune = %v, want none", names)
	}
}

type stubMasRunner struct {
	list string
	err  error
}

func (s *stubMasRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "list" {
		if s.err != nil {
			return nil, s.err
		}
		return []byte(s.list), nil
	}
	return nil, fmt.Errorf("unexpected mas args: %v", args)
}

func TestBuildPlanWith_MasInstallAndImpliedFormula(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lua := `return {
  packages = {
    formulae = {},
    casks = {},
    mas = { Xcode = 497799835 },
  },
  files = {},
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	brew := &stubBrewRunner{}
	mas := &stubMasRunner{list: ""}
	p, err := BuildPlanWith(context.Background(), configPath, brew, discovery.NewExecDefaultsRunner(), mas, "")
	if err != nil {
		t.Fatalf("BuildPlanWith: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 1 || names[0] != "mas" {
		t.Fatalf("formula installs = %v, want [mas]", names)
	}
	masInstalls := plan.ActionNames(p, plan.ActionMasInstall)
	if len(masInstalls) != 1 || masInstalls[0] != "Xcode" {
		t.Fatalf("mas installs = %v, want [Xcode]", masInstalls)
	}
	var xcode plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionMasInstall && a.Name == "Xcode" {
			xcode = a
			break
		}
	}
	if xcode.Value != "497799835" {
		t.Fatalf("Xcode action = %#v, want Value 497799835", xcode)
	}
}

func TestBuildPlanWith_MasSkippedWhenUnconfigured(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lua := `return {
  packages = { formulae = {}, casks = {} },
  files = {},
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	brew := &stubBrewRunner{}
	mas := &stubMasRunner{err: fmt.Errorf("mas should not be called")}
	p, err := BuildPlanWith(context.Background(), configPath, brew, discovery.NewExecDefaultsRunner(), mas, "")
	if err != nil {
		t.Fatalf("BuildPlanWith: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionMasInstall); len(names) != 0 {
		t.Fatalf("mas installs = %v, want none", names)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 0 {
		t.Fatalf("formula installs = %v, want none (mas not implied)", names)
	}
}

func TestBuildPlanWith_MasBinaryMissingContinuesBootstrap(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lua := `return {
  packages = {
    formulae = {},
    casks = {},
    mas = { Xcode = 497799835 },
  },
  files = {},
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	brew := &stubBrewRunner{}
	// Simulate os/exec when mas is not on PATH (bootstrap before brew installs mas).
	mas := &stubMasRunner{err: &exec.Error{Name: "mas", Err: exec.ErrNotFound}}
	p, err := BuildPlanWith(context.Background(), configPath, brew, discovery.NewExecDefaultsRunner(), mas, "")
	if err != nil {
		t.Fatalf("BuildPlanWith: %v (want continue when mas binary missing)", err)
	}
	if names := plan.ActionNames(p, plan.ActionFormulaInstall); len(names) != 1 || names[0] != "mas" {
		t.Fatalf("formula installs = %v, want [mas] bootstrap", names)
	}
	if names := plan.ActionNames(p, plan.ActionMasInstall); len(names) != 1 || names[0] != "Xcode" {
		t.Fatalf("mas installs = %v, want [Xcode]", names)
	}
}

func TestBuildPlanWith_MasDiscoverOtherErrorFails(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lua := `return {
  packages = {
    formulae = {},
    casks = {},
    mas = { Xcode = 497799835 },
  },
  files = {},
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	brew := &stubBrewRunner{}
	mas := &stubMasRunner{err: fmt.Errorf("mas list: exit status 1")}
	_, err := BuildPlanWith(context.Background(), configPath, brew, discovery.NewExecDefaultsRunner(), mas, "")
	if err == nil {
		t.Fatal("BuildPlanWith: want error for non-missing mas failure")
	}
	if !strings.Contains(err.Error(), "discover mas") {
		t.Fatalf("error = %v, want discover mas prefix", err)
	}
}

func TestBuildPlan_Templates(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(configDir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(configDir, "config", "foo.tmpl"), []byte("static\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lua := `return {
  packages = { formulae = {}, casks = {} },
  files = {
    templates = {
      { source = "config/foo.tmpl", target = "~/.foo" },
    },
  },
}
`
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(lua), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{}
	p, err := BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	actions := p.Actions
	var tmplActs []plan.Action
	for _, a := range actions {
		if a.Type == plan.ActionTemplateWrite {
			tmplActs = append(tmplActs, a)
		}
	}
	if len(tmplActs) != 1 || tmplActs[0].Name != "~/.foo" {
		t.Fatalf("template writes = %#v", tmplActs)
	}
	if tmplActs[0].Value == "" || tmplActs[0].Source != "config/foo.tmpl" {
		t.Fatalf("action = %#v", tmplActs[0])
	}

	if err := os.WriteFile(filepath.Join(home, ".foo"), []byte("static\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan same: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionTemplateWrite); len(names) != 0 {
		t.Fatalf("same content should be noop, got %v", names)
	}

	if err := os.WriteFile(filepath.Join(home, ".foo"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = BuildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("BuildPlan differ: %v", err)
	}
	if names := plan.ActionNames(p, plan.ActionTemplateWrite); len(names) != 1 {
		t.Fatalf("differ template writes = %v", names)
	}
}
