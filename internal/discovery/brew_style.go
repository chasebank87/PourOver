package discovery

import (
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/chasebank87/PourOver/internal/tty"
)

// BrewStyleWriter rewrites streamed Homebrew CLI output into PourOver's style.
// It is safe for concurrent use from brew stdout and stderr.
type BrewStyleWriter struct {
	Out io.Writer
	// OnPassword, when set, is called instead of forwarding Password: so the
	// real sudo prompt on /dev/tty can take the cursor.
	OnPassword func()

	mu  sync.Mutex
	buf []byte
}

// NewBrewStyleWriter returns a writer that restyles brew lines to Out.
func NewBrewStyleWriter(out io.Writer) *BrewStyleWriter {
	return &BrewStyleWriter{Out: out}
}

// Write buffers input and emits restyled complete lines.
// Auth prompts (Password:) are handled immediately without waiting for a newline
// so they are not stuck in the buffer or glued to a progress bar. The prompt
// itself is not forwarded: sudo owns /dev/tty (echo off). Re-printing
// "Password:" on the captured pipe leaves a typable line with echo on.
func (w *BrewStyleWriter) Write(p []byte) (int, error) {
	if w.Out == nil {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		if err := w.flushAuthPrompt(); err != nil {
			return len(p), err
		}
		// Prefer newline; also split on bare CR (brew download progress).
		n := bytes.IndexByte(w.buf, '\n')
		r := bytes.IndexByte(w.buf, '\r')
		var i int
		splitCR := false
		switch {
		case n < 0 && r < 0:
			return len(p), nil
		case n < 0:
			i = r
			splitCR = true
		case r < 0:
			i = n
		default:
			if r < n {
				i = r
				splitCR = true
			} else {
				i = n
			}
		}
		line := string(w.buf[:i])
		w.buf = w.buf[i+1:]
		if looksLikeAuthPrompt(line) {
			if err := w.handlePassword(); err != nil {
				return len(p), err
			}
			continue
		}
		if styled := StyleBrewLine(line); styled != "" {
			if _, err := io.WriteString(w.Out, styled+"\n"); err != nil {
				return len(p), err
			}
			continue
		}
		// Homebrew CR-clears with spaces; omitting those bytes can leave the
		// cursor mid-line so the next restyled line looks indented.
		if splitCR && len(strings.TrimSpace(stripANSI(line))) == 0 && len(line) > 0 {
			if _, err := io.WriteString(w.Out, "\r\033[2K"); err != nil {
				return len(p), err
			}
		}
	}
}

func (w *BrewStyleWriter) flushAuthPrompt() error {
	if !looksLikeAuthPrompt(string(w.buf)) {
		return nil
	}
	w.buf = nil
	return w.handlePassword()
}

func (w *BrewStyleWriter) handlePassword() error {
	if w.OnPassword != nil {
		w.OnPassword()
		return nil
	}
	if a, ok := w.Out.(interface{ PrepareAuth() }); ok {
		a.PrepareAuth()
		return nil
	}
	if _, err := io.WriteString(w.Out, "☕ authentication required — enter your password if prompted\n"); err != nil {
		return err
	}
	tty.SyncPromptLine()
	return nil
}

// Flush emits any trailing partial line.
func (w *BrewStyleWriter) Flush() error {
	if w.Out == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) == 0 {
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

	out := stripANSI(trimmed)
	out = strings.ReplaceAll(out, "\t", " ")
	out = strings.Join(strings.Fields(out), " ")
	out = strings.ReplaceAll(out, "🍺", "☕")
	switch {
	case strings.HasPrefix(out, "==> "):
		out = "☕ " + strings.TrimSpace(strings.TrimPrefix(out, "==> "))
	case strings.HasPrefix(out, "==>"):
		out = "☕ " + strings.TrimSpace(strings.TrimPrefix(out, "==>"))
	case hasCheckPrefix(out):
		out = "☕ " + trimCheckPrefix(out)
	}
	// Collapse leftover double coffee if brew used both markers oddly.
	out = strings.ReplaceAll(out, "☕ ☕", "☕")
	if strings.HasPrefix(out, "☕  ") {
		out = "☕ " + strings.TrimSpace(strings.TrimPrefix(out, "☕"))
	}
	return out
}

func hasCheckPrefix(s string) bool {
	return strings.HasPrefix(s, "✔︎") || strings.HasPrefix(s, "✔") || strings.HasPrefix(s, "✓")
}

func trimCheckPrefix(s string) string {
	for _, p := range []string{"✔︎", "✔", "✓"} {
		if strings.HasPrefix(s, p) {
			return strings.TrimSpace(strings.TrimPrefix(s, p))
		}
	}
	return s
}

// stripANSI removes CSI/OSC sequences so CR-padded brew progress does not
// leak spaces or leftover styling into restyled lines.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			break
		}
		switch s[i+1] {
		case '[': // CSI
			i += 2
			for i < len(s) {
				c := s[i]
				if c >= '@' && c <= '~' {
					break
				}
				i++
			}
		case ']': // OSC
			i += 2
			for i < len(s) && s[i] != 0x07 {
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
				i++
			}
		default:
			i++
		}
	}
	return b.String()
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
	lower := strings.ToLower(strings.TrimSpace(stripANSI(s)))
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

// looksLikeBrewNeedsAuth is brew announcing a sudo/installer step. Unlike
// Password:, these lines are complete and should stay newline-terminated.
func looksLikeBrewNeedsAuth(s string) bool {
	if looksLikeAuthPrompt(s) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(stripANSI(s)))
	switch {
	case strings.Contains(lower, "which may request your password"):
		return true
	case strings.Contains(lower, "running installer"):
		return true
	default:
		return false
	}
}

// looksLikeSilentBrewWork is brew output that is often followed by minutes of
// no further stdout (sudo pkg, installer scripts, large bottle extract).
func looksLikeSilentBrewWork(s string) bool {
	if looksLikeBrewNeedsAuth(s) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(stripANSI(s)))
	switch {
	case strings.Contains(lower, "pouring "):
		return true
	case strings.Contains(lower, "with `sudo`"):
		return true
	case strings.Contains(lower, "with sudo"):
		return true
	default:
		return false
	}
}

// summarizeBrewStderr turns captured brew stderr into a short failure suffix.
// Homebrew pads checkmark/progress lines to the terminal width; dumping that
// raw into "☕ failed:" wraps and looks like a crash dump.
func summarizeBrewStderr(s string) string {
	s = stripANSI(s)
	var keep []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || shouldOmitBrewLine(line) {
			continue
		}
		if hasCheckPrefix(line) {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "==> fetching") || strings.HasPrefix(lower, "fetching downloads") {
			continue
		}
		if strings.HasPrefix(lower, "==> downloading homebrew api") {
			continue
		}
		if styled := StyleBrewLine(line); styled != "" {
			keep = append(keep, styled)
		}
	}
	if len(keep) == 0 {
		return ""
	}
	if len(keep) > 3 {
		keep = keep[len(keep)-3:]
	}
	out := strings.Join(keep, " · ")
	if len(out) > 240 {
		return out[:237] + "..."
	}
	return out
}
