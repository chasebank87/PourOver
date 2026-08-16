package ui

import (
	"fmt"
	"io"
)

// WriteSummary prints apply/upgrade outcome lines using the same wording as
// Session.Finish. When fancy is true, lines get the ☕ prefix and semantic colors.
func WriteSummary(w io.Writer, sum Summary, fancy bool) {
	if sum.Failures == 0 && sum.Taps == 0 && sum.Formulae == 0 && sum.Casks == 0 && sum.Mas == 0 && sum.Removed == 0 &&
		sum.Upgraded == 0 && sum.Defaults == 0 && sum.Linked == 0 && sum.Managed == 0 && sum.Templates == 0 &&
		sum.Unlinked == 0 && sum.Pruned == 0 && sum.Skipped == 0 && sum.Renames == 0 {
		writeSummaryLine(w, fancy, kindMuted, "No changes.")
		return
	}

	if sum.Taps > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Added %d tap(s).", sum.Taps))
	}
	if sum.Formulae > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Installed %d formula(s).", sum.Formulae))
	}
	if sum.Casks > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Installed %d cask(s).", sum.Casks))
	}
	if sum.Mas > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Installed %d Mac App Store app(s).", sum.Mas))
	}
	if sum.Upgraded > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Upgraded %d package(s).", sum.Upgraded))
	}
	if sum.Removed > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Removed %d package(s).", sum.Removed))
	}
	if sum.Defaults > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Updated %d macOS default(s).", sum.Defaults))
	}
	if sum.Linked > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Updated %d file(s).", sum.Linked))
	}
	if sum.Managed > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Copied %d managed file(s).", sum.Managed))
	}
	if sum.Templates > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Wrote %d template file(s).", sum.Templates))
	}
	if sum.Unlinked > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Unlinked %d file(s).", sum.Unlinked))
	}
	if sum.Pruned > 0 {
		writeSummaryLine(w, fancy, kindOK, fmt.Sprintf("Pruned %d owned file(s).", sum.Pruned))
	}
	if sum.Skipped > 0 {
		writeSummaryLine(w, fancy, kindWarn, fmt.Sprintf("Skipped %d unsupported action(s).", sum.Skipped))
	}
	if sum.Failures > 0 {
		writeSummaryLine(w, fancy, kindFail, fmt.Sprintf("%d action(s) failed.", sum.Failures))
	}
}

type summaryKind int

const (
	kindOK summaryKind = iota
	kindWarn
	kindFail
	kindMuted
)

func writeSummaryLine(w io.Writer, fancy bool, kind summaryKind, msg string) {
	line := msg
	if fancy {
		line = "☕ " + msg
		switch kind {
		case kindOK:
			line = Success().Render(line)
		case kindWarn:
			line = Warning().Render(line)
		case kindFail:
			line = Fail().Render(line)
		case kindMuted:
			line = Muted().Render(line)
		}
	}
	fmt.Fprintln(w, line)
}
