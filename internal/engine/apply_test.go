package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	if result.Taps != 0 || result.Formulae != 0 || result.Casks != 0 || result.Mas != 0 || result.PAM != 0 ||
		result.Removed != 0 || result.Defaults != 0 || result.Linked != 0 ||
		result.Managed != 0 || result.Templates != 0 || result.Unlinked != 0 || result.Pruned != 0 ||
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

func TestApply_MasInstall(t *testing.T) {
	mas := &recordingApplyMasRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionMasInstall, Name: "Xcode", Value: "497799835"},
	}}

	result, err := Apply(context.Background(), &stubBrewRunner{}, p, ApplyOptions{
		Mode:      config.UninstallModeSafe,
		AutoYes:   true,
		Quiet:     true,
		MasRunner: mas,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Mas != 1 {
		t.Fatalf("Mas = %d, want 1", result.Mas)
	}
	if len(mas.calls) != 1 || mas.calls[0] != "install 497799835" {
		t.Fatalf("mas calls = %v", mas.calls)
	}
}

func TestApply_PruneConfirmIsMultiline(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.DS_Store")
	b := filepath.Join(dir, "b.DS_Store")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	parked := 0
	conf := &promptConfirmer{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFilePrune, Name: a},
		{Type: plan.ActionFilePrune, Name: b},
	}}
	_, err := Apply(context.Background(), &stubBrewRunner{}, p, ApplyOptions{
		FilesMode:    config.FilesModeSafe,
		Quiet:        true,
		Confirm:      conf,
		BeforePrompt: func() { parked++ },
	})
	if err != nil {
		t.Fatal(err)
	}
	if parked != 1 {
		t.Fatalf("BeforePrompt calls=%d, want 1", parked)
	}
	if strings.Contains(conf.prompt, a+",") {
		t.Fatalf("comma-joined prune prompt: %q", conf.prompt)
	}
	if !strings.Contains(conf.prompt, a) || !strings.Contains(conf.prompt, "\n  ") {
		t.Fatalf("want listed paths: %q", conf.prompt)
	}
	if !strings.Contains(conf.prompt, "Proceed?") {
		t.Fatalf("missing Proceed: %q", conf.prompt)
	}
	if _, err := os.Stat(a); err != nil {
		t.Fatalf("declined prune must leave file: %v", err)
	}
}

type promptConfirmer struct{ prompt string }

func (c *promptConfirmer) Confirm(prompt string) bool {
	c.prompt = prompt
	return false
}

type recordingApplyMasRunner struct {
	calls []string
}

func (r *recordingApplyMasRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, strings.Join(args, " "))
	return nil, nil
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
