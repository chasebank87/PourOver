package configimport

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/paths"
)

// FileCandidate is a path on the machine that can be imported as a file link.
type FileCandidate struct {
	TargetPath string // absolute path of the existing file/symlink
	TargetDecl string // declaration form for Lua (e.g. ~/.config/nvim)
	RelSource  string // relative source under config dir (e.g. config/nvim)
}

// DefaultHomeCandidates returns common home/config paths to consider for import.
func DefaultHomeCandidates(home string) []FileCandidate {
	var out []FileCandidate
	dotfiles := []string{".zshrc", ".zprofile", ".bashrc", ".bash_profile", ".gitconfig", ".vimrc", ".tmux.conf"}
	for _, name := range dotfiles {
		out = append(out, FileCandidate{
			TargetPath: filepath.Join(home, name),
			TargetDecl: "~/" + name,
			RelSource:  filepath.ToSlash(filepath.Join("config", "home", strings.TrimPrefix(name, "."))),
		})
	}

	configRoot := filepath.Join(home, ".config")
	entries, err := os.ReadDir(configRoot)
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if SkipImportName(name) {
				continue
			}
			out = append(out, FileCandidate{
				TargetPath: filepath.Join(configRoot, name),
				TargetDecl: "~/.config/" + name,
				RelSource:  filepath.ToSlash(filepath.Join("config", name)),
			})
		}
	}
	return out
}

// SkipImportName reports whether a ~/.config entry name should not be imported
// (Finder metadata, accidental "~" dirs, etc.).
func SkipImportName(name string) bool {
	return name == "~" || paths.SkipFileName(name) || paths.SkipWalkDir(name)
}

// ExistingImportable filters candidates to those that currently exist on disk.
func ExistingImportable(candidates []FileCandidate) ([]FileCandidate, error) {
	var out []FileCandidate
	for _, c := range candidates {
		if _, err := os.Lstat(c.TargetPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// ImportFile copies the candidate into cfgDir and optionally rewrites the live
// path as a regular file/directory tree matching the config copy (not a symlink).
// Returns the FileLink declaration for pourover.lua.
func ImportFile(cfgDir string, c FileCandidate, applyLive bool) (config.FileLink, error) {
	srcAbs := filepath.Join(cfgDir, filepath.FromSlash(c.RelSource))
	if err := os.MkdirAll(filepath.Dir(srcAbs), 0o755); err != nil {
		return config.FileLink{}, err
	}

	info, err := os.Lstat(c.TargetPath)
	if err != nil {
		return config.FileLink{}, err
	}

	var copyFrom string
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(c.TargetPath)
		if err != nil {
			return config.FileLink{}, fmt.Errorf("resolve %s: %w", c.TargetPath, err)
		}
		copyFrom = resolved
	} else {
		copyFrom = c.TargetPath
	}

	// Copy via a temp path first. The live target may already be a symlink into
	// srcAbs from a previous import; deleting srcAbs before copying would remove
	// the only remaining content (stat: no such file or directory).
	tmpDst := srcAbs + ".pourover-import-tmp"
	_ = os.RemoveAll(tmpDst)
	if err := copyPath(copyFrom, tmpDst); err != nil {
		_ = os.RemoveAll(tmpDst)
		return config.FileLink{}, err
	}
	if err := os.RemoveAll(srcAbs); err != nil {
		_ = os.RemoveAll(tmpDst)
		return config.FileLink{}, fmt.Errorf("clear %s: %w", srcAbs, err)
	}
	if err := os.Rename(tmpDst, srcAbs); err != nil {
		// Cross-device rename is rare under the same cfgDir; fall back to copy.
		if err := copyPath(tmpDst, srcAbs); err != nil {
			_ = os.RemoveAll(tmpDst)
			return config.FileLink{}, err
		}
		_ = os.RemoveAll(tmpDst)
	}

	link := config.FileLink{Source: c.RelSource, Target: c.TargetDecl}
	if applyLive {
		if err := retargetAsRegularFile(c.TargetPath, srcAbs); err != nil {
			return link, err
		}
	}
	return link, nil
}

// retargetAsRegularFile replaces targetPath with a copy of sourceAbs (file or tree).
// Used after import so live paths are not live-symlinks into ~/.pourover.
func retargetAsRegularFile(targetPath, sourceAbs string) error {
	if err := os.RemoveAll(targetPath); err != nil {
		return fmt.Errorf("replace %s: %w", targetPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	if err := copyPath(sourceAbs, targetPath); err != nil {
		return fmt.Errorf("copy %s -> %s: %w", sourceAbs, targetPath, err)
	}
	return nil
}

func copyPath(src, dst string) error {
	// Follow symlinks when deciding file vs directory. DirEntry.IsDir() is false for
	// symlinks-to-directories (common with nix / nested configs), which previously
	// made import try to copy those paths as files and fail with "is a directory".
	info, err := os.Stat(src)
	if err != nil {
		li, lerr := os.Lstat(src)
		if lerr == nil && li.Mode()&os.ModeSymlink != 0 {
			target, rerr := os.Readlink(src)
			if rerr != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			return os.Symlink(target, dst)
		}
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	if err := ensureDir(dst); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if paths.SkipWalkDir(e.Name()) || paths.SkipFileName(e.Name()) {
			continue
		}
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if err := copyPath(s, d); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
	}
	return nil
}

func ensureDir(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("cannot copy directory %s as a file", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Replace a leftover wrong-type destination (file where we need a file is fine
	// with O_TRUNC; directory/symlink-to-dir must be removed first).
	if li, err := os.Lstat(dst); err == nil && li.IsDir() {
		if err := os.RemoveAll(dst); err != nil {
			return err
		}
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// FormatRootLua writes pourover.lua with packages require, links, and preserved policy/backup.
func FormatRootLua(links []config.FileLink, policy config.Policy, backup config.Backup) string {
	mode := policy.UninstallMode
	if mode == "" {
		mode = config.UninstallModeSafe
	}
	var b strings.Builder
	b.WriteString("-- Root PourOver config (links may be generated by pourover import).\n")
	b.WriteString("local packages = require(\"packages\")\n\n")
	b.WriteString("return {\n")
	b.WriteString("  packages = packages,\n")
	b.WriteString("  files = {\n")
	b.WriteString("    links = {\n")
	for _, link := range links {
		fmt.Fprintf(&b, "      { source = %q, target = %q },\n", link.Source, link.Target)
	}
	b.WriteString("    },\n")
	b.WriteString("  },\n")
	fmt.Fprintf(&b, "  policy = {\n    uninstall_mode = %q,\n  },\n", mode)
	b.WriteString("  backup = {\n    icloud = {\n")
	fmt.Fprintf(&b, "      enabled = %v,\n", backup.ICloud.Enabled)
	if backup.ICloud.Path != "" {
		fmt.Fprintf(&b, "      path = %q,\n", backup.ICloud.Path)
	}
	b.WriteString("    },\n")
	b.WriteString("    git = {\n")
	fmt.Fprintf(&b, "      enabled = %v,\n", backup.Git.Enabled)
	fmt.Fprintf(&b, "      auto_push = %v,\n", backup.Git.AutoPush)
	if backup.Git.Remote != "" {
		fmt.Fprintf(&b, "      remote = %q,\n", backup.Git.Remote)
	}
	branch := backup.Git.Branch
	if branch == "" {
		branch = "main"
	}
	fmt.Fprintf(&b, "      branch = %q,\n", branch)
	b.WriteString("    },\n  },\n")
	b.WriteString("}\n")
	return b.String()
}
