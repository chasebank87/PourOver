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
	n, err := ApplyUpgrades(context.Background(), runner, p, nil)
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
