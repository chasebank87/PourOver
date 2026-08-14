package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

type installRecordingRunner struct {
	listFormula []byte
	listCask    []byte
	installs    []string
}

func (r *installRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "list" && args[1] == "--formula" {
		return r.listFormula, nil
	}
	if len(args) == 2 && args[0] == "list" && args[1] == "--cask" {
		return r.listCask, nil
	}
	if len(args) == 2 && args[0] == "install" {
		r.installs = append(r.installs, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "install" && args[1] == "--cask" {
		r.installs = append(r.installs, "cask:"+args[2])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestInstallFormula_RunsBrewInstall(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := InstallFormula(context.Background(), runner, "fzf"); err != nil {
		t.Fatalf("InstallFormula: %v", err)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "fzf" {
		t.Fatalf("installs = %v, want [fzf]", runner.installs)
	}
}

func TestApplyFormulaInstalls_OnlyFormulae(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
		{Type: plan.ActionFormulaInstall, Name: "git"},
	}}

	n, err := ApplyFormulaInstalls(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyFormulaInstalls: %v", err)
	}
	if n != 2 {
		t.Fatalf("installed count = %d, want 2", n)
	}
	if got := strings.Join(runner.installs, ","); got != "fzf,git" {
		t.Fatalf("install order = %q, want fzf,git", got)
	}
}

func TestApplyFormulaInstalls_NoFormulae(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskInstall, Name: "raycast"},
	}}

	n, err := ApplyFormulaInstalls(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyFormulaInstalls: %v", err)
	}
	if n != 0 {
		t.Fatalf("installed count = %d, want 0", n)
	}
	if len(runner.installs) != 0 {
		t.Fatalf("unexpected installs: %v", runner.installs)
	}
}

func TestInstallCask_RunsBrewInstallCask(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := InstallCask(context.Background(), runner, "raycast"); err != nil {
		t.Fatalf("InstallCask: %v", err)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "cask:raycast" {
		t.Fatalf("installs = %v, want [cask:raycast]", runner.installs)
	}
}

func TestApplyCaskInstalls_OnlyCasks(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
		{Type: plan.ActionCaskInstall, Name: "iterm2"},
	}}

	n, err := ApplyCaskInstalls(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyCaskInstalls: %v", err)
	}
	if n != 2 {
		t.Fatalf("installed count = %d, want 2", n)
	}
	if got := strings.Join(runner.installs, ","); got != "cask:raycast,cask:iterm2" {
		t.Fatalf("install order = %q, want cask:raycast,cask:iterm2", got)
	}
}

func TestApplyFormulaInstalls_ContinuesAfterFailure(t *testing.T) {
	runner := &selectiveFailInstallRunner{failNames: map[string]bool{"neofetch": true}}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "neofetch"},
		{Type: plan.ActionFormulaInstall, Name: "onefetch"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
	}}

	var progress []string
	n, err := ApplyFormulaInstalls(context.Background(), runner, p, func(line string) {
		progress = append(progress, line)
	})
	if err == nil {
		t.Fatal("expected error from neofetch")
	}
	if !strings.Contains(err.Error(), "neofetch") {
		t.Fatalf("error = %v, want neofetch", err)
	}
	if n != 1 {
		t.Fatalf("installed count = %d, want 1 (onefetch)", n)
	}
	if got := strings.Join(runner.installs, ","); got != "onefetch" {
		t.Fatalf("installs = %q, want onefetch (neofetch failed, raycast is cask)", got)
	}
	foundFail := false
	for _, line := range progress {
		if strings.Contains(line, "failed:") && strings.Contains(line, "neofetch") {
			foundFail = true
		}
	}
	if !foundFail {
		t.Fatalf("progress missing failure line: %v", progress)
	}
}

func TestApplyCaskInstalls_ContinuesAfterFailure(t *testing.T) {
	runner := &selectiveFailInstallRunner{failNames: map[string]bool{"cask:bad": true}}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskInstall, Name: "bad"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
	}}

	n, err := ApplyCaskInstalls(context.Background(), runner, p, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if n != 1 {
		t.Fatalf("installed count = %d, want 1", n)
	}
	if got := strings.Join(runner.installs, ","); got != "cask:raycast" {
		t.Fatalf("installs = %q, want cask:raycast", got)
	}
}

type selectiveFailInstallRunner struct {
	installs  []string
	failNames map[string]bool
}

func (r *selectiveFailInstallRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "install" {
		if r.failNames[args[1]] {
			return nil, fmt.Errorf("no available formula")
		}
		r.installs = append(r.installs, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "install" && args[1] == "--cask" {
		key := "cask:" + args[2]
		if r.failNames[key] {
			return nil, fmt.Errorf("no available cask")
		}
		r.installs = append(r.installs, key)
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestUnsupportedApplyActions(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
		{Type: plan.ActionFormulaRemove, Name: "wget"},
		{Type: plan.ActionCaskRemove, Name: "vlc"},
		{Type: plan.ActionLinkCreate, Name: "/tmp/x", Source: "config/x"},
		{Type: plan.ActionLinkUpdate, Name: "/tmp/y", Source: "config/y"},
	}}
	skipped := UnsupportedApplyActions(p)
	if len(skipped) != 0 {
		t.Fatalf("skipped = %+v, want none", skipped)
	}
}
