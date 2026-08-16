package discovery

import (
	"bytes"
	"io"
	"strings"
)

// BrewStyleWriter rewrites streamed Homebrew CLI output into PourOver's style.
// It is safe for concurrent use only from a single writer (brew stdout or stderr).
type BrewStyleWriter struct {
	Out io.Writer
	buf []byte
}

// NewBrewStyleWriter returns a writer that restyles brew lines to Out.
func NewBrewStyleWriter(out io.Writer) *BrewStyleWriter {
	return &BrewStyleWriter{Out: out}
}

// Write buffers input and emits restyled complete lines.
// Auth prompts (Password:) are flushed immediately without waiting for a newline
// so they are not stuck in the buffer or glued to a progress bar.
func (w *BrewStyleWriter) Write(p []byte) (int, error) {
	if w.Out == nil {
		return len(p), nil
	}
	w.buf = append(w.buf, p...)
	for {
		if err := w.flushAuthPrompt(); err != nil {
			return len(p), err
		}
		// Prefer newline; also split on bare CR (brew download progress).
		n := bytes.IndexByte(w.buf, '\n')
		r := bytes.IndexByte(w.buf, '\r')
		var i int
		switch {
		case n < 0 && r < 0:
			return len(p), nil
		case n < 0:
			i = r
		case r < 0:
			i = n
		default:
			if r < n {
				i = r
			} else {
				i = n
			}
		}
		line := string(w.buf[:i])
		w.buf = w.buf[i+1:]
		if looksLikeAuthPrompt(line) {
			if _, err := io.WriteString(w.Out, "\n"+strings.TrimRight(line, "\r")); err != nil {
				return len(p), err
			}
			continue
		}
		if styled := StyleBrewLine(line); styled != "" {
			if _, err := io.WriteString(w.Out, styled+"\n"); err != nil {
				return len(p), err
			}
		}
	}
}

func (w *BrewStyleWriter) flushAuthPrompt() error {
	if !looksLikeAuthPrompt(string(w.buf)) {
		return nil
	}
	line := strings.TrimRight(string(w.buf), "\r\n")
	w.buf = nil
	// No trailing newline: leave the cursor after "Password:" for typing.
	_, err := io.WriteString(w.Out, "\n"+line)
	return err
}

// Flush emits any trailing partial line.
func (w *BrewStyleWriter) Flush() error {
	if w.Out == nil || len(w.buf) == 0 {
		w.buf = nil
		return nil
	}
	if looksLikeAuthPrompt(string(w.buf)) {
		return w.flushAuthPrompt()
	}
	styled := StyleBrewLine(string(w.buf))
	w.buf = nil
	if styled == "" {
		return nil
	}
	_, err := io.WriteString(w.Out, styled+"\n")
	return err
}

// StyleBrewLine converts one Homebrew CLI line into PourOver presentation.
// Empty string means the line should be omitted (noise).
func StyleBrewLine(line string) string {
	s := strings.TrimRight(line, "\r")
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}

	if shouldOmitBrewLine(trimmed) {
		return ""
	}
	if looksLikeAuthPrompt(trimmed) {
		return trimmed
	}

	out := trimmed
	// Brew/curl progress sometimes emits TABs that shove text into odd columns
	// once we turn CR updates into normal lines.
	out = strings.ReplaceAll(out, "\t", " ")
	out = strings.ReplaceAll(out, "🍺", "☕")
	switch {
	case strings.HasPrefix(out, "==> "):
		out = "☕ " + strings.TrimSpace(strings.TrimPrefix(out, "==> "))
	case strings.HasPrefix(out, "==>"):
		out = "☕ " + strings.TrimSpace(strings.TrimPrefix(out, "==>"))
	}
	// Collapse leftover double coffee if brew used both markers oddly.
	out = strings.ReplaceAll(out, "☕  ☕", "☕")
	return out
}

func shouldOmitBrewLine(line string) bool {
	switch {
	case strings.HasPrefix(line, "Already downloaded:"):
		return true
	case strings.Contains(line, "Running `brew cleanup"):
		return true
	case strings.HasPrefix(line, "Disable this behaviour"):
		return true
	case strings.HasPrefix(line, "Hide these hints"):
		return true
	case strings.HasPrefix(line, "To re-enable"):
		return true
	case line == "Removing:":
		return true
	default:
		return false
	}
}

func looksLikeAuthPrompt(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasSuffix(lower, "password:"):
		return true
	case strings.Contains(lower, "password for"):
		return true
	case strings.Contains(lower, "passphrase"):
		return true
	default:
		return false
	}
}
