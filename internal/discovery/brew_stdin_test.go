package discovery

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecRunner_MutationInheritsStdin(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-brew")
	// Read one line from stdin and echo it; proves Stdin is attached.
	content := "#!/bin/sh\nread line\necho \"got:$line\"\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	in := strings.NewReader("from-stdin\n")
	var out bytes.Buffer
	r := &ExecRunner{
		Path:              script,
		MutationTimeout:   5 * time.Second,
		IdleTimeout:       -1,
		HeartbeatInterval: -1,
		Stdin:             in,
		Stdout:            &out,
		Stderr:            &out,
	}
	got, err := r.Run(context.Background(), "install", "pkg")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(got), "got:from-stdin") {
		t.Fatalf("stdout capture = %q", got)
	}
}

func TestExecRunner_MutationHeartbeatOnSilence(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-brew")
	// Silent sleep then exit — triggers idle heartbeat.
	content := "#!/bin/sh\nsleep 0.08\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	r := &ExecRunner{
		Path:              script,
		MutationTimeout:   5 * time.Second,
		IdleTimeout:       -1,
		HeartbeatInterval: 25 * time.Millisecond,
		Stdin:             strings.NewReader(""),
		Stdout:            &out,
		Stderr:            &out,
	}
	if _, err := r.Run(context.Background(), "install", "--cask", "anaconda"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "still working: brew install --cask anaconda") {
		t.Fatalf("missing heartbeat: %q", out.String())
	}
}

func TestExecRunner_IdleTimeoutKillsSilentMutation(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-brew")
	content := "#!/bin/sh\nexec sleep 5\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	r := &ExecRunner{
		Path:              script,
		MutationTimeout:   10 * time.Second,
		IdleTimeout:       200 * time.Millisecond,
		HeartbeatInterval: -1,
		Stdin:             strings.NewReader(""),
	}
	start := time.Now()
	_, err := r.Run(context.Background(), "install", "--cask", "slow")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected idle kill error")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("elapsed %v, want idle kill well under absolute timeout", elapsed)
	}
}

func TestExecRunner_DiscoveryDoesNotRequireStdin(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-brew")
	content := "#!/bin/sh\necho ok\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	r := &ExecRunner{
		Path:              script,
		Timeout:           5 * time.Second,
		HeartbeatInterval: -1,
	}
	got, err := r.Run(context.Background(), "list", "--formula")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(string(got)) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestExecRunner_DoesNotForwardPasswordPrompt(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-brew")
	content := "#!/bin/sh\nprintf '%s' 'Password:'\necho\necho done\n"
	if err := os.WriteFile(script, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	var raw bytes.Buffer
	styled := NewBrewStyleWriter(&raw)
	r := &ExecRunner{
		Path:              script,
		MutationTimeout:   5 * time.Second,
		IdleTimeout:       -1,
		HeartbeatInterval: -1,
		Stdin:             strings.NewReader(""),
		Stdout:            styled,
		Stderr:            styled,
	}
	if _, err := r.Run(context.Background(), "install", "--cask", "foo"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := raw.String()
	if strings.Contains(got, "Password:") {
		t.Fatalf("must not reprint Password: as a typable line: %q", got)
	}
	if !strings.Contains(got, "authentication required") {
		t.Fatalf("want auth hint, got %q", got)
	}
}
