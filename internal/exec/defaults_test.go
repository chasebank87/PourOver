package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/plan"
)

type recordingDefaults struct {
	writes  [][]string
	killall []string
}

func (r *recordingDefaults) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	r.writes = append(r.writes, append([]string{}, args...))
	return nil, nil
}

func (r *recordingDefaults) Killall(ctx context.Context, process string) error {
	r.killall = append(r.killall, process)
	return nil
}

func TestApplyDefaultsWrites(t *testing.T) {
	rec := &recordingDefaults{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionDefaultsWrite, Domain: config.DomainDock, Key: "autohide", Value: "true", Kind: "bool"},
		{Type: plan.ActionDefaultsWrite, Domain: config.DomainFinder, Key: "ShowPathbar", Value: "true", Kind: "bool"},
		{Type: plan.ActionFormulaInstall, Name: "git"},
	}}
	n, err := ApplyDefaultsWrites(context.Background(), rec, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("n=%d", n)
	}
	if len(rec.writes) != 2 {
		t.Fatalf("writes=%v", rec.writes)
	}
	if got := strings.Join(rec.writes[0], " "); got != "write com.apple.dock autohide -bool true" {
		t.Fatalf("write0=%q", got)
	}
	joined := strings.Join(rec.killall, ",")
	if !strings.Contains(joined, "Dock") || !strings.Contains(joined, "Finder") {
		t.Fatalf("killall=%v", rec.killall)
	}
}

func TestDefaultsWriteArgs_StringExpand(t *testing.T) {
	args, err := defaultsWriteArgs(plan.Action{
		Domain: "com.apple.screencapture",
		Key:    "location",
		Value:  "~/Desktop",
		Kind:   "string",
	})
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "write" || args[3] != "-string" {
		t.Fatalf("args=%v", args)
	}
	if strings.HasPrefix(args[4], "~") {
		t.Fatalf("expected expanded home, got %q", args[4])
	}
}

func TestDefaultsWriteArgs_PersistentApps(t *testing.T) {
	args, err := defaultsWriteArgs(plan.Action{
		Domain: config.DomainDock,
		Key:    config.DockPersistentAppsKey,
		Value:  `["/Applications/Safari.app"]`,
		Kind:   "array",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 4 || args[0] != "write" || args[1] != config.DomainDock || args[2] != config.DockPersistentAppsKey {
		t.Fatalf("args=%v", args)
	}
	if !strings.Contains(args[3], "<array>") || !strings.Contains(args[3], "/Applications/Safari.app") {
		t.Fatalf("plist=%q", args[3])
	}
}

func TestApplyDefaultsWrites_ScreencaptureKillall(t *testing.T) {
	rec := &recordingDefaults{}
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionDefaultsWrite, Domain: "com.apple.screencapture", Key: "type", Value: "png", Kind: "string"},
	}}
	if _, err := ApplyDefaultsWrites(context.Background(), rec, p, nil); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rec.killall, ",")
	if !strings.Contains(joined, "SystemUIServer") {
		t.Fatalf("killall=%v", rec.killall)
	}
}

func TestApplyDefaultsWrites_UnsupportedKind(t *testing.T) {
	rec := &recordingDefaults{}
	_, err := ApplyDefaultsWrites(context.Background(), rec, plan.Plan{Actions: []plan.Action{{
		Type: plan.ActionDefaultsWrite, Domain: "x", Key: "y", Value: "z", Kind: "blob",
	}}}, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("err=%v", err)
	}
}

type failingDefaults struct{ err error }

func (f *failingDefaults) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	return nil, f.err
}

func (f *failingDefaults) Killall(ctx context.Context, process string) error { return nil }

func TestApplyDefaultsWrites_SystemDomainNeedsAdmin(t *testing.T) {
	rec := &failingDefaults{err: fmt.Errorf("permission denied")}
	_, err := ApplyDefaultsWrites(context.Background(), rec, plan.Plan{Actions: []plan.Action{{
		Type:   plan.ActionDefaultsWrite,
		Domain: "/Library/Preferences/com.apple.loginwindow",
		Key:    "GuestEnabled",
		Value:  "false",
		Kind:   "bool",
	}}}, nil)
	if err == nil || !strings.Contains(err.Error(), "requires admin privileges") {
		t.Fatalf("err=%v", err)
	}
}
