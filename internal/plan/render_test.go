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

func TestRenderText_FileActions(t *testing.T) {
	got := RenderText(Plan{Actions: []Action{
		{Type: ActionLinkCreate, Name: "~/.config/nvim", Source: "config/nvim"},
		{Type: ActionLinkUpdate, Name: "~/.zshrc", Source: "config/home/zshrc"},
		{Type: ActionLinkReplace, Name: "~/.gitconfig", Source: "config/home/gitconfig"},
	}})
	want := "create file ~/.config/nvim <- config/nvim\nupdate file ~/.zshrc <- config/home/zshrc\nreplace file ~/.gitconfig <- config/home/gitconfig (backup)\n"
	if got != want {
		t.Fatalf("RenderText() = %q, want %q", got, want)
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
