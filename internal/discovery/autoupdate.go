package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

const (
	// AutoUpdateLabel is the launchd job label used by brew autoupdate.
	AutoUpdateLabel = "com.github.domt4.homebrew-autoupdate"
)

var (
	plistStartIntervalRe = regexp.MustCompile(`(?s)<key>StartInterval</key>\s*<integer>(\d+)</integer>`)
	plistRunAtLoadRe     = regexp.MustCompile(`(?s)<key>RunAtLoad</key>\s*<true\s*/>`)
	plistCalHourRe       = regexp.MustCompile(`(?s)<key>StartCalendarInterval</key>\s*<dict>.*?<key>Hour</key>\s*<integer>(\d+)</integer>`)
	plistCalMinuteRe     = regexp.MustCompile(`(?s)<key>StartCalendarInterval</key>\s*<dict>.*?<key>Minute</key>\s*<integer>(\d+)</integer>`)
)

// AutoupdateState is the current brew autoupdate launch agent.
type AutoupdateState struct {
	Installed bool
	Running   bool
	Interval  config.AutoUpdateInterval
	Upgrade   bool
	Greedy    bool
	Cleanup   bool
	Immediate bool
}

// AutoupdateProbe reads launchd/plist state. Tests inject a fake.
type AutoupdateProbe interface {
	Discover() (AutoupdateState, error)
}

// HostAutoupdateProbe discovers brew autoupdate from the user's LaunchAgents.
type HostAutoupdateProbe struct {
	Home    string                  // override home directory (tests)
	ListJob func(label string) bool // nil uses launchctl when Home is empty
}

// DefaultAutoupdateProbe is the production filesystem/launchctl probe.
var DefaultAutoupdateProbe AutoupdateProbe = HostAutoupdateProbe{}

// StaticAutoupdateProbe returns a fixed state (tests).
type StaticAutoupdateProbe struct {
	State AutoupdateState
	Err   error
}

// Discover implements AutoupdateProbe.
func (p StaticAutoupdateProbe) Discover() (AutoupdateState, error) {
	return p.State, p.Err
}

// Discover implements AutoupdateProbe.
func (p HostAutoupdateProbe) Discover() (AutoupdateState, error) {
	home := p.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return AutoupdateState{}, err
		}
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", AutoUpdateLabel+".plist")
	scriptPath := filepath.Join(home, "Library", "Application Support", AutoUpdateLabel, "brew_autoupdate")

	plist, plistErr := os.ReadFile(plistPath)
	plistOK := plistErr == nil
	if plistErr != nil && !os.IsNotExist(plistErr) {
		return AutoupdateState{}, plistErr
	}
	script, scriptErr := os.ReadFile(scriptPath)
	scriptOK := scriptErr == nil
	if scriptErr != nil && !os.IsNotExist(scriptErr) {
		return AutoupdateState{}, scriptErr
	}

	st := AutoupdateState{
		Installed: plistOK || scriptOK,
		Running:   autoupdateJobLoaded(p),
	}
	if plistOK {
		st.Interval, st.Immediate = parseAutoupdatePlist(plist)
	}
	if scriptOK {
		st.Upgrade, st.Greedy, st.Cleanup = parseAutoupdateScript(script)
	}
	return st, nil
}

func autoupdateJobLoaded(p HostAutoupdateProbe) bool {
	if p.ListJob != nil {
		return p.ListJob(AutoUpdateLabel)
	}
	if p.Home != "" {
		return false
	}
	return launchctlLoaded(AutoUpdateLabel)
}

func launchctlLoaded(label string) bool {
	cmd := exec.Command("launchctl", "list", label)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func parseAutoupdatePlist(data []byte) (config.AutoUpdateInterval, bool) {
	immediate := plistRunAtLoadRe.Match(data)
	if m := plistCalHourRe.FindSubmatch(data); m != nil {
		hour, _ := strconv.Atoi(string(m[1]))
		minute := 0
		if mm := plistCalMinuteRe.FindSubmatch(data); mm != nil {
			minute, _ = strconv.Atoi(string(mm[1]))
		}
		return config.AutoUpdateInterval{Calendar: true, Hour: hour, Minute: minute}, immediate
	}
	if m := plistStartIntervalRe.FindSubmatch(data); m != nil {
		sec, _ := strconv.Atoi(string(m[1]))
		return config.AutoUpdateInterval{Seconds: sec}, immediate
	}
	return config.AutoUpdateInterval{Seconds: config.DefaultAutoUpdateSeconds}, immediate
}

func parseAutoupdateScript(data []byte) (upgrade, greedy, cleanup bool) {
	s := string(data)
	upgrade = strings.Contains(s, " upgrade ") || strings.Contains(s, "/brew upgrade")
	greedy = strings.Contains(s, "--greedy")
	cleanup = strings.Contains(s, " cleanup")
	return upgrade, greedy, cleanup
}
