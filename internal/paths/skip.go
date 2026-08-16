package paths

import "strings"

// SkipFileName reports whether a file or directory basename should never be
// imported or activated (Finder metadata, Windows junk, AppleDouble, bytecode,
// and volatile app caches that churn under linked config trees).
func SkipFileName(name string) bool {
	if strings.HasPrefix(name, "._") {
		return true
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".pyc") || strings.HasSuffix(lower, ".pyo") {
		return true
	}
	if strings.HasSuffix(lower, ".log") {
		return true
	}
	switch lower {
	case ".ds_store", "thumbs.db", "desktop.ini", "statsig-cache.json":
		return true
	default:
		return false
	}
}

// SkipWalkDir reports whether a directory should be skipped during a config tree walk.
func SkipWalkDir(name string) bool {
	if strings.EqualFold(name, ".git") || strings.EqualFold(name, ".svn") || strings.EqualFold(name, "__pycache__") {
		return true
	}
	switch strings.ToLower(name) {
	case "cache", "cacheddata", "code cache", "gpucache", "logs", "crashpad":
		return true
	}
	return SkipFileName(name)
}

// SkipOwnedPath reports whether an absolute owned path should be dropped from
// lock ownership without pruning (volatile / never-managed basenames or dirs).
func SkipOwnedPath(path string) bool {
	if path == "" {
		return false
	}
	// Check each path segment for skip dirs/files.
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i < len(parts)-1 {
			if SkipWalkDir(part) {
				return true
			}
			continue
		}
		if SkipFileName(part) || SkipWalkDir(part) {
			return true
		}
	}
	return false
}

// FilterOwnedPaths drops skipped volatile paths from an owned set.
func FilterOwnedPaths(owned []string) []string {
	if len(owned) == 0 {
		return owned
	}
	out := make([]string, 0, len(owned))
	for _, path := range owned {
		if SkipOwnedPath(path) {
			continue
		}
		out = append(out, path)
	}
	return out
}
