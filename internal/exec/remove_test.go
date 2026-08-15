package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
)

type removeRecordingRunner struct {
	uninstalls []string
	calls      []string
}

func (r *removeRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, strings.Join(args, " "))
	if len(args) >= 1 && args[0] == "untap" {
		for _, name := range args[1:] {
			r.uninstalls = append(r.uninstalls, "untap:"+name)
		}
		return nil, nil
	}
	if len(args) >= 1 && args[0] == "uninstall" {
		if len(args) >= 2 && args[1] == "--cask" {
			for _, name := range args[2:] {
				r.uninstalls = append(r.uninstalls, "cask:"+name)
			}
			return nil, nil
		}
		for _, name := range args[1:] {
			r.uninstalls = append(r.uninstalls, name)
		}
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestRemoveFormula_RunsBrewUninstall(t *testing.T) {
	runner := &removeRecordingRunner{}
	if err := RemoveFormula(context.Background(), runner, "wget"); err != nil {
		t.Fatalf("RemoveFormula: %v", err)
	}
	if len(runner.uninstalls) != 1 || runner.uninstalls[0] != "wget" {
		t.Fatalf("uninstalls = %v, want [wget]", runner.uninstalls)
	}
}

func TestRemoveCask_RunsBrewUninstallCask(t *testing.T) {
	runner := &removeRecordingRunner{}
	if err := RemoveCask(context.Background(), runner, "vlc"); err != nil {
		t.Fatalf("RemoveCask: %v", err)
	}
	if len(runner.uninstalls) != 1 || runner.uninstalls[0] != "cask:vlc" {
		t.Fatalf("uninstalls = %v, want [cask:vlc]", runner.uninstalls)
	}
}

func TestRemoveCasks_BatchesOneBrewInvocation(t *testing.T) {
	runner := &removeRecordingRunner{}
	if err := RemoveCasks(context.Background(), runner, []string{"vlc", "warp"}); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || runner.calls[0] != "uninstall --cask vlc warp" {
		t.Fatalf("calls = %v", runner.calls)
	}
	if got := strings.Join(runner.uninstalls, ","); got != "cask:vlc,cask:warp" {
		t.Fatalf("uninstalls = %q", got)
	}
}

func TestApplyRemoves_ByMode(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionFormulaRemove, Name: "wget"},
		{Type: plan.ActionCaskRemove, Name: "vlc"},
	}}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error { return nil }
	t.Cleanup(func() { ensureSudo = orig })

	t.Run("non_destructive_skips", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		var prompted bool
		confirm := func(names []string) bool {
			prompted = true
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeNonDestructive, confirm, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || len(runner.uninstalls) != 0 {
			t.Fatalf("n=%d uninstalls=%v, want none", n, runner.uninstalls)
		}
		if prompted {
			t.Fatal("should not prompt in non_destructive")
		}
	})

	t.Run("strict_no_prompt", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		var prompted bool
		confirm := func(names []string) bool {
			prompted = true
			return false
		}
		n, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeStrict, confirm, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if prompted {
			t.Fatal("strict should not prompt")
		}
		if n != 2 {
			t.Fatalf("n=%d, want 2", n)
		}
		if got := strings.Join(runner.uninstalls, ","); got != "wget,cask:vlc" {
			t.Fatalf("uninstalls = %q, want wget,cask:vlc", got)
		}
		// One brew call per type (formula batch + cask batch).
		if len(runner.calls) != 2 {
			t.Fatalf("calls = %v, want 2 batched uninstalls", runner.calls)
		}
	})

	t.Run("safe_confirm_true", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		confirm := func(names []string) bool {
			if len(names) != 2 {
				t.Fatalf("names = %v", names)
			}
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeSafe, confirm, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("n=%d", n)
		}
		if got := strings.Join(runner.uninstalls, ","); got != "wget,cask:vlc" {
			t.Fatalf("uninstalls = %q", got)
		}
	})

	t.Run("safe_confirm_false", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		confirm := func(names []string) bool { return false }
		n, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeSafe, confirm, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || len(runner.uninstalls) != 0 {
			t.Fatalf("n=%d uninstalls=%v, want none after decline", n, runner.uninstalls)
		}
	})

	t.Run("no_removes_no_prompt", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		onlyInstall := plan.Plan{Actions: []plan.Action{{Type: plan.ActionFormulaInstall, Name: "git"}}}
		confirm := func(names []string) bool {
			t.Fatal("should not prompt")
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, nil, onlyInstall, config.UninstallModeSafe, confirm, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("n=%d", n)
		}
	})
}

func TestApplyRemoves_UntapStrict(t *testing.T) {
	runner := &removeRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTapRemove, Name: "homebrew/cask-fonts"},
		{Type: plan.ActionFormulaRemove, Name: "wget"},
	}}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		t.Fatal("formula/tap-only removes should not call sudo -v")
		return nil
	}
	t.Cleanup(func() { ensureSudo = orig })

	n, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeStrict, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	if got := strings.Join(runner.uninstalls, ","); got != "wget,untap:homebrew/cask-fonts" {
		// formula batch first, then taps
		t.Fatalf("uninstalls = %q", got)
	}
}

func TestApplyRemoves_MasStrict(t *testing.T) {
	brew := &removeRecordingRunner{}
	mas := &recordingMasRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaRemove, Name: "wget"},
		{Type: plan.ActionMasRemove, Name: "Xcode", Value: "497799835"},
		{Type: plan.ActionMasRemove, Name: "Keynote", Value: "310633997"},
	}}
	sudoCalls := 0
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		sudoCalls++
		return nil
	}
	t.Cleanup(func() { ensureSudo = orig })

	n, err := ApplyRemoves(context.Background(), brew, mas, p, config.UninstallModeStrict, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("n=%d, want 3", n)
	}
	if sudoCalls != 1 {
		t.Fatalf("EnsureSudo calls=%d, want 1", sudoCalls)
	}
	if got := strings.Join(brew.uninstalls, ","); got != "wget" {
		t.Fatalf("brew uninstalls = %q, want wget", got)
	}
	if got := strings.Join(mas.calls, ","); got != "uninstall 497799835 310633997" {
		t.Fatalf("mas calls = %q", got)
	}
}

func TestApplyRemoves_MasSafePromptNamesIncludeID(t *testing.T) {
	brew := &removeRecordingRunner{}
	mas := &recordingMasRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionMasRemove, Name: "Xcode", Value: "497799835"},
	}}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error { return nil }
	t.Cleanup(func() { ensureSudo = orig })

	var gotNames []string
	confirm := func(names []string) bool {
		gotNames = append([]string{}, names...)
		return true
	}
	n, err := ApplyRemoves(context.Background(), brew, mas, p, config.UninstallModeSafe, confirm, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	if len(gotNames) != 1 || gotNames[0] != "Xcode (497799835)" {
		t.Fatalf("names = %v", gotNames)
	}
}

func TestApplyRemoves_EnsureSudoBeforeAuth(t *testing.T) {
	runner := &removeRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskRemove, Name: "vlc"},
	}}
	auth := 0
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		if beforeAuth != nil {
			beforeAuth()
		}
		return nil
	}
	t.Cleanup(func() { ensureSudo = orig })

	_, err := ApplyRemoves(context.Background(), runner, nil, p, config.UninstallModeStrict, nil, func() { auth++ }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if auth != 1 {
		t.Fatalf("beforeAuth calls=%d, want 1", auth)
	}
}
