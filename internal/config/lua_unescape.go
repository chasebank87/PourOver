package config

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// UnescapeLuaUnicode turns leftover Go-style \uXXXX / \UXXXXXXXX sequences in a
// string into real runes. Used when loading macos defaults written by older
// strconv.Quote-based import (Lua does not decode those escapes).
func UnescapeLuaUnicode(s string) string {
	if !strings.Contains(s, `\u`) && !strings.Contains(s, `\U`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'u':
				if i+6 <= len(s) {
					if r, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
						b.WriteRune(rune(r))
						i += 6
						continue
					}
				}
			case 'U':
				if i+10 <= len(s) {
					if r, err := strconv.ParseUint(s[i+2:i+10], 16, 32); err == nil && utf8.ValidRune(rune(r)) {
						b.WriteRune(rune(r))
						i += 10
						continue
					}
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
