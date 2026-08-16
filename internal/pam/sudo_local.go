// Package pam generates and recognizes PourOver-managed PAM sudo_local content.
package pam

import (
	"bytes"
	"os"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

// ManagedMarker identifies a sudo_local file owned by PourOver.
const ManagedMarker = "# pourover: managed"

// DisabledComment explains a managed stub when sudo PAM auth is turned off.
const DisabledComment = "# sudo PAM auth is disabled by PourOver; no auth lines"

// DisabledSudoLocal is the managed stub written when sudo_local.enable=false.
// Keeping a file (instead of deleting) makes `auth include sudo_local` in
// /etc/pam.d/sudo safe.
const DisabledSudoLocal = ManagedMarker + "\n" + DisabledComment + "\n"

// RenderSudoLocal returns the desired /etc/pam.d/sudo_local contents.
//
// When cfg.Configured is false, returns empty (unmanaged — no plan actions).
// When Configured and Enable is false, returns DisabledSudoLocal (stub with
// marker, no auth lines) so include stays safe. When Enable is true, content
// is generated from the auth flags in nix-darwin order: optional reattach,
// sufficient pam_tid, sufficient watchid. reattachPath and watchidPath are
// injected full paths to the .so modules (callers resolve separately so unit
// tests stay stubbable).
func RenderSudoLocal(cfg config.SudoLocalPAM, reattachPath, watchidPath string) string {
	if !cfg.Configured {
		return ""
	}
	if !cfg.Enable {
		return DisabledSudoLocal
	}

	var b strings.Builder
	b.WriteString(ManagedMarker)
	b.WriteByte('\n')
	if cfg.Reattach {
		b.WriteString("auth optional ")
		b.WriteString(reattachPath)
		b.WriteByte('\n')
	}
	if cfg.TouchIDAuth {
		b.WriteString("auth sufficient pam_tid.so\n")
	}
	if cfg.WatchIDAuth && watchidPath != "" {
		b.WriteString("auth sufficient ")
		b.WriteString(watchidPath)
		b.WriteByte('\n')
	}
	return b.String()
}

// IsPourOverManaged reports whether content includes the PourOver managed marker.
func IsPourOverManaged(content []byte) bool {
	return bytes.Contains(content, []byte(ManagedMarker))
}

// SudoLocalIncludeLine is the auth line inserted into /etc/pam.d/sudo when missing.
// Spacing matches Apple's stock sudo PAM file.
const SudoLocalIncludeLine = "auth       include        sudo_local"

// HasSudoLocalInclude reports whether sudo PAM content already includes sudo_local.
func HasSudoLocalInclude(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "auth" && fields[1] == "include" && fields[2] == "sudo_local" {
			return true
		}
	}
	return false
}

// ModulePath joins a Homebrew (or other) prefix with the conventional PAM .so
// relative path under lib/pam. prefix may be empty; callers should validate.
func ModulePath(prefix, module string) string {
	prefix = strings.TrimRight(prefix, "/")
	module = strings.TrimLeft(module, "/")
	if prefix == "" {
		return module
	}
	return prefix + "/lib/pam/" + module
}

// DefaultWatchIDSearchPaths returns common locations for pam_watchid.so.
// brewPrefix is the Homebrew root from `brew --prefix` (may be empty).
// pam-watchid is not a Homebrew core formula; installs typically place the
// module under the brew prefix lib/pam or /opt/homebrew|/usr/local/lib/pam.
func DefaultWatchIDSearchPaths(brewPrefix string) []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if brewPrefix != "" {
		add(ModulePath(brewPrefix, "pam_watchid.so"))
		add(ModulePath(brewPrefix, "pam_watchid.so.2"))
	}
	add("/opt/homebrew/lib/pam/pam_watchid.so")
	add("/opt/homebrew/lib/pam/pam_watchid.so.2")
	add("/usr/local/lib/pam/pam_watchid.so")
	add("/usr/local/lib/pam/pam_watchid.so.2")
	return out
}

// FindModule returns the first candidate path that exists on disk.
// candidates is injectable for tests (pass a custom list pointing at temp files).
func FindModule(candidates []string) (string, bool) {
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}
