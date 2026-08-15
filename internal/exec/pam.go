package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/pam"
	"github.com/chasebank87/PourOver/internal/plan"
)

// PAMApplyOptions configures PAM apply with injectable paths for tests.
// Production callers leave SudoLocalPath / SudoPath empty to use
// plan.DefaultPAMSudoLocalPath and plan.DefaultPAMSudoPath.
//
// Writing under /etc requires elevation: ApplyPAM shells out to sudo when the
// target path is under /etc. Unit tests inject temp dirs and never invoke sudo.
// BeforeAuth, when set, runs immediately before sudo so fancy progress UI can
// park the live status line (sudo Password: goes to /dev/tty, not Session.Write).
type PAMApplyOptions struct {
	SudoLocalPath string
	SudoPath      string
	BeforeAuth    func()
}

func (o PAMApplyOptions) sudoLocalPath(actionName string) string {
	if actionName != "" {
		return actionName
	}
	if o.SudoLocalPath != "" {
		return o.SudoLocalPath
	}
	return plan.DefaultPAMSudoLocalPath
}

func (o PAMApplyOptions) sudoPath(actionName string) string {
	if actionName != "" {
		return actionName
	}
	if o.SudoPath != "" {
		return o.SudoPath
	}
	return plan.DefaultPAMSudoPath
}

// ApplyPAM applies pam_sudo_local_write, pam_sudo_local_remove, and
// pam_sudo_include actions. Call after formula installs so pam_*.so modules exist.
// pam_sudo_local_remove is legacy: it writes the disabled stub instead of
// deleting, so `auth include sudo_local` stays safe. New plans use write+stub.
// Returns how many actions succeeded; error is joined failures.
func ApplyPAM(ctx context.Context, p plan.Plan, opts PAMApplyOptions, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionPAMSudoLocalWrite, plan.ActionPAMSudoLocalRemove, plan.ActionPAMSudoInclude:
		default:
			continue
		}
		report(progress, a)
		var err error
		switch a.Type {
		case plan.ActionPAMSudoLocalWrite:
			err = writeSudoLocal(ctx, opts.sudoLocalPath(a.Name), a.Value, opts.BeforeAuth)
		case plan.ActionPAMSudoLocalRemove:
			// Prefer stub over delete: keep include-safe empty managed file.
			err = writeSudoLocal(ctx, opts.sudoLocalPath(a.Name), pam.DisabledSudoLocal, opts.BeforeAuth)
		case plan.ActionPAMSudoInclude:
			err = ensureSudoLocalInclude(ctx, opts.sudoPath(a.Name), opts.BeforeAuth)
		}
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

func writeSudoLocal(ctx context.Context, path, body string, beforeAuth func()) error {
	if err := validatePAMModulePaths(body); err != nil {
		return err
	}
	if err := backupUnmanagedSudoLocal(ctx, path, beforeAuth); err != nil {
		return err
	}
	return writeElevatedFile(ctx, path, []byte(body), 0o644, beforeAuth)
}

func ensureSudoLocalInclude(ctx context.Context, path string, beforeAuth func()) error {
	current, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("pam sudo include read %s: %w", path, err)
	}
	if pam.HasSudoLocalInclude(current) {
		return nil
	}
	updated := insertSudoLocalInclude(current)
	return writeElevatedFile(ctx, path, updated, 0o644, beforeAuth)
}

func insertSudoLocalInclude(content []byte) []byte {
	line := pam.SudoLocalIncludeLine + "\n"
	if len(content) == 0 {
		return []byte(line)
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	insertAt := 0
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			insertAt = i + 1
			continue
		}
		break
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, pam.SudoLocalIncludeLine)
	out = append(out, lines[insertAt:]...)
	return []byte(strings.Join(out, "\n"))
}

func backupUnmanagedSudoLocal(ctx context.Context, path string, beforeAuth func()) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("pam sudo_local backup read %s: %w", path, err)
	}
	if pam.IsPourOverManaged(data) {
		return nil
	}
	bak := path + ".pourover.bak"
	if err := writeElevatedFile(ctx, bak, data, 0o644, beforeAuth); err != nil {
		return fmt.Errorf("pam sudo_local backup %s: %w", bak, err)
	}
	return nil
}

// validatePAMModulePaths ensures auth lines that reference absolute .so modules
// point at existing files, and that optional/sufficient lines include a module.
// System modules like pam_tid.so (no directory) are not checked on disk.
// Versioned suffixes such as pam_watchid.so.2 are included.
func validatePAMModulePaths(body string) error {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "auth" {
			continue
		}
		ctrl := fields[1]
		if ctrl != "optional" && ctrl != "sufficient" && ctrl != "required" && ctrl != "requisite" {
			continue
		}
		if len(fields) < 3 {
			return fmt.Errorf("pam sudo_local write: empty PAM module path in %q", strings.TrimSpace(line))
		}
		mod := fields[2]
		if !strings.Contains(mod, "/") {
			continue // e.g. pam_tid.so
		}
		base := filepath.Base(mod)
		if !strings.Contains(base, ".so") {
			continue
		}
		if _, err := os.Stat(mod); err != nil {
			return fmt.Errorf("pam sudo_local write: PAM module %s not found (install formula first?): %w", mod, err)
		}
	}
	return nil
}

func writeElevatedFile(ctx context.Context, path string, data []byte, mode os.FileMode, beforeAuth func()) error {
	if needsElevation(path) {
		if beforeAuth != nil {
			beforeAuth()
		}
		return elevatedWrite(ctx, path, data, mode)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("pam write %s: parent dir: %w", path, err)
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return fmt.Errorf("pam write %s: %w", path, err)
	}
	return nil
}

func needsElevation(path string) bool {
	clean := filepath.Clean(path)
	return clean == "/etc" || strings.HasPrefix(clean, "/etc/")
}

// elevatedWrite is sudoWriteFile in production; tests may stub it.
var elevatedWrite = sudoWriteFile

func sudoWriteFile(ctx context.Context, path string, data []byte, mode os.FileMode) error {
	args, cleanup, err := prepareSudoInstall(data, mode, path)
	if err != nil {
		return err
	}
	defer cleanup()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("writing %s requires admin privileges: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// prepareSudoInstall stages data in a temp file and returns sudo install argv.
// macOS `install` rejects /dev/stdin ("Inappropriate file type or format"), so
// content must be copied from a regular file: sudo install -m MODE <tmp> <dest>.
func prepareSudoInstall(data []byte, mode os.FileMode, dest string) (args []string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "pourover-pam-*")
	if err != nil {
		return nil, nil, fmt.Errorf("pam stage write: create temp: %w", err)
	}
	tmpName := tmp.Name()
	cleanup = func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return nil, nil, fmt.Errorf("pam stage write: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("pam stage write: close temp: %w", err)
	}

	modeArg := fmt.Sprintf("%04o", mode.Perm())
	args = []string{"sudo", "install", "-m", modeArg, tmpName, dest}
	return args, cleanup, nil
}
