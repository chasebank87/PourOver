package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	// AutoUpdateTap is the tap that provides `brew autoupdate`.
	AutoUpdateTap = "domt4/autoupdate"
	// AutoUpdateAltTap is the older Homebrew-org tap name still found on some Macs.
	AutoUpdateAltTap = "homebrew/autoupdate"
	// DefaultAutoUpdateSeconds is brew autoupdate's default interval (24 hours).
	DefaultAutoUpdateSeconds = 86400
)

var (
	autoUpdateCalendarRe = regexp.MustCompile(`\A([01]?\d|2[0-3]):([0-5]\d)\z`)
	autoUpdateDurationRe = regexp.MustCompile(`(?i)\A(\d+)([smhdw]?)\z`)
)

// AutoUpdateInterval is a parsed brew autoupdate schedule.
type AutoUpdateInterval struct {
	Calendar bool
	Seconds  int
	Hour     int
	Minute   int
}

// ParseAutoUpdateInterval parses brew autoupdate interval syntax.
// Empty input is the 24-hour default. Accepts seconds, 30m/12h/1d/1w, or HH:MM.
func ParseAutoUpdateInterval(s string) (AutoUpdateInterval, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return AutoUpdateInterval{Seconds: DefaultAutoUpdateSeconds}, nil
	}
	if m := autoUpdateCalendarRe.FindStringSubmatch(s); m != nil {
		hour, _ := strconv.Atoi(m[1])
		minute, _ := strconv.Atoi(m[2])
		return AutoUpdateInterval{Calendar: true, Hour: hour, Minute: minute}, nil
	}
	m := autoUpdateDurationRe.FindStringSubmatch(s)
	if m == nil {
		return AutoUpdateInterval{}, fmt.Errorf("%s", autoUpdateIntervalHint)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return AutoUpdateInterval{}, fmt.Errorf("%s", autoUpdateIntervalHint)
	}
	mult := 1
	switch strings.ToLower(m[2]) {
	case "m":
		mult = 60
	case "h":
		mult = 60 * 60
	case "d":
		mult = 24 * 60 * 60
	case "w":
		mult = 7 * 24 * 60 * 60
	}
	return AutoUpdateInterval{Seconds: n * mult}, nil
}

const autoUpdateIntervalHint = "must be positive seconds, a duration such as 30m, 12h, or 1d, or a 24-hour clock time such as 00:00"

// Equal reports whether two parsed intervals represent the same schedule.
func (i AutoUpdateInterval) Equal(other AutoUpdateInterval) bool {
	if i.Calendar != other.Calendar {
		return false
	}
	if i.Calendar {
		return i.Hour == other.Hour && i.Minute == other.Minute
	}
	return i.Seconds == other.Seconds
}

// Display returns a stable label for plan text (empty/default → "24h").
func (a AutoUpdate) IntervalDisplay() string {
	s := strings.TrimSpace(a.Interval)
	if s == "" {
		return "24h"
	}
	return s
}

// StartArgs returns `brew autoupdate start` arguments (without the brew binary).
func (a AutoUpdate) StartArgs() []string {
	args := []string{"autoupdate", "start"}
	if s := strings.TrimSpace(a.Interval); s != "" {
		args = append(args, s)
	}
	if a.Upgrade {
		args = append(args, "--upgrade")
	}
	if a.Greedy {
		args = append(args, "--greedy")
	}
	if a.Cleanup {
		args = append(args, "--cleanup")
	}
	if a.Immediate {
		args = append(args, "--immediate")
	}
	return args
}

// MatchesSchedule reports whether current launch-agent settings match desired.
func (a AutoUpdate) MatchesSchedule(interval AutoUpdateInterval, upgrade, greedy, cleanup, immediate bool) bool {
	want, err := ParseAutoUpdateInterval(a.Interval)
	if err != nil {
		return false
	}
	return want.Equal(interval) &&
		a.Upgrade == upgrade &&
		a.Greedy == greedy &&
		a.Cleanup == cleanup &&
		a.Immediate == immediate
}

// IsAutoUpdateTap reports whether name is the autoupdate external-command tap.
func IsAutoUpdateTap(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case AutoUpdateTap, AutoUpdateAltTap:
		return true
	default:
		return false
	}
}
