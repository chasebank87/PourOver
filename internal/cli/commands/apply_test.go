package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/plan"
	"github.com/spf13/cobra"
)

type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if isBrewListArgs(args, "--formula") {
		return []byte("git\n"), nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return []byte("git\n"), nil
	}
	if isBrewListArgs(args, "--cask") {
		return []byte(""), nil
	}
	if len(args) >= 2 && args[0] == "deps" && args[1] == "--union" {
		return []byte(""), nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func mutationBrewArgs(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "install", "uninstall", "remove", "reinstall", "upgrade", "untap", "update":
		return true
	case "tap":
		// `brew tap` (list) is discovery; `brew tap name` mutates.
		return len(args) > 1
	case "trust":
		// `brew trust --json=v1` is discovery; `brew trust --tap name` mutates.
		return !(len(args) == 2 && args[1] == "--json=v1")
	default:
		return false
	}
}

func TestApplyDryRun_NoBrewMutations(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" } },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &recordingRunner{}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}
	if err := printPlan(p, false); err != nil {
		t.Fatalf("printPlan: %v", err)
	}

	for _, args := range runner.calls {
		if mutationBrewArgs(args) {
			t.Fatalf("brew mutation during dry-run path: %v", args)
		}
	}
	if len(runner.calls) != 6 {
		t.Fatalf("brew calls = %d, want 6 discovery calls (tap + trust + lists + deps --union)", len(runner.calls))
	}
}

func TestApplyDryRun_MatchesPlanOutput(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "tgt")
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = { "raycast" } },
  files = {
    links = { { source = "config/nvim", target = "`+target+`" } },
  },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &stubBrewRunner{formulae: "git\n", casks: ""}
	planOut, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan plan path: %v", err)
	}
	applyOut, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan apply dry-run path: %v", err)
	}

	planText := plan.RenderText(planOut)
	applyText := plan.RenderText(applyOut)
	if planText != applyText {
		t.Fatalf("plan and apply --dry-run output differ:\nplan:\n%s\napply:\n%s", planText, applyText)
	}
}

func TestExecuteApply_FormulaInstallOnly(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(filepath.Join(configDir, "config", "nvim"), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{
		listFormula: []byte("git\n"),
		listCask:    []byte(""),
	}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts := applyOptions{mode: config.UninstallModeSafe, autoYes: true}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "fzf" {
		t.Fatalf("installs = %v, want [fzf]", runner.installs)
	}

	// Second plan: fzf should no longer need install.
	runner.listFormula = []byte("git\nfzf\n")
	p2, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan after install: %v", err)
	}
	if names := plan.ActionNames(p2, plan.ActionFormulaInstall); len(names) != 0 {
		t.Fatalf("formula installs after apply = %v, want none", names)
	}
}

func (r *installRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if len(args) == 2 && args[0] == "tap" {
		r.installs = append(r.installs, "tap:"+args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "trust" && args[1] == "--tap" {
		r.installs = append(r.installs, "trust:"+args[2])
		return nil, nil
	}
	if isBrewListArgs(args, "--formula") {
		return r.listFormula, nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		if r.listFormulaRequested != nil {
			return r.listFormulaRequested, nil
		}
		return r.listFormula, nil
	}
	if isBrewListArgs(args, "--cask") {
		return r.listCask, nil
	}
	if len(args) >= 2 && args[0] == "deps" && args[1] == "--union" {
		return []byte(""), nil
	}
	if len(args) == 2 && args[0] == "install" {
		r.installs = append(r.installs, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "install" && args[1] == "--cask" {
		r.installs = append(r.installs, "cask:"+args[2])
		return nil, nil
	}
	if len(args) == 2 && args[0] == "uninstall" {
		r.uninstalls = append(r.uninstalls, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "uninstall" && args[1] == "--cask" {
		r.uninstalls = append(r.uninstalls, "cask:"+args[2])
		return nil, nil
	}
	if len(args) == 2 && args[0] == "untap" {
		r.uninstalls = append(r.uninstalls, "untap:"+args[1])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

type installRecordingRunner struct {
	listFormula          []byte
	listFormulaRequested []byte
	listCask             []byte
	installs             []string
	uninstalls           []string
}

func TestExecuteApply_StrictRemoves(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "strict" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{
		listFormula: []byte("git\nwget\n"),
		listCask:    []byte("vlc\n"),
	}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(strings.NewReader(""))
	opts := applyOptions{mode: config.UninstallModeStrict, autoYes: false}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if got := strings.Join(runner.uninstalls, ","); got != "wget,cask:vlc" {
		t.Fatalf("uninstalls = %q, want wget,cask:vlc", got)
	}
}

func TestExecuteApply_SafeRemovesWithYes(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaRemove, Name: "wget"},
	}}
	runner := &installRecordingRunner{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts := applyOptions{mode: config.UninstallModeSafe, autoYes: true}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if len(runner.uninstalls) != 1 || runner.uninstalls[0] != "wget" {
		t.Fatalf("uninstalls = %v, want [wget]", runner.uninstalls)
	}
}

func TestExecuteApply_NonDestructiveSkipsRemoves(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaRemove, Name: "wget"},
	}}
	runner := &installRecordingRunner{}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts := applyOptions{mode: config.UninstallModeNonDestructive, autoYes: true}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if len(runner.uninstalls) != 0 {
		t.Fatalf("uninstalls = %v, want none", runner.uninstalls)
	}
}

func TestNewApplyCmd_HasYesFlag(t *testing.T) {
	cmd := NewApplyCmd()
	if cmd.Flags().Lookup("yes") == nil {
		t.Fatal("missing --yes flag")
	}
	if cmd.Flags().Lookup("quiet") == nil {
		t.Fatal("missing --quiet flag")
	}
}

func TestExecuteApply_PrintsProgressByDefault(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{listFormula: []byte("git\n"), listCask: []byte("")}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetErr(&stderr)
	opts := applyOptions{mode: config.UninstallModeSafe, autoYes: true, quiet: false}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "☕ install formula fzf") {
		t.Fatalf("stderr = %q, want progress line", stderr.String())
	}
}

func TestExecuteApply_QuietSuppressesProgress(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{listFormula: []byte("git\n"), listCask: []byte("")}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetErr(&stderr)
	opts := applyOptions{mode: config.UninstallModeSafe, autoYes: true, quiet: true}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr.String(), "☕") {
		t.Fatalf("quiet stderr still has progress: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Installed 1 formula") {
		t.Fatalf("stderr = %q, want summary", stderr.String())
	}
}

func TestExecuteApply_FormulaAndCaskInstalls(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = { "raycast" } },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{
		listFormula: []byte("git\n"),
		listCask:    []byte(""),
	}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts := applyOptions{mode: config.UninstallModeSafe, autoYes: true}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if got := strings.Join(runner.installs, ","); got != "fzf,cask:raycast" {
		t.Fatalf("installs = %q, want fzf,cask:raycast", got)
	}

	runner.listFormula = []byte("git\nfzf\n")
	runner.listCask = []byte("raycast\n")
	p2, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan after install: %v", err)
	}
	if names := plan.ActionNames(p2, plan.ActionFormulaInstall); len(names) != 0 {
		t.Fatalf("formula installs after apply = %v, want none", names)
	}
	if names := plan.ActionNames(p2, plan.ActionCaskInstall); len(names) != 0 {
		t.Fatalf("cask installs after apply = %v, want none", names)
	}
}

func TestExecuteApply_IdempotentNoChanges(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	src := filepath.Join(configDir, "dot")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "file"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkTgt := filepath.Join(root, "link")
	if err := os.MkdirAll(linkTgt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkTgt, "file"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git" }, casks = {} },
  files = { links = { { source = "dot", target = "`+linkTgt+`" } } },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &installRecordingRunner{
		listFormula: []byte("git\n"),
		listCask:    []byte(""),
	}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}
	if len(p.Actions) != 0 {
		t.Fatalf("expected empty plan for matching state, got %+v", p.Actions)
	}

	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var stderr strings.Builder
	cmd.SetErr(&stderr)

	opts := applyOptions{
		mode:      config.UninstallModeSafe,
		autoYes:   true,
		configDir: configDir,
		stateDir:  stateDir,
		manifest:  manifest,
		now:       func() time.Time { return time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC) },
	}
	if err := executeApply(cmd, runner, p, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}
	if len(runner.installs) != 0 || len(runner.uninstalls) != 0 {
		t.Fatalf("unexpected brew mutations: installs=%v uninstalls=%v", runner.installs, runner.uninstalls)
	}
	if !strings.Contains(stderr.String(), "No changes.") {
		t.Fatalf("stderr = %q, want No changes.", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "lock.json")); err != nil {
		t.Fatalf("lock.json: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "last-plan.json")); err != nil {
		t.Fatalf("last-plan.json: %v", err)
	}
	histDir := filepath.Join(stateDir, "history")
	entries, err := os.ReadDir(histDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("history entries = %v err=%v, want 1", entries, err)
	}
	snaps, err := os.ReadDir(filepath.Join(stateDir, "snapshots"))
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots = %v err=%v, want 1", snaps, err)
	}
}

func TestExecuteApply_FailedApplyStillWritesHistory(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &failingInstallRunner{listFormula: []byte("git\n"), listCask: []byte("")}
	p, err := buildPlan(context.Background(), configPath, runner)
	if err != nil {
		t.Fatalf("buildPlan: %v", err)
	}
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	opts := applyOptions{
		mode:      config.UninstallModeSafe,
		autoYes:   true,
		configDir: configDir,
		stateDir:  stateDir,
		manifest:  manifest,
		now:       func() time.Time { return time.Date(2026, 5, 18, 14, 0, 0, 0, time.UTC) },
	}
	err = executeApply(cmd, runner, p, opts)
	if err == nil {
		t.Fatal("expected apply error")
	}
	if _, statErr := os.Stat(filepath.Join(stateDir, "lock.json")); !os.IsNotExist(statErr) {
		t.Fatalf("lock.json should not exist on failure, stat=%v", statErr)
	}
	entries, readErr := os.ReadDir(filepath.Join(stateDir, "history"))
	if readErr != nil || len(entries) != 1 {
		t.Fatalf("history = %v err=%v, want 1 failure entry", entries, readErr)
	}
}

type failingInstallRunner struct {
	listFormula []byte
	listCask    []byte
}

func (r *failingInstallRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if isBrewListArgs(args, "--formula") {
		return r.listFormula, nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return r.listFormula, nil
	}
	if isBrewListArgs(args, "--cask") {
		return r.listCask, nil
	}
	if len(args) >= 2 && args[0] == "deps" && args[1] == "--union" {
		return []byte(""), nil
	}
	if len(args) >= 1 && args[0] == "install" {
		return nil, fmt.Errorf("brew install failed")
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestExecuteApply_BrewThenFiles(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "cfg")
	src := filepath.Join(configDir, "dot")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "rc"), []byte("export X=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkTgt := filepath.Join(root, "link")
	configPath := filepath.Join(configDir, "pourover.lua")
	if err := os.WriteFile(configPath, []byte(`return {
  packages = { formulae = { "git", "fzf" }, casks = {} },
  files = { links = { { source = "dot", target = "`+linkTgt+`" } } },
  policy = { uninstall_mode = "safe" },
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	runner := &orderRecordingRunner{
		listFormula: []byte("git\n"),
		listCask:    []byte(""),
		linkTarget:  filepath.Join(linkTgt, "rc"),
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	stateDir := filepath.Join(root, "state")
	opts := applyOptions{
		mode:      config.UninstallModeSafe,
		autoYes:   true,
		configDir: configDir,
		stateDir:  stateDir,
	}
	// Rebuild with explicit state so apply can read blobs.
	res, err := engine.BuildPlanResult(context.Background(), configPath, runner, discovery.NewExecDefaultsRunner(), nil, stateDir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	opts.generationID = res.GenerationID
	opts.manifest = res.Manifest
	if err := executeApply(cmd, runner, res.Plan, opts); err != nil {
		t.Fatalf("executeApply: %v", err)
	}

	if got := strings.Join(runner.events, ","); !strings.Contains(got, "brew") {
		t.Fatalf("events = %q, want brew before files", got)
	}
	if _, err := os.Stat(filepath.Join(linkTgt, "rc")); err != nil {
		t.Fatalf("expected activated file after brew phase: %v", err)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "fzf" {
		t.Fatalf("installs = %v, want [fzf]", runner.installs)
	}
}

type orderRecordingRunner struct {
	listFormula []byte
	listCask    []byte
	linkTarget  string
	installs    []string
	events      []string
	t           *testing.T
}

func (r *orderRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 1 && args[0] == "tap" {
		return []byte("homebrew/core\nhomebrew/cask\n"), nil
	}
	if len(args) == 2 && args[0] == "trust" && args[1] == "--json=v1" {
		return []byte(`{"taps":[],"formulae":[],"casks":[],"commands":[]}`), nil
	}
	if isBrewListArgs(args, "--formula") {
		return r.listFormula, nil
	}
	if len(args) == 3 && args[0] == "list" && args[1] == "--formula" && args[2] == "--installed-on-request" {
		return r.listFormula, nil
	}
	if isBrewListArgs(args, "--cask") {
		return r.listCask, nil
	}
	if len(args) >= 2 && args[0] == "deps" && args[1] == "--union" {
		return []byte(""), nil
	}
	if len(args) == 2 && args[0] == "install" {
		if _, err := os.Lstat(r.linkTarget); err == nil {
			return nil, fmt.Errorf("link %s existed before brew install (wrong order)", r.linkTarget)
		}
		r.events = append(r.events, "brew")
		r.installs = append(r.installs, args[1])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestNewApplyCmd_HasDryRunFlag(t *testing.T) {
	cmd := NewApplyCmd()
	flag := cmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("missing --dry-run flag")
	}
	if !strings.Contains(flag.Usage, "plan") {
		t.Errorf("--dry-run usage = %q, want mention of plan", flag.Usage)
	}
}

func TestRunApplyActions_CaskRenamesAreAdvisory(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var stderr strings.Builder
	cmd.SetErr(&stderr)

	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskRename, Name: "windsurf", Value: "devin-desktop"},
		{Type: plan.ActionCaskRename, Name: "vmware-horizon-client", Value: "omnissa-horizon-client"},
	}}
	_, err := runApplyActions(cmd, &recordingRunner{}, p, applyOptions{autoYes: true, quiet: true})
	if err != nil {
		t.Fatalf("runApplyActions: %v", err)
	}
	out := stderr.String()
	if strings.Contains(out, "unsupported") {
		t.Fatalf("renames must not be reported as unsupported: %q", out)
	}
	if !strings.Contains(out, "cask renamed: windsurf → devin-desktop") {
		t.Fatalf("missing windsurf rename: %q", out)
	}
	if !strings.Contains(out, "cask renamed: vmware-horizon-client → omnissa-horizon-client") {
		t.Fatalf("missing vmware rename: %q", out)
	}
	if !strings.Contains(out, "Update packages.lua") {
		t.Fatalf("missing packages.lua tip: %q", out)
	}
	if strings.Contains(out, "0/0") || strings.Contains(out, "starting") {
		t.Fatalf("unexpected progress formatting: %q", out)
	}
}
