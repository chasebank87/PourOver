package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

// MasDiskProbe reports whether a Mac App Store app bundle exists on disk and,
// when Spotlight has indexed it, its App Store Adam ID.
type MasDiskProbe interface {
	// FindApp looks for name.app under /Applications and ~/Applications.
	// exists is true when the bundle path is present. adamID is the App Store
	// ID from mdls when known; 0 means unknown (cold Spotlight index).
	FindApp(name string) (exists bool, adamID int64)
}

// HostMasDiskProbe uses the filesystem and mdls.
type HostMasDiskProbe struct{}

// DefaultMasDiskProbe is used by plan/apply to merge on-disk App Store apps when
// Spotlight lags. Tests may set this to nil to disable disk merge.
var DefaultMasDiskProbe MasDiskProbe = HostMasDiskProbe{}

// FindApp implements MasDiskProbe.
func (HostMasDiskProbe) FindApp(name string) (bool, int64) {
	path, ok := findMasAppBundle(name)
	if !ok {
		return false, 0
	}
	return true, readAppStoreAdamID(path)
}

func findMasAppBundle(name string) (string, bool) {
	candidates := []string{
		filepath.Join("/Applications", name+".app"),
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, filepath.Join(home, "Applications", name+".app"))
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, true
		}
	}
	return "", false
}

func readAppStoreAdamID(appPath string) int64 {
	out, err := exec.Command("mdls", "-name", "kMDItemAppStoreAdamID", "-raw", appPath).Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	if s == "" || s == "(null)" {
		return 0
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// MergeDesiredMasOnDisk adds desired apps that exist on disk but are missing
// from mas list (Spotlight lag after restore). Only desired apps are considered;
// undeclared on-disk apps are never invented (avoids false mas_remove when the
// index is cold). When mdls reports an Adam ID that differs from the desired ID,
// the app is not treated as installed.
func MergeDesiredMasOnDisk(state MasState, desired []config.MasApp, probe MasDiskProbe) MasState {
	if probe == nil || len(desired) == 0 {
		return state
	}
	have := make(map[int64]struct{}, len(state.Apps))
	for _, app := range state.Apps {
		have[app.ID] = struct{}{}
	}
	out := append([]MasInstalled(nil), state.Apps...)
	for _, want := range desired {
		if _, ok := have[want.ID]; ok {
			continue
		}
		exists, adamID := probe.FindApp(want.Name)
		if !exists {
			continue
		}
		if adamID != 0 && adamID != want.ID {
			continue
		}
		out = append(out, MasInstalled{ID: want.ID, Name: want.Name})
		have[want.ID] = struct{}{}
	}
	state.Apps = out
	return state
}

// FormatMasDiskHint is a stable doctor message fragment.
func FormatMasDiskHint(name string, id int64) string {
	return fmt.Sprintf("%s (%d) on disk but not in mas list", name, id)
}
