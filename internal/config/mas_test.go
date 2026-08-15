package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMasManifest(t *testing.T, packagesBody string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pourover.lua")
	src := `
return {
  packages = {
` + packagesBody + `
  },
  files = { links = {} },
  policy = { uninstall_mode = "safe" },
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDecodePackages_MasMap(t *testing.T) {
	path := writeMasManifest(t, `
    formulae = {},
    casks = {},
    mas = {
      Xcode = 497799835,
      ["1Password for Safari"] = 1569813296,
    },
`)

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if !manifest.Packages.MasConfigured {
		t.Fatal("MasConfigured = false, want true")
	}
	got := manifest.Packages.Mas
	if len(got) != 2 {
		t.Fatalf("Mas = %#v, want 2 apps", got)
	}
	// Sorted by ID ascending.
	if got[0].Name != "Xcode" || got[0].ID != 497799835 {
		t.Errorf("Mas[0] = %#v, want Xcode/497799835", got[0])
	}
	if got[1].Name != "1Password for Safari" || got[1].ID != 1569813296 {
		t.Errorf("Mas[1] = %#v, want 1Password for Safari/1569813296", got[1])
	}
}

func TestDecodePackages_MasOmittedUnmanaged(t *testing.T) {
	path := writeMasManifest(t, `
    formulae = {},
    casks = {},
`)

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if manifest.Packages.MasConfigured {
		t.Fatal("MasConfigured = true, want false when mas key omitted")
	}
	if len(manifest.Packages.Mas) != 0 {
		t.Fatalf("Mas = %#v, want empty", manifest.Packages.Mas)
	}
}

func TestDecodePackages_MasEmptyConfigured(t *testing.T) {
	path := writeMasManifest(t, `
    formulae = {},
    casks = {},
    mas = {},
`)

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if !manifest.Packages.MasConfigured {
		t.Fatal("MasConfigured = false, want true when mas = {}")
	}
	if len(manifest.Packages.Mas) != 0 {
		t.Fatalf("Mas = %#v, want len 0", manifest.Packages.Mas)
	}
}

func TestDecodePackages_MasDuplicateID(t *testing.T) {
	path := writeMasManifest(t, `
    formulae = {},
    casks = {},
    mas = {
      Xcode = 497799835,
      ["Also Xcode"] = 497799835,
    },
`)

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for duplicate MAS app ID, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "497799835") {
		t.Fatalf("error = %v, want mention of duplicate ID", err)
	}
}

func TestDecodePackages_MasBadID(t *testing.T) {
	cases := []struct {
		name string
		mas  string
		want string
	}{
		{
			name: "zero",
			mas:  `mas = { Xcode = 0 },`,
			want: "id",
		},
		{
			name: "negative",
			mas:  `mas = { Xcode = -1 },`,
			want: "id",
		},
		{
			name: "non_int",
			mas:  `mas = { Xcode = 1.5 },`,
			want: "id",
		},
		{
			name: "non_number",
			mas:  `mas = { Xcode = "497799835" },`,
			want: "number",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeMasManifest(t, `
    formulae = {},
    casks = {},
    `+tc.mas+`
`)
			_, err := LoadManifest(path)
			if err == nil {
				t.Fatal("expected error for bad MAS id, got nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestDecodePackages_MasDuplicateName(t *testing.T) {
	// Lua maps cannot retain duplicate keys; exercise Validate on a crafted slice.
	err := Validate(&Manifest{
		Policy: Policy{UninstallMode: UninstallModeSafe},
		Packages: Packages{
			MasConfigured: true,
			Mas: []MasApp{
				{Name: "Xcode", ID: 1},
				{Name: "Xcode", ID: 2},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for duplicate MAS app name, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "Xcode") {
		t.Fatalf("error = %v, want mention of duplicate name", err)
	}
}

func TestDecodePackages_MasEmptyName(t *testing.T) {
	path := writeMasManifest(t, `
    formulae = {},
    casks = {},
    mas = {
      [""] = 497799835,
    },
`)

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected error for empty MAS app name, got nil")
	}
	if !strings.Contains(err.Error(), "name") && !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v, want mention of empty name", err)
	}
}
