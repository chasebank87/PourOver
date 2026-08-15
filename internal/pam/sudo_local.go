// Package pam generates and recognizes PourOver-managed PAM sudo_local content.
package pam

import (
	"bytes"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

// ManagedMarker identifies a sudo_local file owned by PourOver.
const ManagedMarker = "# pourover: managed"

// RenderSudoLocal returns the desired /etc/pam.d/sudo_local contents.
//
// When cfg.Enable is false or cfg.Configured is false, RenderSudoLocal returns
// an empty string so the apply layer can remove a managed file. When Enable is
// true (and Configured), content is generated from the auth flags in
// nix-darwin order: optional reattach, sufficient pam_tid, sufficient watchid.
// reattachPath and watchidPath are injected full paths to the .so modules
// (callers resolve brew prefixes separately so unit tests stay stubbable).
func RenderSudoLocal(cfg config.SudoLocalPAM, reattachPath, watchidPath string) string {
	if !cfg.Configured || !cfg.Enable {
		return ""
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
	if cfg.WatchIDAuth {
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
