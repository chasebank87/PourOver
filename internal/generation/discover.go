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
// Symlinks at the target are Differ so apply can replace them with regular files.
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
	gotHash, err := HashFile(targetPath)
	if err != nil {
		return FileStatus{}, err
	}
	if gotHash == e.Hash {
		return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusSame}, nil
	}
	return FileStatus{Entry: e, TargetPath: targetPath, Kind: FileStatusDiffer}, nil
}
