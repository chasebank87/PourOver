package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestEnabled_QuietOrNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	var buf bytes.Buffer
	if Enabled(&buf, false) {
		t.Fatal("non-file writer should be disabled")
	}
	if Enabled(&buf, true) {
		t.Fatal("quiet should disable")
	}
	t.Setenv("NO_COLOR", "1")
	if Enabled(&buf, false) {
		t.Fatal("NO_COLOR should disable")
	}
}

func TestSession_StepFailFinishPlain(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(2)
	s.SetPhase("formulae")
	s.Step("install formula neofetch")
	s.Fail(errors.New("brew install neofetch: gone"))
	s.Step("install formula onefetch")
	s.Finish(Summary{Formulae: 1, Failures: 1})

	out := buf.String()
	if !strings.Contains(out, "PourOver") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "apply") {
		t.Fatalf("missing mode: %q", out)
	}
	if !strings.Contains(out, "failed: brew install neofetch") {
		t.Fatalf("missing failure: %q", out)
	}
	if !strings.Contains(out, "Installed 1 formula(s).") {
		t.Fatalf("missing summary: %q", out)
	}
	if !strings.Contains(out, "1 action(s) failed.") {
		t.Fatalf("missing failure summary: %q", out)
	}
	if s.FailureCount() != 1 {
		t.Fatalf("FailureCount = %d", s.FailureCount())
	}
}

func TestSession_StartZero_NoProgressBar(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(0)
	s.Finish(Summary{Renames: 2})
	out := buf.String()
	if strings.Contains(out, "0/0") || strings.Contains(out, "starting") {
		t.Fatalf("unexpected progress line: %q", out)
	}
	sep := strings.Repeat("─", 40)
	if strings.Count(out, sep) != 1 {
		t.Fatalf("want single header rule (no empty closing box): %q", out)
	}
	if strings.Contains(out, "No changes") {
		t.Fatalf("renames should suppress No changes: %q", out)
	}
}

func TestSession_WritePassthrough(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	if _, err := s.Write([]byte("☕ Pouring fzf\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "☕ Pouring fzf") {
		t.Fatalf("brew line missing: %q", buf.String())
	}
}

func TestSession_WriteInstallerLineIsNotAuthHint(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	before := buf.Len()
	line := "☕ Running installer for gamemaker with `sudo` (which may request your password)...\n"
	if _, err := s.Write([]byte(line)); err != nil {
		t.Fatal(err)
	}
	got := buf.String()[before:]
	if strings.Contains(got, "authentication required") {
		t.Fatalf("installer line must not print auth hint (hint waits for Password:): %q", got)
	}
	if !strings.Contains(got, "Running installer for gamemaker") {
		t.Fatalf("missing installer line: %q", got)
	}
}

func TestSession_WriteAuthPromptHint(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	s.Step("install cask sony-ps-remote-play")
	before := buf.Len()
	if _, err := s.Write([]byte("Password:")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()[before:]
	if !strings.Contains(got, "authentication required") {
		t.Fatalf("missing auth hint: %q", got)
	}
	if strings.Contains(got, "Password:") {
		t.Fatalf("must not reprint Password: as a typable line: %q", got)
	}
}

func TestSession_PrepareAuth_ParksLiveStatus(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	s.Step("pam write /etc/pam.d/sudo_local")
	before := buf.String()
	if !strings.Contains(before, "pam write") {
		t.Fatalf("missing step: %q", before)
	}
	s.PrepareAuth()
	got := buf.String()[len(before):]
	if !strings.Contains(got, "authentication required") {
		t.Fatalf("missing auth hint after PrepareAuth: %q", got)
	}
	if !strings.Contains(got, "\r\033[2K\n") {
		t.Fatalf("PrepareAuth must clear live status with CR+CSI: %q", got)
	}
	// After park, a following Password: must not sit on the bar line.
	if _, err := buf.WriteString("Password:\n"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "sudo_localPassword:") {
		t.Fatalf("Password glued to progress label: %q", out)
	}
	hintIdx := strings.Index(out, "authentication required")
	passIdx := strings.Index(out, "Password:")
	if hintIdx < 0 || passIdx < hintIdx {
		t.Fatalf("Password should follow auth hint on a later line: %q", out)
	}
	between := out[hintIdx:passIdx]
	if !strings.Contains(between, "\n") {
		t.Fatalf("Password glued to auth hint: %q", out)
	}
}

func TestSession_PrepareAuth_BreaksMidLineWhenNotLive(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	// Simulate a mid-line write without liveStatus (cursor not parked).
	buf.WriteString("░░░░  0/1  defaults  → defaults write /Library/Preferences/x")
	s.PrepareAuth()
	out := buf.String()
	if strings.Contains(out, "Preferences/x☕") || strings.Contains(out, "Preferences/xauthentication") {
		t.Fatalf("auth hint glued to mid-line content: %q", out)
	}
	if !strings.Contains(out, "authentication required") {
		t.Fatalf("missing auth hint: %q", out)
	}
}

func TestSession_PreparePrompt_ParksLiveStatus(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	s.Step("update file ~/.config/thefuck/__pycache__/x.pyc")
	before := buf.String()
	s.PreparePrompt()
	got := buf.String()[len(before):]
	if !strings.Contains(got, "\r\033[2K\n") {
		t.Fatalf("PreparePrompt must clear live status with CR+CSI: %q", got)
	}
	if strings.Contains(got, "authentication required") {
		t.Fatalf("PreparePrompt must not print auth hint: %q", got)
	}
	if _, err := buf.WriteString("Remove files?\n"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, ".pycRemove") {
		t.Fatalf("confirm glued to progress label: %q", out)
	}
}

func TestSession_StatusTruncatesToTerminalWidth(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.cols = 64
	s.Start(1)
	long := "defaults write /Library/Preferences/SystemConfiguration/com.apple.smb.server ServerDescription = Chase’s MacBook Pro"
	s.SetPhase("defaults")
	s.Step(long)

	s.mu.Lock()
	line := s.statusLineLocked()
	s.mu.Unlock()
	if strings.Contains(line, "SystemConfiguration") {
		t.Fatalf("expected truncated status without full path: %q", line)
	}
	if !strings.Contains(line, "…") {
		t.Fatalf("expected ellipsis in truncated status: %q", line)
	}
	if w := ansi.StringWidth(line); w > s.cols {
		t.Fatalf("status width %d > terminal %d: %q", w, s.cols, line)
	}
	out := buf.String()
	if strings.Contains(out, "SystemConfiguration") {
		t.Fatalf("buffer still has full path: %q", out)
	}
}

func TestSession_SingleLiveStatusLine(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(3)
	s.SetPhase("formulae")
	s.Step("install formula nerdfetch")
	s.SetPhase("casks") // must not reprint a second bar with stale formula label
	s.Step("install cask cheatsheet")
	s.Step("install cask cinebench")

	out := buf.String()
	// Status updates use CR overlays; only the initial status should start a new line of bars.
	if strings.Count(out, "\n░")+strings.Count(out, "\n█") > 1 {
		t.Fatalf("stacked progress bars on new lines: %q", out)
	}
	if strings.Contains(out, "casks  → install formula nerdfetch") {
		t.Fatalf("phase change reprinted stale action: %q", out)
	}
	if !strings.Contains(out, "install cask cinebench") {
		t.Fatalf("missing latest action: %q", out)
	}
	if !strings.Contains(out, "\r") {
		t.Fatalf("expected CR overlays for live status: %q", out)
	}
}

func TestSession_ParkBeforeBrewWrite(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "apply")
	s.Start(1)
	s.Step("install cask foo")
	if _, err := s.Write([]byte("☕ Fetching downloads\n")); err != nil {
		t.Fatal(err)
	}
	s.Step("install cask bar")
	out := buf.String()
	if !strings.Contains(out, "☕ Fetching downloads") {
		t.Fatalf("brew line missing: %q", out)
	}
	if !strings.Contains(out, "install cask bar") {
		t.Fatalf("status after brew missing: %q", out)
	}
}

func TestProgressAdapter_RoutesFailed(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	s := NewSession(&buf, "upgrade")
	s.Start(1)
	cb := s.ProgressAdapter()
	cb("install formula fzf")
	cb("failed: boom")
	if s.FailureCount() != 1 {
		t.Fatalf("FailureCount = %d", s.FailureCount())
	}
	if !strings.Contains(buf.String(), "failed: boom") {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestRenderBar(t *testing.T) {
	ForcePlain()
	got := renderBar(0, 10, 10)
	if strings.Count(got, "░") != 10 {
		t.Fatalf("empty bar = %q", got)
	}
	got = renderBar(10, 10, 10)
	if strings.Count(got, "█") != 10 {
		t.Fatalf("full bar = %q", got)
	}
}
