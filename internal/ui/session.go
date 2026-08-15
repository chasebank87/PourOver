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

// Session renders a PourOver header, action progress, and streams brew logs underneath.
// Progress is printed as normal lines (not carriage-return overlays) so sudo/password
// prompts from Homebrew stay on their own clean line.
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

// SetPhase updates the phase label shown in the status block.
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
	fmt.Fprintln(s.out, styleFail.Render("☕ failed: "+err.Error()))
}

// Write implements io.Writer for brew restyled output.
// It streams logs as-is without redrawing the progress bar, so interactive
// prompts (Password:) are not glued onto the status line.
func (s *Session) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	// Ensure prompts start on a fresh line when brew omits a leading newline.
	if looksLikeAuthPrompt(string(p)) {
		fmt.Fprint(s.out, "\n")
		fmt.Fprint(s.out, styleAccentPrompt.Render("☕ authentication required — enter your password if prompted\n"))
	}
	return s.out.Write(p)
}

// Finish prints a colored summary.
func (s *Session) Finish(sum Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != "" {
		s.done++
		s.current = ""
	}
	s.started = false
	fmt.Fprintln(s.out, styleMuted.Render(strings.Repeat("─", 40)))

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
	fmt.Fprintln(s.out, brand+"  "+mode)
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
	fmt.Fprintln(s.out)
	fmt.Fprintf(s.out, "%s  %s  %s\n",
		styleBarOn.Render(bar),
		styleMuted.Render(count),
		styleMode.Render(phase),
	)
	fmt.Fprintln(s.out, styleMuted.Render("→ "+cur))
	fmt.Fprintln(s.out)
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
