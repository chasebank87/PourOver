package paths

import "strings"

// SkipFileName reports whether a file or directory basename should never be
// imported or activated (Finder metadata, Windows junk, AppleDouble).
func SkipFileName(name string) bool {
	if strings.HasPrefix(name, "._") {
		return true
	}
	switch strings.ToLower(name) {
	case ".ds_store", "thumbs.db", "desktop.ini":
		return true
	default:
		return false
	}
}

// SkipWalkDir reports whether a directory should be skipped during a config tree walk.
func SkipWalkDir(name string) bool {
	if strings.EqualFold(name, ".git") || strings.EqualFold(name, ".svn") {
		return true
	}
	return SkipFileName(name)
}
