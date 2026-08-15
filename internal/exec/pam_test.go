package exec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/pam"
	"github.com/chasebank87/PourOver/internal/plan"
)

func TestApplyPAM_WriteAndDisableStub(t *testing.T) {
	root := t.TempDir()
	sudoLocal := filepath.Join(root, "sudo_local")
	sudoPath := filepath.Join(root, "sudo")
	if err := os.WriteFile(sudoPath, []byte("# sudo\nauth required pam_opendirectory.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	body := pam.ManagedMarker + "\nauth sufficient pam_tid.so\n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: sudoLocal, Value: body},
		{Type: plan.ActionPAMSudoInclude, Name: sudoPath},
	}}

	n, err := ApplyPAM(context.Background(), p, PAMApplyOptions{
		SudoLocalPath: sudoLocal,
		SudoPath:      sudoPath,
	}, nil)
	if err != nil {
		t.Fatalf("ApplyPAM write: %v", err)
	}
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	got, err := os.ReadFile(sudoLocal)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("sudo_local = %q, want %q", got, body)
	}
	sudoGot, err := os.ReadFile(sudoPath)
	if err != nil {
		t.Fatal(err)
	}
	if !pam.HasSudoLocalInclude(sudoGot) {
		t.Fatalf("sudo missing include:\n%s", sudoGot)
	}

	// Legacy remove action writes disabled stub (does not delete).
	removePlan := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalRemove, Name: sudoLocal},
	}}
	n, err = ApplyPAM(context.Background(), removePlan, PAMApplyOptions{
		SudoLocalPath: sudoLocal,
		SudoPath:      sudoPath,
	}, nil)
	if err != nil {
		t.Fatalf("ApplyPAM remove→stub: %v", err)
	}
	if n != 1 {
		t.Fatalf("remove n = %d, want 1", n)
	}
	got, err = os.ReadFile(sudoLocal)
	if err != nil {
		t.Fatalf("sudo_local should remain as stub: %v", err)
	}
	if string(got) != pam.DisabledSudoLocal {
		t.Fatalf("sudo_local = %q, want disabled stub", got)
	}
}

func TestApplyPAM_BackupUnmanagedOnWrite(t *testing.T) {
	root := t.TempDir()
	sudoLocal := filepath.Join(root, "sudo_local")
	old := []byte("auth sufficient pam_tid.so\n")
	if err := os.WriteFile(sudoLocal, old, 0o644); err != nil {
		t.Fatal(err)
	}
	body := pam.ManagedMarker + "\nauth sufficient pam_tid.so\n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: sudoLocal, Value: body},
	}}
	if _, err := ApplyPAM(context.Background(), p, PAMApplyOptions{SudoLocalPath: sudoLocal}, nil); err != nil {
		t.Fatalf("ApplyPAM: %v", err)
	}
	bak := sudoLocal + ".pourover.bak"
	got, err := os.ReadFile(bak)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if string(got) != string(old) {
		t.Fatalf("backup = %q, want %q", got, old)
	}
}

func TestApplyPAM_MissingSOPathErrors(t *testing.T) {
	root := t.TempDir()
	sudoLocal := filepath.Join(root, "sudo_local")
	missing := filepath.Join(root, "missing", "pam_reattach.so")
	body := pam.ManagedMarker + "\nauth optional " + missing + "\n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: sudoLocal, Value: body},
	}}
	_, err := ApplyPAM(context.Background(), p, PAMApplyOptions{SudoLocalPath: sudoLocal}, nil)
	if err == nil {
		t.Fatal("want error when .so path missing")
	}
	if !strings.Contains(err.Error(), missing) && !strings.Contains(err.Error(), ".so") {
		t.Fatalf("error = %v, want mention of missing module", err)
	}
	if _, err := os.Stat(sudoLocal); !os.IsNotExist(err) {
		t.Fatal("sudo_local should not be written when .so missing")
	}
}

func TestApplyPAM_EmptySOPathErrors(t *testing.T) {
	root := t.TempDir()
	sudoLocal := filepath.Join(root, "sudo_local")
	// Render-like line with empty module path (trailing space after optional).
	body := pam.ManagedMarker + "\nauth optional \n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: sudoLocal, Value: body},
	}}
	_, err := ApplyPAM(context.Background(), p, PAMApplyOptions{SudoLocalPath: sudoLocal}, nil)
	if err == nil {
		t.Fatal("want error for empty .so path")
	}
}

func TestPrepareSudoInstall_StagesFileNotDevStdin(t *testing.T) {
	dest := "/etc/pam.d/sudo_local"
	body := []byte("# pourover: managed\nauth sufficient pam_tid.so\n")
	args, cleanup, err := prepareSudoInstall(body, 0o644, dest)
	if err != nil {
		t.Fatalf("prepareSudoInstall: %v", err)
	}
	defer cleanup()

	if len(args) != 6 {
		t.Fatalf("args = %v, want [sudo install -m MODE src dest]", args)
	}
	if args[0] != "sudo" || args[1] != "install" || args[2] != "-m" || args[3] != "0644" {
		t.Fatalf("args prefix = %v", args[:4])
	}
	if args[5] != dest {
		t.Fatalf("dest arg = %q, want %q", args[5], dest)
	}
	for _, a := range args {
		if a == "/dev/stdin" {
			t.Fatal("sudo install must not use /dev/stdin (macOS rejects it)")
		}
	}
	staged := args[4]
	got, err := os.ReadFile(staged)
	if err != nil {
		t.Fatalf("staged file %s: %v", staged, err)
	}
	if string(got) != string(body) {
		t.Fatalf("staged content = %q, want %q", got, body)
	}

	cleanup()
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("cleanup left staged file %s: %v", staged, err)
	}
}

func TestWriteElevatedFile_CallsBeforeAuthForEtc(t *testing.T) {
	orig := elevatedWrite
	elevatedWrite = func(ctx context.Context, path string, data []byte, mode os.FileMode) error {
		return nil
	}
	t.Cleanup(func() { elevatedWrite = orig })

	called := false
	if err := writeElevatedFile(context.Background(), "/etc/pam.d/sudo_local", []byte("x\n"), 0o644, func() {
		called = true
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("BeforeAuth was not called before elevated /etc write")
	}

	called = false
	if err := writeElevatedFile(context.Background(), filepath.Join(t.TempDir(), "sudo_local"), []byte("x\n"), 0o644, func() {
		called = true
	}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("BeforeAuth must not run for non-/etc paths")
	}
}

func TestApplyPAM_EnsureSudoOnceForEtc(t *testing.T) {
	origElevated := elevatedWrite
	elevatedWrite = func(ctx context.Context, path string, data []byte, mode os.FileMode) error {
		return nil
	}
	t.Cleanup(func() { elevatedWrite = origElevated })

	sudoCalls := 0
	auth := 0
	origSudo := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		sudoCalls++
		if beforeAuth != nil {
			beforeAuth()
		}
		return nil
	}
	t.Cleanup(func() { ensureSudo = origSudo })

	body := pam.ManagedMarker + "\nauth sufficient pam_tid.so\n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: "/etc/pam.d/sudo_local", Value: body},
		{Type: plan.ActionPAMSudoInclude, Name: "/etc/pam.d/sudo"},
	}}
	n, err := ApplyPAM(context.Background(), p, PAMApplyOptions{
		BeforeAuth: func() { auth++ },
	}, nil)
	if err != nil {
		t.Fatalf("ApplyPAM: %v", err)
	}
	if n != 2 {
		t.Fatalf("n=%d, want 2", n)
	}
	if sudoCalls != 1 {
		t.Fatalf("EnsureSudo calls=%d, want 1", sudoCalls)
	}
	if auth != 1 {
		t.Fatalf("beforeAuth calls=%d, want 1", auth)
	}
}

func TestApplyPAM_NoEnsureSudoForTempPaths(t *testing.T) {
	root := t.TempDir()
	sudoLocal := filepath.Join(root, "sudo_local")
	body := pam.ManagedMarker + "\nauth sufficient pam_tid.so\n"
	p := plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionPAMSudoLocalWrite, Name: sudoLocal, Value: body},
	}}
	orig := ensureSudo
	ensureSudo = func(ctx context.Context, beforeAuth func()) error {
		t.Fatal("EnsureSudo should not run for non-/etc PAM paths")
		return nil
	}
	t.Cleanup(func() { ensureSudo = orig })

	if _, err := ApplyPAM(context.Background(), p, PAMApplyOptions{SudoLocalPath: sudoLocal}, nil); err != nil {
		t.Fatalf("ApplyPAM: %v", err)
	}
}
