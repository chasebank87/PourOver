package plan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderText_Golden(t *testing.T) {
	p := Plan{Actions: []Action{
		{Type: ActionFormulaInstall, Name: "fzf"},
		{Type: ActionFormulaInstall, Name: "git"},
		{Type: ActionCaskInstall, Name: "raycast"},
		{Type: ActionFormulaRemove, Name: "wget"},
	}}

	got := RenderText(p)
	want := readGolden(t, "plan-text.golden")
	if got != want {
		t.Errorf("RenderText() diff:\n--- got\n+++ want\n%s", diffLines(got, want))
	}
}

func TestRenderText_NoChanges(t *testing.T) {
	got := RenderText(Plan{})
	if got != "No changes.\n" {
		t.Errorf("RenderText(empty) = %q", got)
	}
}

func TestRenderJSON_StableShape(t *testing.T) {
	p := Plan{Actions: []Action{
		{Type: ActionFormulaInstall, Name: "git"},
	}}

	data, err := RenderJSON(p)
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}

	const want = `{
  "actions": [
    {
      "type": "formula_install",
      "name": "git"
    }
  ]
}
`
	if string(data) != want {
		t.Errorf("RenderJSON() = %s, want %s", string(data), want)
	}
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func diffLines(got, want string) string {
	return "got:\n" + got + "\nwant:\n" + want
}
