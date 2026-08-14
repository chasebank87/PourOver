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

	n, err := ApplyFormulaInstalls(context.Background(), runner, p)
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

	n, err := ApplyFormulaInstalls(context.Background(), runner, p)
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

func TestUnsupportedApplyActions(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionLinkCreate, Name: "/tmp/x", Source: "config/x"},
	}}
	skipped := UnsupportedApplyActions(p)
	if len(skipped) != 1 || skipped[0].Type != plan.ActionLinkCreate {
		t.Fatalf("skipped = %+v, want one link_create", skipped)
	}
}
