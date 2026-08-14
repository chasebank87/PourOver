package exec

import (
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
