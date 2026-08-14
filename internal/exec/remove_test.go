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
}

func (r *removeRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "uninstall" {
		r.uninstalls = append(r.uninstalls, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "uninstall" && args[1] == "--cask" {
		r.uninstalls = append(r.uninstalls, "cask:"+args[2])
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

func TestApplyRemoves_ByMode(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionFormulaRemove, Name: "wget"},
		{Type: plan.ActionCaskRemove, Name: "vlc"},
	}}

	t.Run("non_destructive_skips", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		var prompted bool
		confirm := func(names []string) bool {
			prompted = true
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, p, config.UninstallModeNonDestructive, confirm)
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
		n, err := ApplyRemoves(context.Background(), runner, p, config.UninstallModeStrict, confirm)
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
	})

	t.Run("safe_prompt_yes", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		var gotNames []string
		confirm := func(names []string) bool {
			gotNames = append([]string(nil), names...)
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, p, config.UninstallModeSafe, confirm)
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Fatalf("n=%d, want 2", n)
		}
		if got := strings.Join(gotNames, ","); got != "wget,vlc" {
			t.Fatalf("prompt names = %q, want wget,vlc", got)
		}
		if got := strings.Join(runner.uninstalls, ","); got != "wget,cask:vlc" {
			t.Fatalf("uninstalls = %q", got)
		}
	})

	t.Run("safe_prompt_no", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		confirm := func(names []string) bool { return false }
		n, err := ApplyRemoves(context.Background(), runner, p, config.UninstallModeSafe, confirm)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || len(runner.uninstalls) != 0 {
			t.Fatalf("n=%d uninstalls=%v, want none after decline", n, runner.uninstalls)
		}
	})

	t.Run("safe_no_removes_no_prompt", func(t *testing.T) {
		runner := &removeRecordingRunner{}
		onlyInstall := plan.Plan{Actions: []plan.Action{{Type: plan.ActionFormulaInstall, Name: "fzf"}}}
		var prompted bool
		confirm := func(names []string) bool {
			prompted = true
			return true
		}
		n, err := ApplyRemoves(context.Background(), runner, onlyInstall, config.UninstallModeSafe, confirm)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 || prompted {
			t.Fatalf("n=%d prompted=%v, want no work", n, prompted)
		}
	})
}
