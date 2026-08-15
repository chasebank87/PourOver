package discovery

import (
	"context"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

type mapDefaults struct{ vals map[string]string } // key "domain|key" → raw

func (m *mapDefaults) Defaults(ctx context.Context, args ...string) ([]byte, error) {
	k := args[1] + "|" + args[2]
	if v, ok := m.vals[k]; ok {
		return []byte(v), nil
	}
	return nil, &DefaultsExitError{Args: args, Stderr: "The domain/default pair does not exist"}
}

func TestSnapshotCatalogDefaults_SkipsMissingIncludesPresent(t *testing.T) {
	runner := &mapDefaults{vals: map[string]string{
		config.DomainDock + "|autohide": "1",
		// tilesize missing
	}}
	desired := []config.DesiredSetting{
		{
			Domain:  config.DomainDock,
			Key:     "autohide",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "dock",
		},
		{
			Domain:  config.DomainDock,
			Key:     "tilesize",
			Value:   config.SettingValue{Kind: config.SettingInt},
			Section: "dock",
		},
	}

	got, warnings, err := SnapshotCatalogDefaults(context.Background(), runner, desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1; got %#v warnings=%v", len(got), got, warnings)
	}
	if got[0].Section != "dock" || got[0].Key != "autohide" || got[0].Domain != config.DomainDock {
		t.Fatalf("entry = %#v", got[0])
	}
	if got[0].Value.Kind != config.SettingBool || !got[0].Value.Bool {
		t.Fatalf("value = %#v", got[0].Value)
	}
}

func TestSnapshotCatalogDefaults_AlsoDomainsCollapse(t *testing.T) {
	const primary = "com.apple.AppleMultitouchTrackpad"
	const alternate = "com.apple.driver.AppleBluetoothMultitouch.trackpad"
	runner := &mapDefaults{vals: map[string]string{
		alternate + "|Clicking": "1",
		// primary missing
	}}
	desired := []config.DesiredSetting{
		{
			Domain:  primary,
			Key:     "Clicking",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "trackpad",
		},
		{
			Domain:  alternate,
			Key:     "Clicking",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "trackpad",
		},
	}

	got, warnings, err := SnapshotCatalogDefaults(context.Background(), runner, desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (collapsed); got %#v warnings=%v", len(got), got, warnings)
	}
	if got[0].Section != "trackpad" || got[0].Key != "Clicking" {
		t.Fatalf("section/key = %#v", got[0])
	}
	if got[0].Domain != alternate {
		t.Fatalf("domain = %q, want alternate %q", got[0].Domain, alternate)
	}
	if !got[0].Value.Bool {
		t.Fatalf("value = %#v", got[0].Value)
	}
}

func TestSnapshotCatalogDefaults_SkipsParseError(t *testing.T) {
	runner := &mapDefaults{vals: map[string]string{
		config.DomainDock + "|autohide": "not-a-bool",
		config.DomainDock + "|tilesize": "48",
	}}
	desired := []config.DesiredSetting{
		{
			Domain:  config.DomainDock,
			Key:     "autohide",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "dock",
		},
		{
			Domain:  config.DomainDock,
			Key:     "tilesize",
			Value:   config.SettingValue{Kind: config.SettingInt},
			Section: "dock",
		},
	}

	got, warnings, err := SnapshotCatalogDefaults(context.Background(), runner, desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "tilesize" || got[0].Value.Int != 48 {
		t.Fatalf("got %#v warnings=%v", got, warnings)
	}
	if len(warnings) == 0 {
		t.Fatal("expected parse warning for autohide")
	}
}

func TestSnapshotCatalogDefaults_PrefersPrimaryDomain(t *testing.T) {
	const primary = "com.apple.AppleMultitouchTrackpad"
	const alternate = "com.apple.driver.AppleBluetoothMultitouch.trackpad"
	runner := &mapDefaults{vals: map[string]string{
		primary + "|Clicking":   "1",
		alternate + "|Clicking": "0",
	}}
	desired := []config.DesiredSetting{
		{
			Domain:  primary,
			Key:     "Clicking",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "trackpad",
		},
		{
			Domain:  alternate,
			Key:     "Clicking",
			Value:   config.SettingValue{Kind: config.SettingBool},
			Section: "trackpad",
		},
	}

	got, _, err := SnapshotCatalogDefaults(context.Background(), runner, desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1; got %#v", len(got), got)
	}
	if got[0].Domain != primary || !got[0].Value.Bool {
		t.Fatalf("entry = %#v", got[0])
	}
}
