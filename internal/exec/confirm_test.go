package exec

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestConfirmYes_Yes(t *testing.T) {
	in := strings.NewReader("yes\n")
	var out strings.Builder
	if !ConfirmYes(in, &out, "Proceed?") {
		t.Fatal("ConfirmYes(yes) = false, want true")
	}
	if !strings.Contains(out.String(), "Proceed?") {
		t.Fatalf("prompt not written: %q", out.String())
	}
}

func TestConfirmYes_No(t *testing.T) {
	cases := []string{"no\n", "n\n", "\n", "maybe\n"}
	for _, input := range cases {
		in := strings.NewReader(input)
		var out strings.Builder
		if ConfirmYes(in, &out, "Proceed?") {
			t.Fatalf("ConfirmYes(%q) = true, want false", input)
		}
	}
}

func TestConfirmYes_Y(t *testing.T) {
	in := strings.NewReader("y\n")
	var out strings.Builder
	if !ConfirmYes(in, &out, "ok?") {
		t.Fatal("ConfirmYes(y) = false, want true")
	}
}

func TestConfirmYes_EOF(t *testing.T) {
	in := strings.NewReader("")
	var out strings.Builder
	if ConfirmYes(in, &out, "ok?") {
		t.Fatal("ConfirmYes(EOF) = true, want false")
	}
}

func TestConfirmYes_Writer(t *testing.T) {
	// Discard writer still works.
	in := strings.NewReader("yes\n")
	if !ConfirmYes(in, io.Discard, "ok?") {
		t.Fatal("want true")
	}
}

func TestFormatListPrompt_OnePerLine(t *testing.T) {
	got := FormatListPrompt("Remove PourOver-owned undeclared files", []string{
		"/tmp/a/.DS_Store",
		"/tmp/b/.DS_Store",
	})
	if strings.Contains(got, ", /") {
		t.Fatalf("paths must not be comma-joined: %q", got)
	}
	if !strings.Contains(got, "  /tmp/a/.DS_Store\n") || !strings.Contains(got, "  /tmp/b/.DS_Store\n") {
		t.Fatalf("missing list lines: %q", got)
	}
	if !strings.Contains(got, "(2 items):") || !strings.HasSuffix(got, "Proceed?") {
		t.Fatalf("missing title/proceed: %q", got)
	}
}

func TestFormatListPrompt_Truncates(t *testing.T) {
	items := make([]string, maxConfirmList+3)
	for i := range items {
		items[i] = fmt.Sprintf("/tmp/f%d", i)
	}
	got := FormatListPrompt("Remove", items)
	if strings.Count(got, "  /tmp/") != maxConfirmList {
		t.Fatalf("shown paths = %d, want %d in %q", strings.Count(got, "  /tmp/"), maxConfirmList, got)
	}
	if !strings.Contains(got, "… and 3 more") {
		t.Fatalf("missing truncation: %q", got)
	}
}
