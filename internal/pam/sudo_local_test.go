package pam

import (
	"os"
	"path/filepath"
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

func TestRenderSudoLocal_DisabledStubOrUnconfigured(t *testing.T) {
	reattach := "/opt/homebrew/lib/pam/pam_reattach.so"
	watchid := "/opt/homebrew/lib/pam/pam_watchid.so"

	disabled := config.SudoLocalPAM{
		Configured:  true,
		Enable:      false,
		TouchIDAuth: true,
	}
	got := RenderSudoLocal(disabled, reattach, watchid)
	if got != DisabledSudoLocal {
		t.Errorf("Enable=false: got %q, want stub %q", got, DisabledSudoLocal)
	}
	if strings.Contains(got, "\nauth ") || strings.HasPrefix(got, "auth ") {
		t.Errorf("disabled stub must not contain auth lines:\n%s", got)
	}
	if !IsPourOverManaged([]byte(got)) {
		t.Error("disabled stub must be PourOver-managed")
	}

	unconfigured := config.SudoLocalPAM{
		Enable:      true,
		TouchIDAuth: true,
	}
	if got := RenderSudoLocal(unconfigured, reattach, watchid); got != "" {
		t.Errorf("!Configured: got %q, want empty", got)
	}
}

func TestHasSudoLocalInclude(t *testing.T) {
	if !HasSudoLocalInclude([]byte("auth       include        sudo_local\n")) {
		t.Fatal("expected true for spaced include")
	}
	if !HasSudoLocalInclude([]byte("auth include sudo_local\n")) {
		t.Fatal("expected true for compact include")
	}
	if HasSudoLocalInclude([]byte("auth required pam_opendirectory.so\n")) {
		t.Fatal("expected false when include absent")
	}
}

func TestFindModule_InjectableCandidates(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.so")
	found := filepath.Join(root, "pam_watchid.so")
	if err := os.WriteFile(found, []byte("so"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := FindModule([]string{missing, found})
	if !ok || got != found {
		t.Fatalf("FindModule = %q,%v want %q,true", got, ok, found)
	}
	if _, ok := FindModule([]string{missing}); ok {
		t.Fatal("expected not found")
	}
}

func TestDefaultWatchIDSearchPaths(t *testing.T) {
	paths := DefaultWatchIDSearchPaths("/opt/homebrew")
	wantFirst := "/opt/homebrew/lib/pam/pam_watchid.so"
	if len(paths) == 0 || paths[0] != wantFirst {
		t.Fatalf("paths[0] = %v, want %q first", paths, wantFirst)
	}
	joined := strings.Join(paths, "\n")
	for _, s := range []string{
		"/opt/homebrew/lib/pam/pam_watchid.so.2",
		"/usr/local/lib/pam/pam_watchid.so",
	} {
		if !strings.Contains(joined, s) {
			t.Errorf("missing candidate %s in %v", s, paths)
		}
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
