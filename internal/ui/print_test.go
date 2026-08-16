package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintHelpers_PlainWhenNotTTY(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	Header(&buf, "doctor")
	Successf(&buf, "☕ All checks passed.")
	Warnf(&buf, "warning: something")
	Failf(&buf, "☕ failed: boom")
	Mutedf(&buf, "☕ No changes.")
	Errorf(&buf, errString("nope"))

	out := buf.String()
	if !strings.Contains(out, "☕ PourOver  doctor") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, strings.Repeat("─", 40)) {
		t.Fatalf("missing rule: %q", out)
	}
	if !strings.Contains(out, "☕ All checks passed.") {
		t.Fatalf("missing success: %q", out)
	}
	if !strings.Contains(out, "warning: something") {
		t.Fatalf("missing warn: %q", out)
	}
	if !strings.Contains(out, "Error: nope") {
		t.Fatalf("missing error: %q", out)
	}
	// Buffers are not TTYs; Enabled is false so no ANSI CSI even without ForcePlain.
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("unexpected ANSI: %q", out)
	}
}

func TestErrorf_NilNoOp(t *testing.T) {
	var buf bytes.Buffer
	Errorf(&buf, nil)
	if buf.Len() != 0 {
		t.Fatalf("expected empty, got %q", buf.String())
	}
}

type errString string

func (e errString) Error() string { return string(e) }
