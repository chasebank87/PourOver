package pam

import (
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestRenderSudoLocal_FullFlags(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		Reattach:    true,
		TouchIDAuth: true,
		WatchIDAuth: true,
	}
	reattach := "/opt/homebrew/lib/pam/pam_reattach.so"
	watchid := "/opt/homebrew/lib/pam/pam_watchid.so"

	got := RenderSudoLocal(cfg, reattach, watchid)
	if !strings.Contains(got, ManagedMarker) {
		t.Fatalf("missing managed marker %q in:\n%s", ManagedMarker, got)
	}

	lines := nonEmptyLines(got)
	want := []string{
		ManagedMarker,
		"auth optional " + reattach,
		"auth sufficient pam_tid.so",
		"auth sufficient " + watchid,
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(want), got)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestRenderSudoLocal_TouchIDOnly(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		TouchIDAuth: true,
	}
	got := RenderSudoLocal(cfg, "/unused/reattach.so", "/unused/watchid.so")
	lines := nonEmptyLines(got)
	want := []string{
		ManagedMarker,
		"auth sufficient pam_tid.so",
	}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d:\n%s", len(lines), len(want), got)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, lines[i], want[i])
		}
	}
}

func TestRenderSudoLocal_OrderOmitsDisabled(t *testing.T) {
	cfg := config.SudoLocalPAM{
		Configured:  true,
		Enable:      true,
		Reattach:    true,
		WatchIDAuth: true,
	}
	reattach := "/prefix/lib/pam/pam_reattach.so"
	watchid := "/prefix/lib/pam/pam_watchid.so"
	got := RenderSudoLocal(cfg, reattach, watchid)
	if strings.Contains(got, "pam_tid") {
		t.Fatalf("unexpected pam_tid when TouchIDAuth=false:\n%s", got)
	}
	lines := nonEmptyLines(got)
	want := []string{
		ManagedMarker,
		"auth optional " + reattach,
		"auth sufficient " + watchid,
	}
	for i := range want {
		if i >= len(lines) || lines[i] != want[i] {
			t.Fatalf("order mismatch at %d; got:\n%s", i, got)
		}
	}
}

func TestRenderSudoLocal_EmptyWhenDisabledOrUnconfigured(t *testing.T) {
	reattach := "/opt/homebrew/lib/pam/pam_reattach.so"
	watchid := "/opt/homebrew/lib/pam/pam_watchid.so"

	disabled := config.SudoLocalPAM{
		Configured:  true,
		Enable:      false,
		TouchIDAuth: true,
	}
	if got := RenderSudoLocal(disabled, reattach, watchid); got != "" {
		t.Errorf("Enable=false: got %q, want empty", got)
	}

	unconfigured := config.SudoLocalPAM{
		Enable:      true,
		TouchIDAuth: true,
	}
	if got := RenderSudoLocal(unconfigured, reattach, watchid); got != "" {
		t.Errorf("!Configured: got %q, want empty", got)
	}
}

func TestIsPourOverManaged(t *testing.T) {
	if !IsPourOverManaged([]byte(ManagedMarker + "\nauth sufficient pam_tid.so\n")) {
		t.Fatal("expected true when marker present")
	}
	if IsPourOverManaged([]byte("auth sufficient pam_tid.so\n")) {
		t.Fatal("expected false when marker absent")
	}
	if IsPourOverManaged(nil) {
		t.Fatal("expected false for nil")
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
