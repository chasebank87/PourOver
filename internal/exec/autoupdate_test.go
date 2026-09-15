package exec

import (
	"context"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

func TestApplyAutoUpdate_StartAndDelete(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionBrewAutoupdateDelete, Name: "com.github.domt4.homebrew-autoupdate"},
		{Type: plan.ActionBrewAutoupdateStart, Name: "12h", Value: "--upgrade --cleanup"},
	}}
	n, err := ApplyAutoUpdate(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	got := strings.Join(runner.installs, ",")
	want := "autoupdate:delete,autoupdate:start:12h:--upgrade:--cleanup"
	if got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
}

func TestApplyAutoUpdate_IgnoresOtherActions(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "git"},
		{Type: plan.ActionBrewAutoupdateStart, Name: "24h"},
	}}
	n, err := ApplyAutoUpdate(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n = %d", n)
	}
	if strings.Join(runner.installs, ",") != "autoupdate:start:24h" {
		t.Fatalf("calls = %v", runner.installs)
	}
}
