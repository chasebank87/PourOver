package config

import "testing"

func TestCatalogDesiredSettings_IncludesLoginwindowText(t *testing.T) {
	got := CatalogDesiredSettings()
	found := false
	for _, d := range got {
		if d.Section == "loginwindow" && d.Key == "LoginwindowText" {
			found = true
			if d.Domain != "/Library/Preferences/com.apple.loginwindow" {
				t.Fatalf("domain = %q", d.Domain)
			}
			if d.Value.Kind != SettingString {
				t.Fatalf("kind = %v", d.Value.Kind)
			}
			break
		}
	}
	if !found {
		t.Fatal("LoginwindowText missing from catalog flatten")
	}
}
