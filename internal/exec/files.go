package exec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/generation"
	"github.com/chasebank87/PourOver/internal/plan"
	tmpl "github.com/chasebank87/PourOver/internal/template"
)

// CreateLink creates a symlink at targetPath pointing to sourcePath.
// Parent directories of targetPath are created when missing.
func CreateLink(targetPath, sourcePath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create link %s: parent dir: %w", targetPath, err)
	}
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		return fmt.Errorf("create link %s -> %s: %w", targetPath, sourcePath, err)
	}
	return nil
}

// UpdateLink replaces an existing symlink at targetPath to point to sourcePath.
// Refuses to replace a non-symlink target.
func UpdateLink(targetPath, sourcePath string) error {
	info, err := os.Lstat(targetPath)
	if err != nil {
		return fmt.Errorf("update link %s: %w", targetPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("update link %s: target exists and is not a symlink", targetPath)
	}
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("update link %s: remove old: %w", targetPath, err)
	}
	return CreateLink(targetPath, sourcePath)
}

// FileApplyOptions configures file link/copy apply with optional backup-on-replace.
type FileApplyOptions struct {
	ConfigDir    string
	StateDir     string
	GenerationID string // when set, link/managed/template payloads come from generation blobs
	FileReplace  config.FileReplaceMode
	Now          func() time.Time
}

func (o FileApplyOptions) clock() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// ApplyFileLinks runs link_create, link_update, and link_replace actions.
// When GenerationID is set, writes regular files from generation blobs (Value = hash).
// Otherwise copies from ConfigDir sources. Never creates symlinks.
// Existing symlinks at the target are replaced with regular files.
// link_replace / backup mode moves unexpected targets aside first.
func ApplyFileLinks(p plan.Plan, opts FileApplyOptions, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		switch a.Type {
		case plan.ActionLinkCreate, plan.ActionLinkUpdate, plan.ActionLinkReplace:
		default:
			continue
		}

		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		data, mode, err := loadFilePayload(a, opts)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		backupFirst := opts.FileReplace == config.FileReplaceBackup ||
			a.Type == plan.ActionLinkReplace ||
			a.Kind == "backup"
		if err := writeManagedBytes(targetPath, data, mode, ManagedCopyOptions{
			StateDir:    opts.StateDir,
			BackupFirst: backupFirst,
			Now:         opts.clock(),
		}, "link activate"); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

func loadFilePayload(a plan.Action, opts FileApplyOptions) ([]byte, os.FileMode, error) {
	mode := parseFileMode(a.Kind)
	if opts.GenerationID != "" && a.Value != "" {
		data, err := generation.ReadBlob(opts.StateDir, opts.GenerationID, a.Value)
		if err != nil {
			return nil, 0, err
		}
		return data, mode, nil
	}
	configDir, err := filepath.Abs(opts.ConfigDir)
	if err != nil {
		return nil, 0, fmt.Errorf("config directory: %w", err)
	}
	sourcePath, err := resolveLinkSource(a.Source, configDir)
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, 0, fmt.Errorf("read source %s: %w", a.Source, err)
	}
	if info, err := os.Stat(sourcePath); err == nil {
		mode = info.Mode().Perm()
	}
	return data, mode, nil
}

func parseFileMode(kind string) os.FileMode {
	if kind == "" || kind == "backup" {
		return 0o644
	}
	var m uint32
	if _, err := fmt.Sscanf(kind, "%o", &m); err != nil {
		return 0o644
	}
	return os.FileMode(m).Perm()
}

// BackupAside moves path aside under stateDir/backups/files/<timestamp>/<escaped-path>.
// Returns the destination backup path.
func BackupAside(stateDir, path string, at time.Time) (string, error) {
	if strings.TrimSpace(stateDir) == "" {
		return "", fmt.Errorf("backup: state directory required")
	}
	stamp := at.UTC().Format("20060102T150405Z")
	destDir := filepath.Join(stateDir, "backups", "files", stamp)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("backup: create dir: %w", err)
	}
	dest := filepath.Join(destDir, EscapeBackupPath(path))
	if err := os.Rename(path, dest); err != nil {
		return "", fmt.Errorf("backup move %s -> %s: %w", path, dest, err)
	}
	return dest, nil
}

// EscapeBackupPath turns an absolute path into a single path segment for backup storage.
func EscapeBackupPath(path string) string {
	cleaned := filepath.Clean(path)
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		switch r {
		case '/', '\\', ':':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" || out == "." {
		return "_root"
	}
	return out
}

func resolveLinkSource(source, configDir string) (string, error) {
	expanded, err := expandHomePath(source)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(configDir, expanded)), nil
}

func resolveLinkTarget(target string) (string, error) {
	expanded, err := expandHomePath(target)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("target %q must be absolute or start with ~", target)
	}
	return filepath.Clean(expanded), nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// ApplyManagedCopies runs managed_copy actions with atomic writes.
// When GenerationID is set, payloads come from generation blobs (Value = hash).
// Otherwise sources are resolved relative to configDir. Targets expand ~.
// When FileReplace is backup, unexpected target types (e.g. directories) are moved aside first.
// Per-file failures are collected so later copies still run.
func ApplyManagedCopies(p plan.Plan, opts FileApplyOptions, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionManagedCopy {
			continue
		}
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		data, mode, err := loadFilePayload(a, opts)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		backupFirst := opts.FileReplace == config.FileReplaceBackup || a.Kind == "backup"
		if err := writeManagedBytes(targetPath, data, mode, ManagedCopyOptions{
			StateDir:    opts.StateDir,
			BackupFirst: backupFirst,
			Now:         opts.clock(),
		}, "managed copy"); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// ManagedCopyOptions controls backup-before-write for unexpected targets.
type ManagedCopyOptions struct {
	StateDir    string
	BackupFirst bool
	Now         time.Time
}

// ManagedCopy copies sourcePath to targetPath atomically (temp file + rename).
// Creates parent directories when missing. If target is a symlink, removes it
// first so the result is a regular file. Refuses directory targets unless
// BackupFirst is set (moves the directory aside under StateDir, then writes).
func ManagedCopy(sourcePath, targetPath string, opts ...ManagedCopyOptions) error {
	var o ManagedCopyOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("managed copy %s: read source: %w", targetPath, err)
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(sourcePath); err == nil {
		mode = info.Mode().Perm()
	}
	return writeManagedBytes(targetPath, data, mode, o, "managed copy")
}

// ApplyTemplateWrites runs template_write actions.
// When GenerationID is set, writes the pre-rendered generation blob (Value = hash).
// Otherwise re-renders from ConfigDir sources at apply time.
// Soft-fails per file.
func ApplyTemplateWrites(p plan.Plan, opts FileApplyOptions, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionTemplateWrite {
			continue
		}
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}

		var data []byte
		var mode os.FileMode = 0o644
		if opts.GenerationID != "" && a.Value != "" {
			data, mode, err = loadFilePayload(a, opts)
			if err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
		} else {
			configDir, err := filepath.Abs(opts.ConfigDir)
			if err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			sourcePath, err := resolveLinkSource(a.Source, configDir)
			if err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			ctx, err := tmpl.DefaultContext()
			if err != nil {
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			src, err := os.ReadFile(sourcePath)
			if err != nil {
				err = fmt.Errorf("template write %s: read source: %w", targetPath, err)
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			rendered, err := tmpl.Render(string(src), ctx)
			if err != nil {
				err = fmt.Errorf("template write %s: %w", targetPath, err)
				reportFailure(progress, err)
				errs = append(errs, err)
				continue
			}
			data = []byte(rendered)
		}

		backupFirst := opts.FileReplace == config.FileReplaceBackup || a.Kind == "backup"
		if err := writeManagedBytes(targetPath, data, mode, ManagedCopyOptions{
			StateDir:    opts.StateDir,
			BackupFirst: backupFirst,
			Now:         opts.clock(),
		}, "template write"); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// writeManagedBytes prepares the target (backup/remove as needed) and writes
// data atomically via temp file + rename. op is used in error messages
// (e.g. "managed copy" or "template write").
func writeManagedBytes(targetPath string, data []byte, mode os.FileMode, o ManagedCopyOptions, op string) error {
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	if mode == 0 {
		mode = 0o644
	}

	// Replace directory-symlink ancestors with real dirs before MkdirAll, otherwise
	// writes would follow ~/.config/nvim -> ~/.pourover/config/nvim (still live).
	if err := materializeSymlinkAncestors(targetPath, o, op); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("%s %s: parent dir: %w", op, targetPath, err)
	}

	if info, err := os.Lstat(targetPath); err == nil {
		if info.IsDir() {
			if !o.BackupFirst {
				return fmt.Errorf("%s %s: target is a directory", op, targetPath)
			}
			if _, err := BackupAside(o.StateDir, targetPath, o.Now); err != nil {
				return fmt.Errorf("%s %s: %w", op, targetPath, err)
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			if o.BackupFirst {
				if _, err := BackupAside(o.StateDir, targetPath, o.Now); err != nil {
					return fmt.Errorf("%s %s: %w", op, targetPath, err)
				}
			} else if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("%s %s: remove symlink: %w", op, targetPath, err)
			}
		} else if o.BackupFirst {
			if _, err := BackupAside(o.StateDir, targetPath, o.Now); err != nil {
				return fmt.Errorf("%s %s: %w", op, targetPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("%s %s: %w", op, targetPath, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".pourover-managed-*")
	if err != nil {
		return fmt.Errorf("%s %s: create temp: %w", op, targetPath, err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s %s: chmod temp: %w", op, targetPath, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s %s: write temp: %w", op, targetPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s %s: close temp: %w", op, targetPath, err)
	}
	if err := os.Rename(tmpName, targetPath); err != nil {
		return fmt.Errorf("%s %s: rename: %w", op, targetPath, err)
	}
	cleanup = false
	return nil
}

// materializeSymlinkAncestors replaces symlink directories on the path to target
// with real directories so subsequent writes do not follow into the config tree.
// The leaf itself is left alone (handled by the caller). Skips OS volume aliases.
func materializeSymlinkAncestors(targetPath string, o ManagedCopyOptions, op string) error {
	targetPath = filepath.Clean(targetPath)
	var links []string
	p := filepath.Dir(targetPath)
	for p != "" && p != "." && !stopAncestorWalk(p) {
		info, err := os.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(p)
				if parent == p {
					break
				}
				p = parent
				continue
			}
			return fmt.Errorf("%s %s: %w", op, targetPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			links = append(links, p)
		}
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	// Root-most first so we replace ~/.config/nvim before nested lookups.
	for i := len(links) - 1; i >= 0; i-- {
		link := links[i]
		if o.BackupFirst {
			if _, err := BackupAside(o.StateDir, link, o.Now); err != nil {
				return fmt.Errorf("%s %s: materialize %s: %w", op, targetPath, link, err)
			}
		} else if err := os.Remove(link); err != nil {
			return fmt.Errorf("%s %s: remove symlink ancestor %s: %w", op, targetPath, link, err)
		}
		if err := os.MkdirAll(link, 0o755); err != nil {
			return fmt.Errorf("%s %s: mkdir %s: %w", op, targetPath, link, err)
		}
	}
	return nil
}

// stopAncestorWalk mirrors generation.stopAncestorWalk: do not materialize
// macOS volume aliases such as /var → /private/var.
func stopAncestorWalk(path string) bool {
	switch filepath.Clean(path) {
	case "/", ".",
		"/var", "/tmp", "/etc", "/private", "/home", "/Volumes",
		"/System", "/Library", "/Applications", "/Users", "/opt", "/usr":
		return true
	default:
		return false
	}
}

// ApplyFileUnlinks runs file_unlink actions with apply-time safety checks.
// Paths expand ~. Symlinks and regular files are removed; directories are refused.
// Per-path failures are collected so later unlinks still run.
func ApplyFileUnlinks(p plan.Plan, progress Progress) (int, error) {
	n := 0
	var errs []error
	for _, a := range p.Actions {
		if a.Type != plan.ActionFileUnlink {
			continue
		}
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		if err := SafeUnlink(targetPath); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		n++
	}
	return n, errors.Join(errs...)
}

// SafeUnlink removes a symlink or regular file at path. Refuses directories.
// Missing paths are a no-op success (already gone).
func SafeUnlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("unlink %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("unlink %s: path is a directory (refusing unlink)", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("unlink %s: %w", path, err)
	}
	return nil
}

// ConfirmPrunes asks whether to proceed with removing PourOver-owned undeclared files.
// paths are target paths only (no action type prefix).
type ConfirmPrunes func(paths []string) bool

// ApplyFilePrunes runs file_prune actions according to files_mode.
//
//   - non_destructive: skip all prunes (no prompt); plan should normally be empty
//   - strict: prune without prompting
//   - safe: prompt once via confirm; if declined, skip all prunes
//
// confirm may be nil when mode is not safe (or when there are no prunes).
// Returns absolute paths successfully removed. Per-path failures are collected
// so later prunes still run; failed paths are omitted from the returned list.
func ApplyFilePrunes(p plan.Plan, mode config.FilesMode, confirm ConfirmPrunes, progress Progress) ([]string, error) {
	var prunes []plan.Action
	for _, a := range p.Actions {
		if a.Type == plan.ActionFilePrune {
			prunes = append(prunes, a)
		}
	}
	if len(prunes) == 0 {
		return nil, nil
	}

	switch mode {
	case config.FilesModeNonDestructive:
		return nil, nil
	case config.FilesModeSafe, "":
		paths := make([]string, len(prunes))
		for i, a := range prunes {
			paths[i] = a.Name
		}
		if confirm == nil || !confirm(paths) {
			return nil, nil
		}
	case config.FilesModeStrict:
		// no prompt
	default:
		paths := make([]string, len(prunes))
		for i, a := range prunes {
			paths[i] = a.Name
		}
		if confirm == nil || !confirm(paths) {
			return nil, nil
		}
	}

	var removed []string
	var errs []error
	for _, a := range prunes {
		report(progress, a)

		targetPath, err := resolveLinkTarget(a.Name)
		if err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		if err := SafeUnlink(targetPath); err != nil {
			reportFailure(progress, err)
			errs = append(errs, err)
			continue
		}
		removed = append(removed, targetPath)
	}
	return removed, errors.Join(errs...)
}
