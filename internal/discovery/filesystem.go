package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chasebank87/PourOver/internal/config"
)

// DiscoverFileLinks inspects each declared link against the filesystem.
// Sources are resolved relative to configDir when not absolute; targets expand ~.
func DiscoverFileLinks(links []config.FileLink, configDir string) ([]FileLinkStatus, error) {
	configDir, err := filepath.Abs(configDir)
	if err != nil {
		return nil, fmt.Errorf("config directory: %w", err)
	}

	statuses := make([]FileLinkStatus, 0, len(links))
	for i, link := range links {
		st, err := inspectFileLink(link, configDir)
		if err != nil {
			return nil, fmt.Errorf("files.links[%d]: %w", i+1, err)
		}
		statuses = append(statuses, st)
	}
	return statuses, nil
}

func inspectFileLink(link config.FileLink, configDir string) (FileLinkStatus, error) {
	sourcePath, err := resolveSourcePath(link.Source, configDir)
	if err != nil {
		return FileLinkStatus{}, err
	}
	targetPath, err := resolveTargetPath(link.Target)
	if err != nil {
		return FileLinkStatus{}, err
	}

	if _, err := os.Stat(sourcePath); err != nil {
		if os.IsNotExist(err) {
			return FileLinkStatus{}, fmt.Errorf("source %q does not exist", link.Source)
		}
		return FileLinkStatus{}, fmt.Errorf("source %q: %w", link.Source, err)
	}

	sourceCanonical, err := filepath.EvalSymlinks(sourcePath)
	if err != nil {
		return FileLinkStatus{}, fmt.Errorf("source %q: %w", link.Source, err)
	}

	kind, actual, err := classifyTarget(targetPath, sourceCanonical)
	if err != nil {
		return FileLinkStatus{}, err
	}

	return FileLinkStatus{
		Link:         link,
		SourcePath:   sourceCanonical,
		TargetPath:   targetPath,
		Kind:         kind,
		ActualTarget: actual,
	}, nil
}

func classifyTarget(targetPath, wantSource string) (LinkStatusKind, string, error) {
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return LinkStatusMissing, "", nil
		}
		return "", "", err
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return LinkStatusBlocked, "", nil
	}

	got, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return LinkStatusWrong, "", nil
	}

	if filepath.Clean(got) == filepath.Clean(wantSource) {
		return LinkStatusCorrect, got, nil
	}
	return LinkStatusWrong, got, nil
}

func resolveSourcePath(source, configDir string) (string, error) {
	expanded, err := expandHome(source)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	return filepath.Clean(filepath.Join(configDir, expanded)), nil
}

func resolveTargetPath(target string) (string, error) {
	expanded, err := expandHome(target)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(expanded) {
		return "", fmt.Errorf("target %q must be absolute or start with ~", target)
	}
	return filepath.Clean(expanded), nil
}

func expandHome(path string) (string, error) {
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
