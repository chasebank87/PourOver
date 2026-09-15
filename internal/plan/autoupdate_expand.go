package plan

import "github.com/chasebank87/PourOver/internal/config"

// ExpandAutoUpdateTap appends the brew autoupdate tap when packages.auto_update
// is managed and the command is needed (enabled, or disabled but still installed).
// Skipped if domt4/autoupdate or homebrew/autoupdate is already desired or tapped.
func ExpandAutoUpdateTap(pkgs config.Packages, currentTaps []string, installed bool) config.Packages {
	if !pkgs.AutoUpdate.Configured {
		return pkgs
	}
	if !pkgs.AutoUpdate.Enable && !installed {
		return pkgs
	}
	if hasAutoUpdateTap(pkgs.TapNames()) || hasAutoUpdateTap(currentTaps) {
		return pkgs
	}
	out := pkgs
	out.Taps = append(append([]config.TapSpec(nil), pkgs.Taps...), config.TapSpec{
		Name:    config.AutoUpdateTap,
		Trusted: true,
	})
	return out
}

func hasAutoUpdateTap(names []string) bool {
	for _, name := range names {
		if config.IsAutoUpdateTap(name) {
			return true
		}
	}
	return false
}
