package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
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
