package config

import (
	"encoding/json"
	"testing"
)

func TestManifest_JSONTags(t *testing.T) {
	in := Manifest{
		Packages: Packages{
			Formulae: []string{"git", "fzf"},
			Casks:    []string{"raycast"},
		},
		Files: Files{
			Links: []FileLink{
				{Source: "config/nvim", Target: "~/.config/nvim"},
			},
		},
		Policy: Policy{UninstallMode: UninstallModeSafe},
		Backup: Backup{
			ICloud: ICloudBackup{Enabled: true, Path: "/custom/icloud"},
		},
	}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out Manifest
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(out.Packages.Formulae) != 2 || out.Packages.Formulae[0] != "git" {
		t.Errorf("formulae: %#v", out.Packages.Formulae)
	}
	if out.Policy.UninstallMode != UninstallModeSafe {
		t.Errorf("uninstall_mode: %q", out.Policy.UninstallMode)
	}
	if !out.Backup.ICloud.Enabled || out.Backup.ICloud.Path != "/custom/icloud" {
		t.Errorf("icloud: %#v", out.Backup.ICloud)
	}
}
