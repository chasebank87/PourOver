package generation

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileStatusKind describes a generation file entry vs the live path.
type FileStatusKind string

const (
	FileStatusMissing FileStatusKind = "missing"
	FileStatusSame    FileStatusKind = "same"
	FileStatusDiffer  FileStatusKind = "differ"
	FileStatusBlocked FileStatusKind = "blocked" // unexpected target type (e.g. directory)
)

// FileStatus is the discovered state of one generation file entry.
type FileStatus struct {
	Entry      FileEntry
	TargetPath string
	Kind       FileStatusKind
}

// DiscoverFiles compares each generation file blob hash to the live target.
// Symlinks at the target, or any symlink ancestor (directory-link roots), are
// Differ so apply can replace live symlink trees with regular files.
func DiscoverFiles(entries []FileEntry) ([]FileStatus, error) {
	out := make([]FileStatus, 0, len(entries))
	for i, e := range entries {
		st, err := inspectFile(e)
		if err != nil {
			return nil, fmt.Errorf("generation files[%d] %s: %w", i+1, e.Target, err)
		}
		out = append(out, st)
	}
	return out, nil
}

func inspectFile(e FileEntry) (FileStatus, error) {
	targetPath := filepath.Clean(e.Target)
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Path may be missing because a parent dir symlink was not traversed
			// the same way; still check ancestors for live directory links.
			if through, err := pathHasSymlinkAncestor(targetPath); err != nil {
				return FileStatus{}, err
			} else if through {
				return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusDiffer}, nil
			}
			return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusMissing}, nil
		}
		return FileStatus{}, err
	}
	if info.IsDir() {
		return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusBlocked}, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusDiffer}, nil
	}
	// Nested path under a directory symlink (e.g. ~/.config/nvim -> config/nvim):
	// Lstat sees a regular file whose bytes match the blob, but the path is still live.
	if through, err := pathHasSymlinkAncestor(targetPath); err != nil {
		return FileStatus{}, err
	} else if through {
		return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusDiffer}, nil
	}
	gotHash, err := HashFile(targetPath)
	if err != nil {
		return FileStatus{}, err
	}
	if gotHash == e.Hash {
		return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusSame}, nil
	}
	return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusDiffer}, nil
}

// pathHasSymlinkAncestor reports whether any path component of path is a symlink.
// The leaf itself is included (so a file symlink returns true). Walks stop before
// OS volume aliases such as /var → /private/var.
func pathHasSymlinkAncestor(path string) (bool, error) {
	path = filepath.Clean(path)
	for path != "" && path != "." && !stopAncestorWalk(path) {
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(path)
				if parent == path {
					return false, nil
				}
				path = parent
				continue
			}
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true, nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false, nil
		}
		path = parent
	}
	return false, nil
}

// stopAncestorWalk reports paths we must not treat as materializable symlink
// ancestors (macOS firmlinks / volume aliases).
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
