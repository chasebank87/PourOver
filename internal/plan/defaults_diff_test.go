package plan

import (
	"strings"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
)

func TestBuildDefaultsPlan(t *testing.T) {
	p := BuildDefaultsPlan([]discovery.SettingStatus{
		{
			Desired: config.DesiredSetting{
				Domain: config.DomainDock,
				Key:    "autohide",
				Value:  config.SettingValue{Kind: config.SettingBool, Bool: true},
			},
			Drift: false,
		},
		{
			Desired: config.DesiredSetting{
				Domain: config.DomainDock,
				Key:    "tilesize",
				Value:  config.SettingValue{Kind: config.SettingInt, Int: 48},
			},
			Drift: true,
		},
	})
	if len(p.Actions) != 1 {
		t.Fatalf("actions=%v", p.Actions)
	}
	a := p.Actions[0]
	if a.Type != ActionDefaultsWrite || a.Domain != config.DomainDock || a.Key != "tilesize" || a.Value != "48" || a.Kind != "int" {
		t.Fatalf("action=%#v", a)
	}
}

func TestBuildDefaultsPlan_PersistentApps(t *testing.T) {
	p := BuildDefaultsPlan([]discovery.SettingStatus{{
		Desired: config.DesiredSetting{
			Domain: config.DomainDock,
			Key:    config.DockPersistentAppsKey,
			Value:  config.SettingValue{Kind: config.SettingArray, Array: []string{"/Applications/Safari.app"}},
		},
		Drift: true,
	}})
	if len(p.Actions) != 1 {
		t.Fatalf("actions=%v", p.Actions)
	}
	a := p.Actions[0]
	if a.Kind != "array" || a.Value != `["/Applications/Safari.app"]` {
		t.Fatalf("action=%#v", a)
	}
}

func TestRenderText_DefaultsWrite(t *testing.T) {
	text := RenderText(Plan{Actions: []Action{{
		Type:   ActionDefaultsWrite,
		Domain: config.DomainDock,
		Key:    "autohide",
		Value:  "true",
		Kind:   "bool",
	}}})
	if !strings.Contains(text, "defaults write com.apple.dock autohide = true") {
		t.Fatalf("text=%q", text)
	}
}
