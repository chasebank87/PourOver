package exec

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	n, err := ApplyDefaultsWrites(context.Background(), rec, p, DefaultsApplyOptions{}, nil)
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
	if _, err := ApplyDefaultsWrites(context.Background(), rec, p, DefaultsApplyOptions{}, nil); err != nil {
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
	}}}, DefaultsApplyOptions{}, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported kind") {
		t.Fatalf("err=%v", err)
	}
}

func TestApplyDefaultsWrites_SystemDomainUsesSudo(t *testing.T) {
	rec := &recordingDefaults{}
	var gotArgs []string
	authCalls := 0
	orig := elevatedDefaultsWrite
	elevatedDefaultsWrite = func(ctx context.Context, timeout time.Duration, args []string, beforeAuth func()) error {
		if beforeAuth != nil {
			beforeAuth()
		}
		gotArgs = append([]string{}, args...)
		return nil
	}
	t.Cleanup(func() { elevatedDefaultsWrite = orig })

	n, err := ApplyDefaultsWrites(context.Background(), rec, plan.Plan{Actions: []plan.Action{{
		Type:   plan.ActionDefaultsWrite,
		Domain: "/Library/Preferences/com.apple.loginwindow",
		Key:    "LoginwindowText",
		Value:  "hello",
		Kind:   "string",
	}}}, DefaultsApplyOptions{BeforeAuth: func() { authCalls++ }}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
	if len(rec.writes) != 0 {
		t.Fatalf("user defaults applier should not run for system domain; writes=%v", rec.writes)
	}
	want := "write /Library/Preferences/com.apple.loginwindow LoginwindowText -string hello"
	if got := strings.Join(gotArgs, " "); got != want {
		t.Fatalf("elevated args=%q, want %q", got, want)
	}
	if authCalls != 1 {
		t.Fatalf("BeforeAuth calls=%d, want 1", authCalls)
	}
}

func TestApplyDefaultsWrites_SystemDomainElevatedError(t *testing.T) {
	orig := elevatedDefaultsWrite
	elevatedDefaultsWrite = func(ctx context.Context, timeout time.Duration, args []string, beforeAuth func()) error {
		return fmt.Errorf("sudo denied")
	}
	t.Cleanup(func() { elevatedDefaultsWrite = orig })

	_, err := ApplyDefaultsWrites(context.Background(), &recordingDefaults{}, plan.Plan{Actions: []plan.Action{{
		Type:   plan.ActionDefaultsWrite,
		Domain: "/Library/Preferences/com.apple.loginwindow",
		Key:    "GuestEnabled",
		Value:  "false",
		Kind:   "bool",
	}}}, DefaultsApplyOptions{}, nil)
	if err == nil || !strings.Contains(err.Error(), "writing system preference") || !strings.Contains(err.Error(), "sudo denied") {
		t.Fatalf("err=%v", err)
	}
}
