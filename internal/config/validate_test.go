package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_InvalidPolicy(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "invalid_policy.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "policy.uninstall_mode")
}

func TestLoadManifest_EmptyFormula(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_formula.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "packages.formulae[2]")
}

func TestLoadManifest_EmptyLinkTarget(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_link_target.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "files.links[1].target")
}

func TestLoadManifest_EmptyTemplateSource(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_template_source.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "files.templates[1].source")
}

func TestLoadManifest_EmptyUnlinkPath(t *testing.T) {
	path := filepath.Join("..", "..", "test", "fixtures", "config", "invalid", "empty_unlink.lua")
	_, err := LoadManifest(path)
	assertLoadErrorContains(t, err, "files.unlink[1]")
}

func TestValidate_UnknownUninstallMode(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallMode("bogus")},
	})
	if err == nil || !strings.Contains(err.Error(), "policy.uninstall_mode") {
		t.Fatalf("Validate() = %v, want uninstall_mode error", err)
	}
}

func TestValidate_UnknownFileReplace(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe, FileReplace: FileReplaceMode("wipe")},
	})
	if err == nil || !strings.Contains(err.Error(), "policy.file_replace") {
		t.Fatalf("Validate() = %v, want file_replace error", err)
	}
}

func TestValidate_UnknownFilesMode(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe, FilesMode: FilesMode("wipe")},
	})
	if err == nil || !strings.Contains(err.Error(), "policy.files_mode") {
		t.Fatalf("Validate() = %v, want files_mode error", err)
	}
}

func TestValidate_FileReplaceForceAlias(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe, FileReplace: FileReplaceMode("force")},
	})
	if err != nil {
		t.Fatalf("Validate() = %v, want nil for force alias", err)
	}
}

func TestValidate_CapitalizedPackageName(t *testing.T) {
	err := Validate(&Manifest{
		Policy:   Policy{UninstallMode: UninstallModeSafe},
		Packages: Packages{Casks: []string{"Raycast"}},
	})
	if err == nil || !strings.Contains(err.Error(), "lowercase Homebrew token") || !strings.Contains(err.Error(), "raycast") {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidate_EmptyTap(t *testing.T) {
	err := Validate(&Manifest{
		Policy:   Policy{UninstallMode: UninstallModeSafe},
		Packages: Packages{Taps: []TapSpec{{Name: "homebrew/cask-fonts"}, {Name: ""}}},
	})
	if err == nil || !strings.Contains(err.Error(), "packages.taps[2]") {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestValidate_EmptyUnlinkPath(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe},
		Files:  Files{Unlink: []string{"~/.keep", "  "}},
	})
	if err == nil || !strings.Contains(err.Error(), "files.unlink[2]") {
		t.Fatalf("Validate() = %v, want files.unlink[2] empty error", err)
	}
}

func TestValidate_EmptyTemplateFields(t *testing.T) {
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe},
		Files: Files{
			Templates: []TemplateFile{
				{Source: "config/ok.tmpl", Target: "~/.ok"},
				{Source: "  ", Target: "   "},
			},
		},
	})
	if err == nil {
		t.Fatal("Validate() = nil, want empty template field errors")
	}
	msg := err.Error()
	if !strings.Contains(msg, "files.templates[2].source") || !strings.Contains(msg, "files.templates[2].target") {
		t.Fatalf("Validate() = %v, want templates[2] source and target empty errors", err)
	}
}

func TestValidate_CapitalizedTap(t *testing.T) {
	err := Validate(&Manifest{
		Policy:   Policy{UninstallMode: UninstallModeSafe},
		Packages: Packages{Taps: []TapSpec{{Name: "Homebrew/Cask-Fonts"}}},
	})
	if err == nil || !strings.Contains(err.Error(), "lowercase Homebrew token") {
		t.Fatalf("Validate() = %v", err)
	}
}

func assertLoadErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want substring %q", err.Error(), substr)
	}
}
