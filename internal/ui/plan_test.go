package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/plan"
)

func TestWritePlan_Empty(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	WritePlan(&buf, plan.Plan{})
	got := buf.String()
	if !strings.Contains(got, "PourOver") || !strings.Contains(got, "plan") {
		t.Fatalf("missing header: %q", got)
	}
	if !strings.Contains(got, "No changes.") {
		t.Fatalf("missing empty state: %q", got)
	}
}

func TestWritePlan_PendingAndAdvisory(t *testing.T) {
	ForcePlain()
	var buf bytes.Buffer
	WritePlan(&buf, plan.Plan{Actions: []plan.Action{
		{Type: plan.ActionLinkUpdate, Name: "~/.zshrc", Source: "config/home/zshrc"},
		{Type: plan.ActionCaskRename, Name: "old", Value: "new"},
	}})
	got := buf.String()
	if !strings.Contains(got, "update file ~/.zshrc <- config/home/zshrc") {
		t.Fatalf("missing pending file line: %q", got)
	}
	if !strings.Contains(got, "cask renamed:") {
		t.Fatalf("missing rename advice: %q", got)
	}
}
