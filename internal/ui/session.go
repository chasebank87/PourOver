package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Summary holds final apply/upgrade counts for Finish.
type Summary struct {
	Taps     int
	Formulae int
	Casks    int
	Mas      int
	Removed  int
	Upgraded int
	Defaults int
	Linked   int
	Managed   int
	Templates int
	Unlinked  int
	Pruned   int
	Skipped  int
	Renames  int
	Failures int
}

// Session renders a PourOver header, one live progress line, and streams brew
// logs underneath. The progress line is updated in place with CR; it is parked
// (cleared onto its own finished line) before brew output or auth prompts so
// Password: is never glued onto the bar.
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
	width      int
	liveStatus bool // true when the cursor sits on the CR status line
}

// NewSession creates a UI session writing to out. mode is "apply" or "upgrade".
func NewSession(out io.Writer, mode string) *Session {
	return &Session{out: out, mode: mode, width: 28, phase: "starting"}
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

// PrepareAuth parks the live progress line and prints the auth hint so a
// subsequent sudo Password: prompt on /dev/tty is not glued onto the bar.
// Brew mutations get this via Write; PAM/system elevation must call it first.
//
// sudo reads the password from /dev/tty and shares the terminal cursor with
// stderr, so we must leave the cursor at column 0 on a fresh line and flush
// before returning — otherwise Password: lands mid-progress-line.
func (s *Session) PrepareAuth() {
	s.mu.Lock()
	defer s.mu.Unlock()
	wasLive := s.liveStatus
	s.parkStatusLocked()
	if !wasLive {
		// liveStatus can be false while the cursor still sits mid-line (e.g. a
		// prior clear missed); force a newline so Password: is never appended.
		fmt.Fprint(s.out, "\n")
	}
	fmt.Fprint(s.out, styleAccentPrompt.Render("☕ authentication required — enter your password if prompted\n"))
	flushWriter(s.out)
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

	if sum.Failures == 0 && sum.Taps == 0 && sum.Formulae == 0 && sum.Casks == 0 && sum.Mas == 0 && sum.Removed == 0 &&
		sum.Upgraded == 0 && sum.Defaults == 0 && sum.Linked == 0 && sum.Managed == 0 && sum.Templates == 0 &&
		sum.Unlinked == 0 && sum.Pruned == 0 && sum.Skipped == 0 && sum.Renames == 0 {
		fmt.Fprintln(s.out, styleMuted.Render("☕ No changes."))
		return
	}

	if sum.Taps > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Added %d tap(s).", sum.Taps)))
	}
	if sum.Formulae > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Installed %d formula(s).", sum.Formulae)))
	}
	if sum.Casks > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Installed %d cask(s).", sum.Casks)))
	}
	if sum.Mas > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Installed %d Mac App Store app(s).", sum.Mas)))
	}
	if sum.Upgraded > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Upgraded %d package(s).", sum.Upgraded)))
	}
	if sum.Removed > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Removed %d package(s).", sum.Removed)))
	}
	if sum.Defaults > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Updated %d macOS default(s).", sum.Defaults)))
	}
	if sum.Linked > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Updated %d file link(s).", sum.Linked)))
	}
	if sum.Managed > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Copied %d managed file(s).", sum.Managed)))
	}
	if sum.Templates > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Wrote %d template file(s).", sum.Templates)))
	}
	if sum.Unlinked > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Unlinked %d file(s).", sum.Unlinked)))
	}
	if sum.Pruned > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Pruned %d owned file(s).", sum.Pruned)))
	}
	if sum.Skipped > 0 {
		fmt.Fprintln(s.out, styleMuted.Render(fmt.Sprintf("☕ Skipped %d unsupported action(s).", sum.Skipped)))
	}
	if sum.Failures > 0 {
		fmt.Fprintln(s.out, styleFail.Render(fmt.Sprintf("☕ %d action(s) failed.", sum.Failures)))
	}
	// Renames: caller prints detail lines after Finish (status must be parked first).
}

func (s *Session) renderHeaderLocked() {
	brand := styleBrand.Render("☕ PourOver")
	mode := styleMode.Render(s.mode)
	fmt.Fprintln(s.out, brand+"  "+mode)
	fmt.Fprintln(s.out, styleMuted.Render(strings.Repeat("─", 40)))
}

func (s *Session) statusLineLocked() string {
	bar := renderBar(s.done, s.total, s.width)
	count := fmt.Sprintf("%d/%d", s.done, s.total)
	phase := s.phase
	if phase == "" {
		phase = "…"
	}
	cur := s.current
	if cur == "" {
		cur = "…"
	}
	return fmt.Sprintf("%s  %s  %s  %s",
		styleBarOn.Render(bar),
		styleMuted.Render(count),
		styleMode.Render(phase),
		styleMuted.Render("→ "+cur),
	)
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
