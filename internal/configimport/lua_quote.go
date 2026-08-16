package configimport

import (
	"strings"
	"unicode/utf8"
)

// luaQuote returns a double-quoted Lua string literal. Unlike strconv.Quote,
// non-ASCII runes are emitted as UTF-8 (not \uXXXX), which gopher-lua / Lua 5.1
// preserve correctly.
func luaQuote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\a':
			b.WriteString(`\a`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\v':
			b.WriteString(`\v`)
		case 0:
			b.WriteString(`\0`)
		default:
			if r < 0x20 {
				b.WriteString(`\x`)
				const hex = "0123456789abcdef"
				b.WriteByte(hex[r>>4])
				b.WriteByte(hex[r&0xf])
				continue
			}
			if !utf8.ValidRune(r) {
				b.WriteRune(utf8.RuneError)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
