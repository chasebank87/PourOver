package template

import (
	"strings"
	"testing"
)

func TestRender_HostnameUserHome(t *testing.T) {
	ctx := Context{
		Hostname: "mymac",
		User:     "chase",
		Home:     "/Users/chase",
		Env:      map[string]string{},
	}
	src := "host={{.Hostname}} user={{.User}} home={{.Home}}\n"
	got, err := Render(src, ctx)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want := "host=mymac user=chase home=/Users/chase\n"
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestRender_MissingKeyError(t *testing.T) {
	ctx := Context{Hostname: "h", User: "u", Home: "/home/u"}
	_, err := Render("{{.NoSuchField}}\n", ctx)
	if err == nil {
		t.Fatal("expected error for undefined field")
	}
	if !strings.Contains(err.Error(), "NoSuchField") && !strings.Contains(err.Error(), "can't evaluate") {
		t.Fatalf("error = %v, want missing key mention", err)
	}
}

func TestRender_RejectsUnknownFunc(t *testing.T) {
	ctx := Context{Hostname: "h", User: "u", Home: "/home/u"}
	_, err := Render(`{{exec "rm" "-rf" "/"}}`, ctx)
	if err == nil {
		t.Fatal("expected error for exec func (no custom FuncMap)")
	}
}

func TestDefaultContext_PopulatesIdentity(t *testing.T) {
	ctx, err := DefaultContext()
	if err != nil {
		t.Fatalf("DefaultContext: %v", err)
	}
	if ctx.Hostname == "" {
		t.Fatal("Hostname empty")
	}
	if ctx.User == "" {
		t.Fatal("User empty")
	}
	if ctx.Home == "" {
		t.Fatal("Home empty")
	}
	if ctx.Env == nil {
		t.Fatal("Env should be non-nil empty map")
	}
	if len(ctx.Env) != 0 {
		t.Fatalf("Env = %#v, want empty (no arbitrary env)", ctx.Env)
	}
}
