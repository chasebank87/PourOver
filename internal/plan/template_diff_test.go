package plan

import (
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildTemplatePlan_MissingDifferSame(t *testing.T) {
	p, err := BuildTemplatePlan([]discovery.TemplateStatus{
		{
			File:     config.TemplateFile{Source: "config/a.tmpl", Target: "~/.a"},
			Kind:     discovery.TemplateStatusMissing,
			Rendered: "new-a\n",
		},
		{
			File:     config.TemplateFile{Source: "config/b.tmpl", Target: "~/.b"},
			Kind:     discovery.TemplateStatusDiffer,
			Rendered: "new-b\n",
			Current:  "old-b\n",
		},
		{
			File:     config.TemplateFile{Source: "config/c.tmpl", Target: "~/.c"},
			Kind:     discovery.TemplateStatusSame,
			Rendered: "same\n",
			Current:  "same\n",
		},
	}, config.FileReplaceError)
	if err != nil {
		t.Fatal(err)
	}
	types := ActionTypes(p)
	want := []ActionType{ActionTemplateWrite, ActionTemplateWrite}
	if len(types) != len(want) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	if p.Actions[0].Name != "~/.a" || p.Actions[0].Source != "config/a.tmpl" {
		t.Fatalf("action[0] = %#v", p.Actions[0])
	}
	if p.Actions[0].Value == "" || !strings.Contains(p.Actions[0].Value, "+new-a") {
		t.Fatalf("missing Value should be unified diff, got %q", p.Actions[0].Value)
	}
	if p.Actions[1].Name != "~/.b" {
		t.Fatalf("action[1] = %#v", p.Actions[1])
	}
	if !strings.Contains(p.Actions[1].Value, "-old-b") || !strings.Contains(p.Actions[1].Value, "+new-b") {
		t.Fatalf("differ Value = %q", p.Actions[1].Value)
	}
}

func TestBuildTemplatePlan_BlockedErrorAndBackup(t *testing.T) {
	st := discovery.TemplateStatus{
		File:     config.TemplateFile{Source: "config/x.tmpl", Target: "~/.x"},
		Kind:     discovery.TemplateStatusBlocked,
		Rendered: "body\n",
	}
	_, err := BuildTemplatePlan([]discovery.TemplateStatus{st}, config.FileReplaceError)
	if err == nil {
		t.Fatal("expected error for blocked template target")
	}

	p, err := BuildTemplatePlan([]discovery.TemplateStatus{st}, config.FileReplaceBackup)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 1 || p.Actions[0].Type != ActionTemplateWrite || p.Actions[0].Kind != "backup" {
		t.Fatalf("action = %#v", p.Actions)
	}
}

func TestBuildTemplatePlan_TruncatesLongDiff(t *testing.T) {
	long := strings.Repeat("x", 5000) + "\n"
	p, err := BuildTemplatePlan([]discovery.TemplateStatus{{
		File:     config.TemplateFile{Source: "config/big.tmpl", Target: "~/.big"},
		Kind:     discovery.TemplateStatusMissing,
		Rendered: long,
	}}, config.FileReplaceError)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 1 {
		t.Fatalf("actions = %#v", p.Actions)
	}
	if len(p.Actions[0].Value) > 4096+64 {
		t.Fatalf("Value length %d, expected truncated near 4KB", len(p.Actions[0].Value))
	}
	if !strings.Contains(p.Actions[0].Value, "truncated") {
		t.Fatalf("Value missing truncation note: %q", p.Actions[0].Value[:min(80, len(p.Actions[0].Value))])
	}
}

func TestRenderText_TemplateWrite(t *testing.T) {
	got := RenderText(Plan{Actions: []Action{
		{Type: ActionTemplateWrite, Name: "~/.config/foo", Source: "config/foo.tmpl"},
		{Type: ActionTemplateWrite, Name: "~/.blocked", Source: "config/x.tmpl", Kind: "backup"},
	}})
	want := "template write ~/.config/foo <- config/foo.tmpl\ntemplate write ~/.blocked <- config/x.tmpl (backup)\n"
	if got != want {
		t.Fatalf("RenderText() = %q, want %q", got, want)
	}
}
