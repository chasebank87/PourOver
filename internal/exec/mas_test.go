package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

type recordingMasRunner struct {
	calls       []string
	failIDs     map[string]bool
	failGet     map[string]bool
	failUnins   map[string]bool
	failUpgrade map[string]bool
}

func (r *recordingMasRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	r.calls = append(r.calls, key)
	if len(args) == 2 && args[0] == "install" && r.failIDs[args[1]] {
		return nil, fmt.Errorf("mas install failed")
	}
	if len(args) == 2 && args[0] == "get" && r.failGet[args[1]] {
		return nil, fmt.Errorf("mas get failed")
	}
	if len(args) == 2 && args[0] == "uninstall" && r.failUnins[args[1]] {
		return nil, fmt.Errorf("mas uninstall failed")
	}
	if len(args) == 2 && args[0] == "upgrade" && r.failUpgrade[args[1]] {
		return nil, fmt.Errorf("mas upgrade failed")
	}
	return nil, nil
}

func TestInstallMas_RunsMasInstall(t *testing.T) {
	runner := &recordingMasRunner{}
	if err := InstallMas(context.Background(), runner, "497799835"); err != nil {
		t.Fatalf("InstallMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "install 497799835" {
		t.Fatalf("calls = %q, want install 497799835", got)
	}
}

func TestInstallMas_SkipsGetWhenInstallSucceeds(t *testing.T) {
	runner := &recordingMasRunner{}
	if err := InstallMas(context.Background(), runner, "1518423503"); err != nil {
		t.Fatalf("InstallMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "install 1518423503" {
		t.Fatalf("calls = %q, want only install", got)
	}
}

func TestRemoveMas_RunsMasUninstall(t *testing.T) {
	runner := &recordingMasRunner{}
	if err := RemoveMas(context.Background(), runner, "310633997"); err != nil {
		t.Fatalf("RemoveMas: %v", err)
	}
	if got := strings.Join(runner.calls, ","); got != "uninstall 310633997" {
		t.Fatalf("calls = %q, want uninstall 310633997", got)
	}
}

func TestApplyMasInstalls_OnlyMas(t *testing.T) {
	runner := &recordingMasRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionMasInstall, Name: "Xcode", Value: "497799835"},
		{Type: plan.ActionCaskInstall, Name: "raycast"},
		{Type: plan.ActionMasInstall, Name: "Keynote", Value: "409183694"},
	}}

	n, err := ApplyMasInstalls(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyMasInstalls: %v", err)
	}
	if n != 2 {
		t.Fatalf("count = %d, want 2", n)
	}
	if got := strings.Join(runner.calls, ","); got != "install 497799835,install 409183694" {
		t.Fatalf("calls = %q", got)
	}
}

func TestApplyMasInstalls_ContinuesAfterFailure(t *testing.T) {
	runner := &recordingMasRunner{failIDs: map[string]bool{"1": true}}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionMasInstall, Name: "Bad", Value: "1"},
		{Type: plan.ActionMasInstall, Name: "Good", Value: "2"},
	}}

	n, err := ApplyMasInstalls(context.Background(), runner, p, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
	if got := strings.Join(runner.calls, ","); got != "install 1,install 2" {
		t.Fatalf("calls = %q", got)
	}
}
