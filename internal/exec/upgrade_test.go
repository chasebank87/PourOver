package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

type upgradeRecordingRunner struct {
	upgrades []string
}

func (r *upgradeRecordingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "upgrade" {
		r.upgrades = append(r.upgrades, args[1])
		return nil, nil
	}
	if len(args) == 3 && args[0] == "upgrade" && args[1] == "--cask" {
		r.upgrades = append(r.upgrades, "cask:"+args[2])
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
}

func TestApplyUpgrades_FormulaAndCask(t *testing.T) {
	runner := &upgradeRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaUpgrade, Name: "git"},
		{Type: plan.ActionCaskUpgrade, Name: "raycast"},
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
	}}
	n, err := ApplyUpgrades(context.Background(), runner, nil, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}
	if got := strings.Join(runner.upgrades, ","); got != "git,cask:raycast" {
		t.Fatalf("upgrades = %q", got)
	}
}

func TestUpgradeMas_RunsMasUpgrade(t *testing.T) {
	runner := &recordingMasRunner{}
	if err := UpgradeMas(context.Background(), runner, "497799835"); err != nil {
		t.Fatalf("UpgradeMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "upgrade 497799835" {
		t.Fatalf("calls = %q, want upgrade 497799835", got)
	}
}

func TestApplyUpgrades_MasUpgrade(t *testing.T) {
	brew := &upgradeRecordingRunner{}
	mas := &recordingMasRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaUpgrade, Name: "git"},
		{Type: plan.ActionMasUpgrade, Name: "Xcode", Value: "497799835"},
		{Type: plan.ActionCaskUpgrade, Name: "raycast"},
		{Type: plan.ActionMasInstall, Name: "Keynote", Value: "409183694"},
	}}
	n, err := ApplyUpgrades(context.Background(), brew, mas, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("n=%d, want 3", n)
	}
	if got := strings.Join(brew.upgrades, ","); got != "git,cask:raycast" {
		t.Fatalf("brew upgrades = %q", got)
	}
	if got := strings.Join(mas.calls, ","); got != "upgrade 497799835" {
		t.Fatalf("mas calls = %q", got)
	}
}

func TestApplyUpgrades_MasContinuesAfterFailure(t *testing.T) {
	brew := &upgradeRecordingRunner{}
	mas := &recordingMasRunner{failUpgrade: map[string]bool{"1": true}}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionMasUpgrade, Name: "Bad", Value: "1"},
		{Type: plan.ActionMasUpgrade, Name: "Good", Value: "2"},
	}}
	n, err := ApplyUpgrades(context.Background(), brew, mas, p, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if n != 1 {
		t.Fatalf("n=%d, want 1", n)
	}
	if got := strings.Join(mas.calls, ","); got != "upgrade 1,upgrade 2" {
		t.Fatalf("mas calls = %q", got)
	}
}
