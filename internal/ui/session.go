package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/chasebank87/PourOver/internal/tty"
)

// Summary holds final apply/upgrade counts for Finish.
type Summary struct {
	Taps      int
	Formulae  int
	Casks     int
	Mas       int
	Removed   int
	Upgraded  int
	Defaults  int
	Linked    int
	Managed   int
	Templates int
	Unlinked  int
	Pruned    int
	Skipped   int
	Renames   int
	Failures  int
}

// Session renders a PourOver header, one live progress line, and streams brew
// logs underneath. The progress line is updated in place with CR; it is parked
// (cleared onto its own finished line) before brew output or auth prompts so
// Password: is never glued onto the bar.
//
// The live status is always truncated to the terminal width so soft-wrap cannot
// leave remnant rows after \r\033[2K (which only clears one visual row).
type Session struct {
	out  io.Writer
	mode string

	mu         sync.Mutex
	total      int
	done       int
	phase      string
	current    string
	started    bool
	failed     int
	barWidth   int
	cols       int // 0 = detect; tests may set explicitly
	liveStatus bool // true when the cursor sits on the CR status line
}

// NewSession creates a UI session writing to out. mode is "apply" or "upgrade".
func NewSession(out io.Writer, mode string) *Session {
	return &Session{out: out, mode: mode, barWidth: 28, phase: "starting"}
}

// Start prints the header and initializes the progress total.
// When total is 0, only the header is shown (no live 0/0 progress line).
func (s *Session) Start(total int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if total < 0 {
		total = 0
	}
	s.total = total
	s.done = 0
	s.failed = 0
	s.started = true
	s.liveStatus = false
	s.renderHeaderLocked()
	if total > 0 {
		s.renderStatusLocked()
	}
}

// SetPhase updates the phase label. It does not redraw by itself so a phase
// change does not reprint a stale action under a new phase name.
func (s *Session) SetPhase(phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = phase
}

// Step marks the next action as in-progress and advances the completed count
// for the previous action (except the first Step).
func (s *Session) Step(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != "" {
		s.done++
	}
	s.current = label
	s.renderStatusLocked()
}

// Fail records a soft-fail for the current action and prints a failure line.
func (s *Session) Fail(err error) {
	if err == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed++
	s.parkStatusLocked()
	fmt.Fprintln(s.out, styleFail.Render("☕ failed: "+err.Error()))
}

// Write implements io.Writer for brew restyled output.
// Parks the live progress line first so logs and Password: prompts stay clean.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	s.parkStatusLocked()
	if looksLikeAuthPrompt(string(p)) {
		fmt.Fprint(s.out, styleAccentPrompt.Render("☕ authentication required — enter your password if prompted\n"))
	}
	return s.out.Write(p)
}

// PreparePrompt parks the live progress line so a following y/n confirm is
// not glued onto the bar.
func (s *Session) PreparePrompt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parkForPromptLocked()
}

// PrepareAuth parks the live progress line and prints the auth hint so a
// subsequent sudo Password: prompt on /dev/tty is not glued onto the bar.
// Brew mutations get this via Write; PAM/system elevation must call it first.
func (s *Session) PrepareAuth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.parkForPromptLocked()
	fmt.Fprint(s.out, styleAccentPrompt.Render("☕ authentication required — enter your password if prompted"))
	fmt.Fprint(s.out, "\n")
	flushWriter(s.out)
	tty.SyncPromptLine()
}

// parkForPromptLocked clears a one-row live status (or breaks a mid-line write)
// and resets the /dev/tty cursor without inserting blank lines.
func (s *Session) parkForPromptLocked() {
	if s.liveStatus {
		s.parkStatusLocked()
	} else {
		// Best-effort: break mid-line remnants from non-live writes.
		fmt.Fprint(s.out, "\n")
	}
	flushWriter(s.out)
	tty.SyncPromptLine()
}

// Finish parks the status line and prints a colored summary.
func (s *Session) Finish(sum Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != "" {
		s.done++
		s.current = ""
	}
	s.parkStatusLocked()
	s.started = false
	// When Start(0) skipped the progress line, stay under the header rule —
	// a second separator with nothing between looks broken (rename-only apply).
	if s.total > 0 {
		fmt.Fprintln(s.out, styleMuted.Render(strings.Repeat("─", 40)))
	}
	WriteSummary(s.out, sum, true)
	// Renames: caller prints detail lines after Finish (status must be parked first).
}

func (s *Session) renderHeaderLocked() {
	Header(s.out, s.mode)
}

func (s *Session) termColsLocked() int {
	if s.cols > 0 {
		return s.cols
	}
	return writerColumns(s.out)
}

func (s *Session) statusLineLocked() string {
	cols := s.termColsLocked()
	barW := s.barWidth
	if barW > cols/3 {
		barW = cols / 3
	}
	if barW < 8 {
		barW = 8
	}
	if barW > cols-16 {
		barW = cols - 16
	}
	if barW < 4 {
		barW = 4
	}

	bar := renderBar(s.done, s.total, barW)
	count := fmt.Sprintf("%d/%d", s.done, s.total)
	phase := s.phase
	if phase == "" {
		phase = "…"
	}
	cur := s.current
	if cur == "" {
		cur = "…"
	}

	// Keep bar + count + phase always; truncate the action so the whole line
	// fits one terminal row (CR clear only erases one row).
	prefix := fmt.Sprintf("%s  %s  %s",
		styleBarOn.Render(bar),
		styleMuted.Render(count),
		styleMode.Render(phase),
	)
	avail := cols - ansi.StringWidth(prefix) - 1 // leave a margin cell
	if avail < 4 {
		return truncateDisplay(prefix, cols, "…")
	}
	arrow := styleMuted.Render("→ " + cur)
	if ansi.StringWidth(arrow) > avail-1 {
		arrow = styleMuted.Render(truncateDisplay("→ "+cur, avail-1, "…"))
	}
	line := prefix + "  " + arrow
	if ansi.StringWidth(line) > cols {
		return truncateDisplay(line, cols, "…")
	}
	return line
}

func (s *Session) renderStatusLocked() {
	line := s.statusLineLocked()
	if s.liveStatus {
		fmt.Fprintf(s.out, "\r\033[2K%s", line)
		return
	}
	fmt.Fprintf(s.out, "%s", line)
	s.liveStatus = true
}

// parkStatusLocked clears the live CR status line and advances to a new line
// so subsequent brew/log output is not appended beside the bar.
func (s *Session) parkStatusLocked() {
	if !s.liveStatus {
		return
	}
	fmt.Fprint(s.out, "\r\033[2K\n")
	s.liveStatus = false
}

func flushWriter(w io.Writer) {
	type flusher interface{ Flush() error }
	if f, ok := w.(flusher); ok {
		_ = f.Flush()
	}
}

func renderBar(done, total, width int) string {
	if width < 4 {
		width = 4
	}
	if total <= 0 {
		return styleBarOff.Render(strings.Repeat("░", width))
	}
	filled := done * width / total
	if filled > width {
		filled = width
	}
	if done > 0 && filled == 0 {
		filled = 1
	}
	return styleBarOn.Render(strings.Repeat("█", filled)) +
		styleBarOff.Render(strings.Repeat("░", width-filled))
}

// FailureCount returns how many Fail calls occurred this session.
func (s *Session) FailureCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failed
}

// ProgressAdapter returns an exec.Progress-compatible callback for this session.
// Lines starting with "failed:" are routed to Fail; others to Step.
func (s *Session) ProgressAdapter() func(string) {
	return func(line string) {
		if strings.HasPrefix(line, "failed:") {
			msg := strings.TrimSpace(strings.TrimPrefix(line, "failed:"))
			s.Fail(fmt.Errorf("%s", msg))
			return
		}
		s.Step(line)
	}
}

func looksLikeAuthPrompt(s string) bool {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "password:"):
		return true
	case strings.Contains(lower, "password for"):
		return true
	case strings.Contains(lower, "passphrase"):
		return true
	default:
		return false
	}
}
