package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteSummary_PlainAndFancy(t *testing.T) {
	ForcePlain()
	var plain bytes.Buffer
	WriteSummary(&plain, Summary{Formulae: 2, Failures: 1, Skipped: 1}, false)
	got := plain.String()
	if !strings.Contains(got, "Installed 2 formula(s).") {
		t.Fatalf("plain missing install: %q", got)
	}
	if strings.Contains(got, "☕") {
		t.Fatalf("plain should not use coffee prefix: %q", got)
	}
	if !strings.Contains(got, "1 action(s) failed.") || !strings.Contains(got, "Skipped 1 unsupported") {
		t.Fatalf("plain missing fail/skip: %q", got)
	}

	var fancy bytes.Buffer
	WriteSummary(&fancy, Summary{Casks: 1}, true)
	got = fancy.String()
	if !strings.Contains(got, "☕ Installed 1 cask(s).") {
		t.Fatalf("fancy missing cask line: %q", got)
	}
}

func TestWriteSummary_Empty(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	WriteSummary(&buf, Summary{}, false)
	if buf.String() != "No changes.\n" {
		t.Fatalf("got %q", buf.String())
	}
	buf.Reset()
	WriteSummary(&buf, Summary{}, true)
	if buf.String() != "☕ No changes.\n" {
		t.Fatalf("fancy empty = %q", buf.String())
	}
}
