package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chasebank87/PourOver/internal/config"
)

func TestDiscoverAutoupdate_Missing(t *testing.T) {
	t.Parallel()
	probe := HostAutoupdateProbe{Home: t.TempDir()}
	st, err := probe.Discover()
	if err != nil {
		t.Fatal(err)
	}
	if st.Installed || st.Running {
		t.Fatalf("empty home: %#v", st)
	}
}

func TestDiscoverAutoupdate_PlistAndScript(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	plistDir := filepath.Join(home, "Library", "LaunchAgents")
	scriptDir := filepath.Join(home, "Library", "Application Support", AutoUpdateLabel)
	if err := os.MkdirAll(plistDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0"?>
<plist><dict>
  <key>Label</key><string>` + AutoUpdateLabel + `</string>
  <key>StartInterval</key>
  <integer>43200</integer>
  <key>RunAtLoad</key>
  <true/>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(plistDir, AutoUpdateLabel+".plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n/opt/homebrew/bin/brew update && /opt/homebrew/bin/brew upgrade --formula -v && /opt/homebrew/bin/brew upgrade --cask -v --greedy && /opt/homebrew/bin/brew cleanup\n"
	if err := os.WriteFile(filepath.Join(scriptDir, "brew_autoupdate"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	st, err := (HostAutoupdateProbe{Home: home}).Discover()
	if err != nil {
		t.Fatal(err)
	}
	if !st.Installed {
		t.Fatal("Installed = false")
	}
	if st.Interval.Seconds != 43200 || st.Interval.Calendar {
		t.Fatalf("Interval = %#v", st.Interval)
	}
	if !st.Immediate || !st.Upgrade || !st.Greedy || !st.Cleanup {
		t.Fatalf("flags: %#v", st)
	}
}

func TestParseAutoupdatePlist_Calendar(t *testing.T) {
	t.Parallel()
	data := []byte(`<key>StartCalendarInterval</key>
<dict>
<key>Hour</key>
<integer>0</integer>
<key>Minute</key>
<integer>30</integer>
</dict>`)
	iv, immediate := parseAutoupdatePlist(data)
	if !iv.Calendar || iv.Hour != 0 || iv.Minute != 30 {
		t.Fatalf("interval = %#v", iv)
	}
	if immediate {
		t.Fatal("RunAtLoad should be false")
	}
}

func TestParseAutoUpdateIntervalMatchesDesired(t *testing.T) {
	t.Parallel()
	desired := config.AutoUpdate{Configured: true, Enable: true, Interval: "12h", Upgrade: true, Cleanup: true}
	current := config.AutoUpdateInterval{Seconds: 12 * 3600}
	if !desired.MatchesSchedule(current, true, false, true, false) {
		t.Fatal("expected match")
	}
	if desired.MatchesSchedule(current, false, false, true, false) {
		t.Fatal("upgrade mismatch should not match")
	}
}
