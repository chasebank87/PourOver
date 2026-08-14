package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Summary holds final apply/upgrade counts for Finish.
type Summary struct {
	Formulae int
	Casks    int
	Removed  int
	Upgraded int
	Defaults int
	Linked   int
	Skipped  int
	Failures int
}

// Session renders a PourOver header, action progress bar, and streams brew logs underneath.
type Session struct {
	out  io.Writer
	mode string

	mu      sync.Mutex
	total   int
	done    int
	phase   string
	current string
	started bool
	failed  int
	width   int
}

// NewSession creates a UI session writing to out. mode is "apply" or "upgrade".
func NewSession(out io.Writer, mode string) *Session {
	return &Session{out: out, mode: mode, width: 28, phase: "starting"}
}

// Start prints the header and initializes the progress total.
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
	s.renderHeaderLocked()
	s.renderStatusLocked()
}

// SetPhase updates the phase label shown in the header/status (formulae, casks, …).
func (s *Session) SetPhase(phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phase = phase
	if s.started {
		s.renderStatusLocked()
	}
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
	s.clearStatusLocked()
	fmt.Fprintln(s.out, styleFail.Render("☕ failed: "+err.Error()))
	s.renderStatusLocked()
}

// Write implements io.Writer for brew restyled output. Clears the status row,
// writes the payload, then redraws status so brew logs stay readable.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clearStatusLocked()
	n, err := s.out.Write(p)
	if s.started {
		s.renderStatusLocked()
	}
	return n, err
}

// Finish clears the live status row and prints a colored summary.
func (s *Session) Finish(sum Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != "" {
		s.done++
		s.current = ""
	}
	s.clearStatusLocked()
	s.started = false

	if sum.Failures == 0 && sum.Formulae == 0 && sum.Casks == 0 && sum.Removed == 0 &&
		sum.Upgraded == 0 && sum.Defaults == 0 && sum.Linked == 0 && sum.Skipped == 0 {
		fmt.Fprintln(s.out, styleMuted.Render("☕ No changes."))
		return
	}

	if sum.Formulae > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Installed %d formula(s).", sum.Formulae)))
	}
	if sum.Casks > 0 {
		fmt.Fprintln(s.out, styleOK.Render(fmt.Sprintf("☕ Installed %d cask(s).", sum.Casks)))
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
	if sum.Skipped > 0 {
		fmt.Fprintln(s.out, styleMuted.Render(fmt.Sprintf("☕ Skipped %d unsupported action(s).", sum.Skipped)))
	}
	if sum.Failures > 0 {
		fmt.Fprintln(s.out, styleFail.Render(fmt.Sprintf("☕ %d action(s) failed.", sum.Failures)))
	}
}

func (s *Session) renderHeaderLocked() {
	brand := styleBrand.Render("☕ PourOver")
	mode := styleMode.Render(s.mode)
	line := brand + "  " + mode
	fmt.Fprintln(s.out, line)
	fmt.Fprintln(s.out, styleMuted.Render(strings.Repeat("─", 40)))
}

func (s *Session) renderStatusLocked() {
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
	status := fmt.Sprintf("%s %s  %s  %s",
		styleBarOn.Render(bar),
		styleMuted.Render(count),
		styleMode.Render(phase),
		styleMuted.Render(cur),
	)
	// Trailing spaces help clear leftover glyphs when the new line is shorter.
	fmt.Fprintf(s.out, "\r\033[2K%s", status)
}

func (s *Session) clearStatusLocked() {
	fmt.Fprint(s.out, "\r\033[2K")
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
