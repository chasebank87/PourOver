package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestApply_NoChanges(t *testing.T) {
	result, err := Apply(context.Background(), &stubBrewRunner{}, plan.Plan{}, ApplyOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Taps != 0 || result.Formulae != 0 || result.Casks != 0 ||
		result.Removed != 0 || result.Defaults != 0 || result.Linked != 0 ||
		result.Managed != 0 || result.Unlinked != 0 || result.Pruned != 0 ||
		result.Renames != 0 || result.Skipped != 0 || result.Failures != 0 {
		t.Fatalf("Apply empty plan = %+v, want zero counts", result)
	}
}

func TestApply_FormulaInstall(t *testing.T) {
	runner := &recordingApplyRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
	}}

	result, err := Apply(context.Background(), runner, p, ApplyOptions{
		Mode:    config.UninstallModeSafe,
		AutoYes: true,
		Quiet:   true,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Formulae != 1 {
		t.Fatalf("Formulae = %d, want 1", result.Formulae)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "fzf" {
		t.Fatalf("installs = %v, want [fzf]", runner.installs)
	}
}

func TestApply_InstallFailureContinues(t *testing.T) {
	runner := &recordingApplyRunner{failInstall: map[string]bool{"bad": true}}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "bad"},
		{Type: plan.ActionFormulaInstall, Name: "good"},
	}}

	result, err := Apply(context.Background(), runner, p, ApplyOptions{
		Mode:    config.UninstallModeSafe,
		AutoYes: true,
		Quiet:   true,
		// Non-nil Progress so Failures are counted from soft-fail lines.
		Progress: func(string) {},
	})
	if err == nil {
		t.Fatal("Apply: want joined install error")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Fatalf("error = %v, want mention of bad", err)
	}
	if result.Formulae != 1 {
		t.Fatalf("Formulae = %d, want 1 (good still installed)", result.Formulae)
	}
	if result.Failures != 1 {
		t.Fatalf("Failures = %d, want 1", result.Failures)
	}
	if len(runner.installs) != 1 || runner.installs[0] != "good" {
		t.Fatalf("installs = %v, want [good]", runner.installs)
	}
}

type recordingApplyRunner struct {
	installs    []string
	failInstall map[string]bool
}

func (r *recordingApplyRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "install" {
		name := args[1]
		if r.failInstall[name] {
			return nil, fmt.Errorf("brew install failed")
		}
		r.installs = append(r.installs, name)
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}
