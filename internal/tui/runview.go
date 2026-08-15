package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/policy"
)

// RunKind selects apply vs upgrade execution.
type RunKind string

const (
	RunApply   RunKind = "apply"
	RunUpgrade RunKind = "upgrade"
)

const maxRunLogLines = 200

// RunModel shows phase + scrolling progress for Apply or Upgrade.
// Cancel mid-run is not implemented in 1.5 (esc only returns home when done).
type RunModel struct {
	kind       RunKind
	configPath string
	home       HomeModel

	phase  string
	lines  []string
	done   bool
	summary string
	err    string

	confirm   ConfirmModel
	confirmer *AsyncConfirmer
	events    chan tea.Msg

	width  int
	height int
}

// NewRunModel constructs an apply/upgrade run view returning to home on esc when done.
func NewRunModel(kind RunKind, configPath string, home HomeModel) RunModel {
	return RunModel{
		kind:       kind,
		configPath: configPath,
		home:       home,
		confirmer:  NewAsyncConfirmer(),
		events:     make(chan tea.Msg, 64),
		phase:      "starting",
	}
}

type progressLineMsg struct {
	line string
}

type phaseMsg struct {
	phase string
}

type runDoneMsg struct {
	summary string
	err     error
}

func (m RunModel) Init() tea.Cmd {
	return tea.Batch(
		waitConfirmCmd(m.confirmer),
		waitRunEventCmd(m.events),
		startRunCmd(m),
	)
}

func waitRunEventCmd(events <-chan tea.Msg) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

func startRunCmd(m RunModel) tea.Cmd {
	return func() tea.Msg {
		runEngine(m)
		return nil
	}
}

func runEngine(m RunModel) {
	ctx := context.Background()
	runner := discovery.NewExecRunner()
	send := func(msg tea.Msg) {
		m.events <- msg
	}
	sendDone := func(summary string, err error) {
		m.events <- runDoneMsg{summary: summary, err: err}
	}

	progress := engine.Progress(func(line string) {
		send(progressLineMsg{line: line})
	})

	switch m.kind {
	case RunUpgrade:
		p, err := engine.BuildUpgradePlan(ctx, m.configPath, runner)
		if err != nil {
			sendDone("", err)
			return
		}
		send(phaseMsg{phase: "upgrade"})
		result, err := engine.UpgradePackages(ctx, runner, p, engine.UpgradeOptions{
			Progress: progress,
		})
		sendDone(formatUpgradeSummary(result), err)
	default: // Apply
		p, err := engine.BuildPlan(ctx, m.configPath, runner)
		if err != nil {
			sendDone("", err)
			return
		}
		mode := config.UninstallModeSafe
		if manifest, err := config.LoadManifest(m.configPath); err == nil {
			mode = policy.ResolveModeFromManifest(manifest)
		}
		result, err := engine.Apply(ctx, runner, p, engine.ApplyOptions{
			ConfigPath: m.configPath,
			ConfigDir:  filepath.Dir(m.configPath),
			Mode:       mode,
			AutoYes:    false,
			Progress:   progress,
			Confirm:    m.confirmer,
			OnPhase: func(phase string) {
				send(phaseMsg{phase: phase})
			},
		})
		sendDone(formatApplySummary(result), err)
	}
}

func (m RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case progressLineMsg:
		m.lines = append(m.lines, msg.line)
		if len(m.lines) > maxRunLogLines {
			m.lines = m.lines[len(m.lines)-maxRunLogLines:]
		}
		return m, waitRunEventCmd(m.events)

	case phaseMsg:
		m.phase = msg.phase
		return m, waitRunEventCmd(m.events)

	case runDoneMsg:
		m.done = true
		m.phase = "done"
		m.summary = msg.summary
		if msg.err != nil {
			m.err = friendlyErr(msg.err)
		}
		return m, nil

	case confirmNeededMsg:
		m.confirm = ConfirmModel{Prompt: msg.Prompt, Active: true}
		return m, nil

	case tea.KeyMsg:
		if m.confirm.Active {
			var answered *bool
			m.confirm, answered = m.confirm.Update(msg)
			if answered != nil && m.confirmer != nil {
				m.confirmer.Answer(*answered)
				return m, tea.Batch(waitConfirmCmd(m.confirmer), waitRunEventCmd(m.events))
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.done {
				return m, tea.Quit
			}
			return m, tea.Quit
		case "esc":
			if m.done {
				return m.home, nil
			}
			return m, nil
		}
	}
	return m, nil
}

func (m RunModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")

	title := "Apply"
	if m.kind == RunUpgrade {
		title = "Upgrade"
	}
	b.WriteString(styleSummary.Render(title))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("phase: " + m.phase))
	b.WriteString("\n\n")

	if m.confirm.Active {
		b.WriteString(m.confirm.View())
		b.WriteString("\n")
	}

	for _, line := range m.lines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.done {
		b.WriteString("\n")
		if m.summary != "" {
			b.WriteString(styleSummary.Render(m.summary))
			b.WriteString("\n")
		}
		if m.err != "" {
			b.WriteString(styleMuted.Render("error: " + m.err))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(styleMuted.Render("esc back · q quit"))
		b.WriteString("\n")
	} else if !m.confirm.Active {
		b.WriteString("\n")
		b.WriteString(styleMuted.Render("running… (cancel not implemented)"))
		b.WriteString("\n")
	}

	return b.String()
}

func formatApplySummary(r engine.ApplyResult) string {
	parts := []string{}
	add := func(n int, singular, plural string) {
		if n == 0 {
			return
		}
		label := singular
		if n != 1 {
			label = plural
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, label))
	}
	add(r.Taps, "tap", "taps")
	add(r.Formulae, "formula", "formulae")
	add(r.Casks, "cask", "casks")
	add(r.Removed, "removed", "removed")
	add(r.Defaults, "default", "defaults")
	add(r.Linked, "link", "links")
	add(r.Renames, "rename note", "rename notes")
	add(r.Skipped, "skipped", "skipped")
	add(r.Failures, "failure", "failures")
	if len(parts) == 0 {
		return "No changes."
	}
	return "Done: " + strings.Join(parts, ", ")
}

func formatUpgradeSummary(r engine.UpgradeResult) string {
	parts := []string{}
	if r.Upgraded > 0 {
		label := "upgraded"
		parts = append(parts, fmt.Sprintf("%d %s", r.Upgraded, label))
	}
	if r.Failures > 0 {
		label := "failure"
		if r.Failures != 1 {
			label = "failures"
		}
		parts = append(parts, fmt.Sprintf("%d %s", r.Failures, label))
	}
	if len(parts) == 0 {
		return "No upgrades."
	}
	return "Done: " + strings.Join(parts, ", ")
}
