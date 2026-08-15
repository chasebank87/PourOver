package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DockPersistentAppsKey / DockPersistentOthersKey are com.apple.dock array keys.
const (
	DockPersistentAppsKey   = "persistent-apps"
	DockPersistentOthersKey = "persistent-others"
)

// EncodeDockPersistentPlist builds an XML plist for dock persistent-apps or
// persistent-others, matching nix-darwin's tile encoding.
func EncodeDockPersistentPlist(key string, paths []string) (string, error) {
	expanded := make([]string, 0, len(paths))
	for i, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			return "", fmt.Errorf("%s[%d]: path must not be empty", key, i+1)
		}
		abs, err := expandHomePath(p)
		if err != nil {
			return "", err
		}
		expanded = append(expanded, abs)
	}
	var b strings.Builder
	b.WriteString(plistHeader)
	b.WriteString("<array>\n")
	switch key {
	case DockPersistentAppsKey:
		for _, p := range expanded {
			writeAppTile(&b, p)
		}
	case DockPersistentOthersKey:
		for _, p := range expanded {
			if looksLikeFilePath(p) {
				writeFileTile(&b, p)
			} else {
				writeFolderTile(&b, p)
			}
		}
	default:
		return "", fmt.Errorf("unsupported dock array key %q", key)
	}
	b.WriteString("</array>\n")
	b.WriteString(plistFooter)
	return b.String(), nil
}

// ExtractDockPersistentPaths pulls ordered _CFURLString values from `defaults read`
// output for persistent-apps / persistent-others.
func ExtractDockPersistentPaths(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := splitOpenStepAssignment(line)
		if !ok {
			continue
		}
		if key != "_CFURLString" {
			continue
		}
		if val == "" {
			continue
		}
		out = append(out, normalizeDockURL(val))
	}
	return out
}

func splitOpenStepAssignment(line string) (key, val string, ok bool) {
	eq := strings.Index(line, "=")
	if eq < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:eq])
	key = strings.Trim(key, `"`)
	val = strings.TrimSpace(line[eq+1:])
	val = strings.TrimSuffix(val, ";")
	val = strings.TrimSpace(val)
	val = strings.Trim(val, `"`)
	return key, val, true
}

func normalizeDockURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "file://")
	s = strings.TrimRight(s, "/")
	return s
}

func looksLikeFilePath(p string) bool {
	base := filepath.Base(p)
	return strings.Contains(base, ".")
}

func writeAppTile(b *strings.Builder, path string) {
	b.WriteString("\t<dict>\n")
	b.WriteString("\t\t<key>tile-data</key>\n")
	b.WriteString("\t\t<dict>\n")
	b.WriteString("\t\t\t<key>file-data</key>\n")
	b.WriteString("\t\t\t<dict>\n")
	b.WriteString("\t\t\t\t<key>_CFURLString</key>\n")
	fmt.Fprintf(b, "\t\t\t\t<string>%s</string>\n", xmlEscape(path))
	b.WriteString("\t\t\t\t<key>_CFURLStringType</key>\n")
	b.WriteString("\t\t\t\t<integer>0</integer>\n")
	b.WriteString("\t\t\t</dict>\n")
	b.WriteString("\t\t</dict>\n")
	b.WriteString("\t</dict>\n")
}

func writeFolderTile(b *strings.Builder, path string) {
	url := "file://" + path
	b.WriteString("\t<dict>\n")
	b.WriteString("\t\t<key>tile-data</key>\n")
	b.WriteString("\t\t<dict>\n")
	b.WriteString("\t\t\t<key>file-data</key>\n")
	b.WriteString("\t\t\t<dict>\n")
	b.WriteString("\t\t\t\t<key>_CFURLString</key>\n")
	fmt.Fprintf(b, "\t\t\t\t<string>%s</string>\n", xmlEscape(url))
	b.WriteString("\t\t\t\t<key>_CFURLStringType</key>\n")
	b.WriteString("\t\t\t\t<integer>15</integer>\n")
	b.WriteString("\t\t\t</dict>\n")
	b.WriteString("\t\t\t<key>arrangement</key>\n")
	b.WriteString("\t\t\t<integer>1</integer>\n")
	b.WriteString("\t\t\t<key>displayas</key>\n")
	b.WriteString("\t\t\t<integer>0</integer>\n")
	b.WriteString("\t\t\t<key>showas</key>\n")
	b.WriteString("\t\t\t<integer>0</integer>\n")
	b.WriteString("\t\t</dict>\n")
	b.WriteString("\t\t<key>tile-type</key>\n")
	b.WriteString("\t\t<string>directory-tile</string>\n")
	b.WriteString("\t</dict>\n")
}

func writeFileTile(b *strings.Builder, path string) {
	url := "file://" + path
	b.WriteString("\t<dict>\n")
	b.WriteString("\t\t<key>tile-data</key>\n")
	b.WriteString("\t\t<dict>\n")
	b.WriteString("\t\t\t<key>file-data</key>\n")
	b.WriteString("\t\t\t<dict>\n")
	b.WriteString("\t\t\t\t<key>_CFURLString</key>\n")
	fmt.Fprintf(b, "\t\t\t\t<string>%s</string>\n", xmlEscape(url))
	b.WriteString("\t\t\t\t<key>_CFURLStringType</key>\n")
	b.WriteString("\t\t\t\t<integer>15</integer>\n")
	b.WriteString("\t\t\t</dict>\n")
	b.WriteString("\t\t</dict>\n")
	b.WriteString("\t\t<key>tile-type</key>\n")
	b.WriteString("\t\t<string>file-tile</string>\n")
	b.WriteString("\t</dict>\n")
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		return osUserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := osUserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

// Overridable for tests.
var osUserHomeDir = os.UserHomeDir

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

const plistHeader = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`

const plistFooter = `</plist>
`
