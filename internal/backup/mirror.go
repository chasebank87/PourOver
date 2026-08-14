package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MirrorSnapshot copies a local snapshot directory into iCloudRoot/snapshots/<name>.
// snapshotDir must be .../snapshots/<timestamp>.
func MirrorSnapshot(snapshotDir, iCloudRoot string) (string, error) {
	info, err := os.Stat(snapshotDir)
	if err != nil {
		return "", fmt.Errorf("snapshot source: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("snapshot source %s is not a directory", snapshotDir)
	}

	name := filepath.Base(snapshotDir)
	dest := filepath.Join(iCloudRoot, "snapshots", name)
	if err := copyDir(snapshotDir, dest); err != nil {
		return "", fmt.Errorf("mirror snapshot: %w", err)
	}
	return dest, nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(s, d); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
