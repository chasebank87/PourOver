package paths

import "strings"

// SkipFileName reports whether a file or directory basename should never be
// imported or activated (Finder metadata, Windows junk, AppleDouble, bytecode).
func SkipFileName(name string) bool {
	if strings.HasPrefix(name, "._") {
		return true
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".pyc") || strings.HasSuffix(lower, ".pyo") {
		return true
	}
	switch lower {
	case ".ds_store", "thumbs.db", "desktop.ini":
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
	return SkipFileName(name)
}
