package config

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// PatchICloudFile sets backup.icloud.enabled (and optional path) in pourover.lua,
// then validates the result loads as a Manifest.
func PatchICloudFile(path string, enabled bool, icloudPath string, setPath bool) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := PatchICloud(string(src), enabled, icloudPath, setPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	if _, err := LoadManifest(path); err != nil {
		return fmt.Errorf("patched config invalid: %w", err)
	}
	return nil
}

// PatchGitFile sets backup.git fields in pourover.lua and validates the result.
func PatchGitFile(path string, git GitBackup) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out, err := PatchGit(string(src), git)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return err
	}
	if _, err := LoadManifest(path); err != nil {
		return fmt.Errorf("patched config invalid: %w", err)
	}
	return nil
}

// PatchICloud surgically updates backup.icloud in Lua source text.
func PatchICloud(src string, enabled bool, icloudPath string, setPath bool) (string, error) {
	out, err := ensureNestedTable(src, []string{"backup", "icloud"})
	if err != nil {
		return "", err
	}
	out, err = setBoolField(out, []string{"backup", "icloud"}, "enabled", enabled)
	if err != nil {
		return "", err
	}
	if setPath {
		if icloudPath == "" {
			out, err = removeStringField(out, []string{"backup", "icloud"}, "path")
		} else {
			out, err = setStringField(out, []string{"backup", "icloud"}, "path", icloudPath)
		}
		if err != nil {
			return "", err
		}
	}
	return out, nil
}

// PatchGit surgically updates backup.git in Lua source text.
func PatchGit(src string, git GitBackup) (string, error) {
	out, err := ensureNestedTable(src, []string{"backup", "git"})
	if err != nil {
		return "", err
	}
	out, err = setBoolField(out, []string{"backup", "git"}, "enabled", git.Enabled)
	if err != nil {
		return "", err
	}
	out, err = setBoolField(out, []string{"backup", "git"}, "auto_push", git.AutoPush)
	if err != nil {
		return "", err
	}
	if git.Remote == "" {
		out, err = removeStringField(out, []string{"backup", "git"}, "remote")
	} else {
		out, err = setStringField(out, []string{"backup", "git"}, "remote", git.Remote)
	}
	if err != nil {
		return "", err
	}
	branch := git.Branch
	if branch == "" {
		branch = "main"
	}
	out, err = setStringField(out, []string{"backup", "git"}, "branch", branch)
	if err != nil {
		return "", err
	}
	return out, nil
}

func ensureNestedTable(src string, path []string) (string, error) {
	if len(path) == 0 {
		return src, nil
	}
	rootStart, rootEnd, err := findReturnTable(src)
	if err != nil {
		return "", err
	}
	curStart, curEnd := rootStart, rootEnd
	prefix := make([]string, 0, len(path))
	for _, key := range path {
		prefix = append(prefix, key)
		bodyStart, bodyEnd, ok := findTableField(src, curStart, curEnd, key)
		if ok {
			curStart, curEnd = bodyStart, bodyEnd
			continue
		}
		insert := formatNestedInsert(prefix[len(prefix)-1:], 1)
		// Insert before the closing brace of the current table.
		src = src[:curEnd] + insert + src[curEnd:]
		bodyStart, bodyEnd, ok = findTableField(src, curStart, curEnd+len(insert), key)
		if !ok {
			return "", fmt.Errorf("failed to insert %s table", strings.Join(prefix, "."))
		}
		curStart, curEnd = bodyStart, bodyEnd
		// Re-find root after mutation for subsequent iterations safety:
		rootStart, rootEnd, err = findReturnTable(src)
		if err != nil {
			return "", err
		}
		curStart, curEnd = rootStart, rootEnd
		for _, k := range prefix {
			bodyStart, bodyEnd, ok = findTableField(src, curStart, curEnd, k)
			if !ok {
				return "", fmt.Errorf("lost table %s after insert", k)
			}
			curStart, curEnd = bodyStart, bodyEnd
		}
	}
	return src, nil
}

func formatNestedInsert(keys []string, indentLevel int) string {
	if len(keys) == 0 {
		return ""
	}
	indent := strings.Repeat("  ", indentLevel)
	var b strings.Builder
	b.WriteByte('\n')
	b.WriteString(indent)
	b.WriteString(keys[0])
	b.WriteString(" = {")
	if len(keys) > 1 {
		b.WriteString(formatNestedInsert(keys[1:], indentLevel+1))
	}
	b.WriteByte('\n')
	b.WriteString(indent)
	b.WriteString("},")
	return b.String()
}

func setBoolField(src string, tablePath []string, field string, value bool) (string, error) {
	start, end, err := locateTable(src, tablePath)
	if err != nil {
		return "", err
	}
	if s, e, ok := findAssign(src, start, end, field); ok {
		lit := "false"
		if value {
			lit = "true"
		}
		return src[:s] + lit + src[e:], nil
	}
	indent := detectIndent(src, start, end)
	lit := "false"
	if value {
		lit = "true"
	}
	insert := fmt.Sprintf("\n%s%s = %s,", indent, field, lit)
	return src[:end] + insert + src[end:], nil
}

func setStringField(src string, tablePath []string, field, value string) (string, error) {
	start, end, err := locateTable(src, tablePath)
	if err != nil {
		return "", err
	}
	quoted := fmt.Sprintf("%q", value)
	if s, e, ok := findAssign(src, start, end, field); ok {
		return src[:s] + quoted + src[e:], nil
	}
	indent := detectIndent(src, start, end)
	insert := fmt.Sprintf("\n%s%s = %s,", indent, field, quoted)
	return src[:end] + insert + src[end:], nil
}

func removeStringField(src string, tablePath []string, field string) (string, error) {
	start, end, err := locateTable(src, tablePath)
	if err != nil {
		return "", err
	}
	lineStart, lineEnd, ok := findAssignLine(src, start, end, field)
	if !ok {
		return src, nil
	}
	return src[:lineStart] + src[lineEnd:], nil
}

func locateTable(src string, tablePath []string) (start, end int, err error) {
	rootStart, rootEnd, err := findReturnTable(src)
	if err != nil {
		return 0, 0, err
	}
	start, end = rootStart, rootEnd
	for _, key := range tablePath {
		bodyStart, bodyEnd, ok := findTableField(src, start, end, key)
		if !ok {
			return 0, 0, fmt.Errorf("missing table %s", strings.Join(tablePath, "."))
		}
		start, end = bodyStart, bodyEnd
	}
	return start, end, nil
}

func findReturnTable(src string) (bodyStart, bodyEnd int, err error) {
	// Prefer `return {` then fall back to first top-level `{`.
	idx := strings.Index(src, "return")
	for idx >= 0 {
		rest := src[idx+len("return"):]
		j := 0
		for j < len(rest) && unicode.IsSpace(rune(rest[j])) {
			j++
		}
		if j < len(rest) && rest[j] == '{' {
			open := idx + len("return") + j
			closeIdx, ok := matchingBrace(src, open)
			if !ok {
				return 0, 0, fmt.Errorf("unbalanced braces in return table")
			}
			return open + 1, closeIdx, nil
		}
		next := strings.Index(src[idx+1:], "return")
		if next < 0 {
			break
		}
		idx = idx + 1 + next
	}
	open := strings.IndexByte(src, '{')
	if open < 0 {
		return 0, 0, fmt.Errorf("no table found in config")
	}
	closeIdx, ok := matchingBrace(src, open)
	if !ok {
		return 0, 0, fmt.Errorf("unbalanced braces in config")
	}
	return open + 1, closeIdx, nil
}

func findTableField(src string, start, end int, key string) (bodyStart, bodyEnd int, ok bool) {
	i := start
	for i < end {
		i = skipSpaceAndComments(src, i, end)
		if i >= end {
			break
		}
		name, next, found := readIdent(src, i, end)
		if !found {
			i++
			continue
		}
		i = skipSpaceAndComments(src, next, end)
		if i >= end || src[i] != '=' {
			continue
		}
		i = skipSpaceAndComments(src, i+1, end)
		if i >= end {
			break
		}
		if name != key {
			if src[i] == '{' {
				if closeIdx, mok := matchingBrace(src, i); mok {
					i = closeIdx + 1
					continue
				}
			}
			// skip value until comma or end at this depth
			i = skipValue(src, i, end)
			continue
		}
		if src[i] != '{' {
			return 0, 0, false
		}
		closeIdx, mok := matchingBrace(src, i)
		if !mok {
			return 0, 0, false
		}
		return i + 1, closeIdx, true
	}
	return 0, 0, false
}

func findAssign(src string, start, end int, field string) (valStart, valEnd int, ok bool) {
	i := start
	for i < end {
		i = skipSpaceAndComments(src, i, end)
		if i >= end {
			break
		}
		name, next, found := readIdent(src, i, end)
		if !found {
			i++
			continue
		}
		i = skipSpaceAndComments(src, next, end)
		if i >= end || src[i] != '=' {
			continue
		}
		i = skipSpaceAndComments(src, i+1, end)
		if name != field {
			i = skipValue(src, i, end)
			continue
		}
		valStart = i
		valEnd = skipValue(src, i, end)
		// trim trailing spaces from value end for clean replace
		for valEnd > valStart && unicode.IsSpace(rune(src[valEnd-1])) {
			valEnd--
		}
		if valEnd > valStart && src[valEnd-1] == ',' {
			valEnd--
		}
		for valEnd > valStart && unicode.IsSpace(rune(src[valEnd-1])) {
			valEnd--
		}
		return valStart, valEnd, true
	}
	return 0, 0, false
}

func findAssignLine(src string, start, end int, field string) (lineStart, lineEnd int, ok bool) {
	valStart, valEnd, ok := findAssign(src, start, end, field)
	if !ok {
		return 0, 0, false
	}
	lineStart = valStart
	for lineStart > start && src[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd = valEnd
	for lineEnd < end && src[lineEnd] != '\n' {
		lineEnd++
	}
	if lineEnd < len(src) && src[lineEnd] == '\n' {
		lineEnd++
	}
	return lineStart, lineEnd, true
}

func detectIndent(src string, start, end int) string {
	// Prefer indent of an existing field line; else two spaces deeper than parent.
	i := start
	for i < end {
		if src[i] == '\n' {
			j := i + 1
			spaces := 0
			for j < end && (src[j] == ' ' || src[j] == '\t') {
				spaces++
				j++
			}
			if j < end && (isIdentStart(src[j]) || src[j] == '[') {
				return src[i+1 : i+1+spaces]
			}
		}
		i++
	}
	return "      "
}

func matchingBrace(src string, open int) (int, bool) {
	if open >= len(src) || src[open] != '{' {
		return 0, false
	}
	depth := 0
	inStr := false
	var strDelim byte
	escape := false
	for i := open; i < len(src); i++ {
		c := src[i]
		if inStr {
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == strDelim {
				inStr = false
			}
			continue
		}
		switch c {
		case '"', '\'':
			inStr = true
			strDelim = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func skipSpaceAndComments(src string, i, end int) int {
	for i < end {
		if unicode.IsSpace(rune(src[i])) {
			i++
			continue
		}
		if src[i] == '-' && i+1 < end && src[i+1] == '-' {
			i += 2
			for i < end && src[i] != '\n' {
				i++
			}
			continue
		}
		break
	}
	return i
}

func readIdent(src string, i, end int) (name string, next int, ok bool) {
	if i < end && src[i] == '[' {
		// ["key"] form — not needed for backup patching; skip.
		return "", i, false
	}
	if i >= end || !isIdentStart(src[i]) {
		return "", i, false
	}
	j := i + 1
	for j < end && isIdentCont(src[j]) {
		j++
	}
	return src[i:j], j, true
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func skipValue(src string, i, end int) int {
	i = skipSpaceAndComments(src, i, end)
	if i >= end {
		return i
	}
	switch src[i] {
	case '{':
		if closeIdx, ok := matchingBrace(src, i); ok {
			i = closeIdx + 1
		} else {
			return end
		}
	case '"', '\'':
		delim := src[i]
		i++
		escape := false
		for i < end {
			c := src[i]
			i++
			if escape {
				escape = false
				continue
			}
			if c == '\\' {
				escape = true
				continue
			}
			if c == delim {
				break
			}
		}
	default:
		for i < end && src[i] != ',' && src[i] != '}' && src[i] != '\n' {
			i++
		}
	}
	i = skipSpaceAndComments(src, i, end)
	if i < end && src[i] == ',' {
		i++
	}
	return i
}
