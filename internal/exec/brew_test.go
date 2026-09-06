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
	if len(args) >= 2 && args[0] == "tap" {
		if len(args) == 3 {
			r.installs = append(r.installs, "tap:"+args[1]+"@"+args[2])
		} else {
			r.installs = append(r.installs, "tap:"+args[1])
		}
		return nil, nil
	}
	if len(args) == 3 && args[0] == "trust" && args[1] == "--tap" {
		r.installs = append(r.installs, "trust:"+args[2])
		return nil, nil
	}
	if len(args) == 1 && args[0] == "update" {
		r.installs = append(r.installs, "update")
		return nil, nil
	}
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
	if len(args) >= 3 && args[0] == "install" && args[1] == "--cask" {
		for _, name := range args[2:] {
			r.installs = append(r.installs, "cask:"+name)
		}
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

func TestAddTap_TrustsAfterTap(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := AddTap(context.Background(), runner, "nikitabobko/tap", "", true); err != nil {
		t.Fatalf("AddTap: %v", err)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:nikitabobko/tap,trust:nikitabobko/tap" {
		t.Fatalf("calls = %q, want tap then trust", got)
	}
}

func TestAddTap_WithURL(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := AddTap(context.Background(), runner, "jundot/omlx", "https://github.com/jundot/omlx", true); err != nil {
		t.Fatalf("AddTap: %v", err)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:jundot/omlx@https://github.com/jundot/omlx,trust:jundot/omlx" {
		t.Fatalf("calls = %q, want tap with url then trust", got)
	}
}

func TestAddTap_SkipsTrustWhenUntrusted(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := AddTap(context.Background(), runner, "nikitabobko/tap", "", false); err != nil {
		t.Fatalf("AddTap: %v", err)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:nikitabobko/tap" {
		t.Fatalf("calls = %q, want tap only when trusted=false", got)
	}
}

func TestAddTap_SkipsTrustForOfficial(t *testing.T) {
	runner := &installRecordingRunner{}
	if err := AddTap(context.Background(), runner, "homebrew/cask-fonts", "", true); err != nil {
		t.Fatalf("AddTap: %v", err)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:homebrew/cask-fonts" {
		t.Fatalf("calls = %q, want tap only for official", got)
	}
}

func TestApplyTapAdds_OnlyTaps(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTapAdd, Name: "homebrew/cask-fonts", Trusted: true},
		{Type: plan.ActionFormulaInstall, Name: "fzf"},
		{Type: plan.ActionTapAdd, Name: "nikitabobko/tap", Trusted: true},
	}}
	n, err := ApplyTapAdds(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyTapAdds: %v", err)
	}
	if n != 2 {
		t.Fatalf("count = %d, want 2", n)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:homebrew/cask-fonts,tap:nikitabobko/tap,trust:nikitabobko/tap,update" {
		t.Fatalf("order = %q", got)
	}
}

func TestApplyTapAdds_SkipsTrustWhenUntrusted(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTapAdd, Name: "nikitabobko/tap", Trusted: false},
	}}
	n, err := ApplyTapAdds(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatalf("ApplyTapAdds: %v", err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
	if got := strings.Join(runner.installs, ","); got != "tap:nikitabobko/tap,update" {
		t.Fatalf("order = %q, want tap only then update", got)
	}
}

func TestApplyTapAdds_NoUpdateWhenOnlyTrust(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionTapTrust, Name: "nikitabobko/tap"},
	}}
	n, err := ApplyTapAdds(context.Background(), runner, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	if got := strings.Join(runner.installs, ","); got != "trust:nikitabobko/tap" {
		t.Fatalf("got %q, want trust only (no update)", got)
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
	sudoCalls := 0
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		sudoCalls++
		return nil
	}
	t.Cleanup(func() { ensureSudo = orig })

	n, err := ApplyCaskInstalls(context.Background(), runner, p, nil, nil)
	if err != nil {
		t.Fatalf("ApplyCaskInstalls: %v", err)
	}
	if n != 2 {
		t.Fatalf("installed count = %d, want 2", n)
	}
	if sudoCalls != 1 {
		t.Fatalf("EnsureSudo calls=%d, want 1", sudoCalls)
	}
	if got := strings.Join(runner.installs, ","); got != "cask:raycast,cask:iterm2" {
		t.Fatalf("install order = %q, want single batch cask:raycast,cask:iterm2", got)
	}
}

func TestApplyCaskInstalls_EnsureSudoBeforeAuth(t *testing.T) {
	runner := &installRecordingRunner{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskInstall, Name: "vlc"},
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

	_, err := ApplyCaskInstalls(context.Background(), runner, p, func() { auth++ }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if auth != 1 {
		t.Fatalf("beforeAuth calls=%d, want 1", auth)
	}
}

func TestApplyCaskInstalls_ContinuesAfterChunkFailure(t *testing.T) {
	// First chunk (size 8) fails; second chunk succeeds.
	names := make([]plan.Action, 0, 10)
	for i := 0; i < 8; i++ {
		names = append(names, plan.Action{Type: plan.ActionCaskInstall, Name: fmt.Sprintf("bad%d", i)})
	}
	names = append(names,
		plan.Action{Type: plan.ActionCaskInstall, Name: "raycast"},
		plan.Action{Type: plan.ActionCaskInstall, Name: "vlc"},
	)
	runner := &selectiveFailInstallRunner{failNames: map[string]bool{"batch": true}, failBatchOnce: true}
	p := plan.Plan{Actions: names}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error { return nil }
	t.Cleanup(func() { ensureSudo = orig })

	n, err := ApplyCaskInstalls(context.Background(), runner, p, nil, nil)
	if err == nil {
		t.Fatal("expected error from first chunk")
	}
	if n != 2 {
		t.Fatalf("installed count = %d, want 2 (second chunk)", n)
	}
	if got := strings.Join(runner.installs, ","); got != "cask:raycast,cask:vlc" {
		t.Fatalf("installs = %q, want second chunk only", got)
	}
}

func TestApplyCaskInstalls_ChunksByEight(t *testing.T) {
	runner := &chunkCountingRunner{}
	var actions []plan.Action
	for i := 0; i < 10; i++ {
		actions = append(actions, plan.Action{Type: plan.ActionCaskInstall, Name: fmt.Sprintf("cask%d", i)})
	}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error { return nil }
	t.Cleanup(func() { ensureSudo = orig })

	n, err := ApplyCaskInstalls(context.Background(), runner, plan.Plan{Actions: actions}, nil, nil)
	if err != nil {
		t.Fatalf("ApplyCaskInstalls: %v", err)
	}
	if n != 10 {
		t.Fatalf("installed = %d, want 10", n)
	}
	if len(runner.chunkSizes) != 2 || runner.chunkSizes[0] != 8 || runner.chunkSizes[1] != 2 {
		t.Fatalf("chunkSizes = %v, want [8 2]", runner.chunkSizes)
	}
}

type chunkCountingRunner struct {
	chunkSizes []int
}

func (r *chunkCountingRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) >= 3 && args[0] == "install" && args[1] == "--cask" {
		r.chunkSizes = append(r.chunkSizes, len(args)-2)
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected brew args: %v", args)
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

type selectiveFailInstallRunner struct {
	installs      []string
	failNames     map[string]bool
	failBatchOnce bool
	batchFailed   bool
}

func (r *selectiveFailInstallRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) == 2 && args[0] == "install" {
		if r.failNames[args[1]] {
			return nil, fmt.Errorf("no available formula")
		}
		r.installs = append(r.installs, args[1])
		return nil, nil
	}
	if len(args) >= 3 && args[0] == "install" && args[1] == "--cask" {
		if r.failNames["batch"] {
			if r.failBatchOnce {
				if !r.batchFailed {
					r.batchFailed = true
					return nil, fmt.Errorf("no available cask")
				}
			} else {
				return nil, fmt.Errorf("no available cask")
			}
		}
		for _, name := range args[2:] {
			key := "cask:" + name
			if r.failNames[key] {
				return nil, fmt.Errorf("no available cask")
			}
			r.installs = append(r.installs, key)
		}
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
		{Type: plan.ActionMasInstall, Name: "Xcode", Value: "497799835"},
		{Type: plan.ActionMasRemove, Name: "WhatsApp Messenger", Value: "310633997"},
		{Type: plan.ActionLinkCreate, Name: "/tmp/x", Source: "config/x"},
		{Type: plan.ActionLinkUpdate, Name: "/tmp/y", Source: "config/y"},
		{Type: plan.ActionManagedCopy, Name: "/tmp/m", Source: "config/m"},
		{Type: plan.ActionFileUnlink, Name: "/tmp/u"},
		{Type: plan.ActionFilePrune, Name: "/tmp/p"},
		{Type: plan.ActionCaskRename, Name: "windsurf", Value: "devin-desktop"},
		{Type: plan.ActionFormulaUpgrade, Name: "git"},
		{Type: plan.ActionMasUpgrade, Name: "Keynote", Value: "409183694"},
		{Type: plan.ActionTemplateWrite, Name: "/tmp/t", Source: "config/t.tmpl"},
	}}
	skipped := UnsupportedApplyActions(p)
	if len(skipped) != 2 {
		t.Fatalf("skipped = %+v, want formula_upgrade and mas_upgrade", skipped)
	}
	if skipped[0].Type != plan.ActionFormulaUpgrade || skipped[1].Type != plan.ActionMasUpgrade {
		t.Fatalf("skipped = %+v, want formula_upgrade then mas_upgrade", skipped)
	}
}

func TestCaskRenameActions(t *testing.T) {
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionCaskInstall, Name: "raycast"},
		{Type: plan.ActionCaskRename, Name: "windsurf", Value: "devin-desktop"},
		{Type: plan.ActionCaskRename, Name: "vmware-horizon-client", Value: "omnissa-horizon-client"},
	}}
	got := CaskRenameActions(p)
	if len(got) != 2 {
		t.Fatalf("got %d renames, want 2", len(got))
	}
	if got[0].Name != "windsurf" || got[1].Name != "vmware-horizon-client" {
		t.Fatalf("got %+v", got)
	}
}
