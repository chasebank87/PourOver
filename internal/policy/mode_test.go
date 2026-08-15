package policy

import (
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestResolveMode_DefaultsToSafe(t *testing.T) {
	if got := ResolveMode(""); got != config.UninstallModeSafe {
		t.Fatalf("ResolveMode(\"\") = %q, want %q", got, config.UninstallModeSafe)
	}
}

func TestResolveMode_ExplicitModes(t *testing.T) {
	cases := []struct {
		in   string
		want config.UninstallMode
	}{
		{"safe", config.UninstallModeSafe},
		{"strict", config.UninstallModeStrict},
		{"non_destructive", config.UninstallModeNonDestructive},
	}
	for _, tc := range cases {
		if got := ResolveMode(tc.in); got != tc.want {
			t.Errorf("ResolveMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveMode_UnknownFallsBackToSafe(t *testing.T) {
	if got := ResolveMode("bogus"); got != config.UninstallModeSafe {
		t.Fatalf("ResolveMode(\"bogus\") = %q, want %q", got, config.UninstallModeSafe)
	}
}

func TestResolveMode_FromConfig(t *testing.T) {
	m := config.Manifest{Policy: config.Policy{UninstallMode: config.UninstallModeStrict}}
	if got := ResolveModeFromManifest(m); got != config.UninstallModeStrict {
		t.Fatalf("ResolveModeFromManifest = %q, want strict", got)
	}

	empty := config.Manifest{}
	if got := ResolveModeFromManifest(empty); got != config.UninstallModeSafe {
		t.Fatalf("empty manifest = %q, want safe", got)
	}
}

func TestResolveFileReplace(t *testing.T) {
	cases := []struct {
		in   string
		want config.FileReplaceMode
	}{
		{"", config.FileReplaceError},
		{"error", config.FileReplaceError},
		{"backup", config.FileReplaceBackup},
		{"force", config.FileReplaceBackup},
		{"bogus", config.FileReplaceError},
	}
	for _, tc := range cases {
		if got := ResolveFileReplace(tc.in); got != tc.want {
			t.Errorf("ResolveFileReplace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
