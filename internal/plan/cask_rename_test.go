package plan

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type renameInfoRunner struct {
	infoJSON string
}

func (r *renameInfoRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if len(args) >= 3 && args[0] == "info" && args[1] == "--json=v2" && args[2] == "--cask" {
		return []byte(r.infoJSON), nil
	}
	return nil, fmt.Errorf("unexpected %v", args)
}

func TestAdviseCaskRenames(t *testing.T) {
	p := Plan{Actions: []Action{
		{Type: ActionCaskInstall, Name: "windsurf"},
		{Type: ActionCaskInstall, Name: "vmware-horizon-client"},
		{Type: ActionCaskInstall, Name: "raycast"},
		{Type: ActionCaskRemove, Name: "devin-desktop"},
		{Type: ActionCaskRemove, Name: "vlc"},
	}}
	runner := &renameInfoRunner{infoJSON: `{
  "casks": [
    {"token": "devin-desktop", "old_tokens": ["windsurf"]},
    {"token": "omnissa-horizon-client", "old_tokens": ["vmware-horizon-client"]},
    {"token": "raycast", "old_tokens": []}
  ]
}`}
	got, err := AdviseCaskRenames(context.Background(), runner, p, []string{
		"devin-desktop", "omnissa-horizon-client",
	})
	if err != nil {
		t.Fatal(err)
	}
	if names := ActionNames(got, ActionCaskInstall); len(names) != 1 || names[0] != "raycast" {
		t.Fatalf("installs = %v, want [raycast]", names)
	}
	if names := ActionNames(got, ActionCaskRename); len(names) != 2 || names[0] != "vmware-horizon-client" || names[1] != "windsurf" {
		t.Fatalf("renames = %v", names)
	}
	if names := ActionNames(got, ActionCaskRemove); len(names) != 1 || names[0] != "vlc" {
		t.Fatalf("removes = %v, want [vlc] (keepdevin-desktop)", names)
	}
	text := RenderText(got)
	want := fmt.Sprintf("cask renamed: %s → %s (update packages.lua)", "windsurf", "devin-desktop")
	if !strings.Contains(text, want) {
		t.Fatalf("text=%q, want substring %q", text, want)
	}
}
