package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
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
// Cancel mid-run is not implemented (esc/q/ctrl+c ignored until done).
// Config auto-push after apply is skipped in TUI (Phase 2 config screens).
type RunModel struct {
	kind       RunKind
	configPath string
	home       HomeModel

	phase   string
	lines   []string
	done    bool
	summary string
	err     string

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

// logWriter streams brew stdout/stderr into the run log as line messages.
type logWriter struct {
	send func(string)
	mu   sync.Mutex
	buf  bytes.Buffer
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	for {
		data := w.buf.Bytes()
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		line := string(bytes.TrimRight(data[:i], "\r"))
		w.buf.Next(i + 1)
		if line != "" && w.send != nil {
			w.send(line)
		}
	}
	return len(p), nil
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
	sendLine := func(line string) {
		send(progressLineMsg{line: line})
	}

	progress := engine.Progress(sendLine)
	brewLog := &logWriter{send: sendLine}

	switch m.kind {
	case RunUpgrade:
		runUpgradeFlow(ctx, m, runner, progress, brewLog, send, sendDone)
	default:
		runApplyFlow(ctx, m, runner, progress, brewLog, send, sendDone)
	}
}

func runApplyFlow(
	ctx context.Context,
	m RunModel,
	runner discovery.Runner,
	progress engine.Progress,
	brewLog *logWriter,
	send func(tea.Msg),
	sendDone func(string, error),
) {
	p, err := engine.BuildPlan(ctx, m.configPath, runner)
	if err != nil {
		sendDone("", err)
		return
	}
	manifest, mode, stateDir, prepErr := prepareFinalize(m.configPath)
	if prepErr != nil {
		sendDone("", prepErr)
		return
	}
	result, applyErr := engine.Apply(ctx, runner, p, engine.ApplyOptions{
		ConfigPath:  m.configPath,
		ConfigDir:   filepath.Dir(m.configPath),
		StateDir:    stateDir,
		Mode:        mode,
		FileReplace: policy.ResolveFileReplaceFromManifest(manifest),
		FilesMode:   policy.ResolveFilesModeFromManifest(manifest),
		AutoYes:     false,
		Progress:    progress,
		Confirm:     m.confirmer,
		Stdout:      brewLog,
		Stderr:      brewLog,
		OnPhase: func(phase string) {
			send(phaseMsg{phase: phase})
		},
	})
	finalErr := engine.FinalizeApply(engine.FinalizeOptions{
		StateDir:  stateDir,
		ConfigDir: filepath.Dir(m.configPath),
		Manifest:  manifest,
	}, p, applyErr)
	// Skip maybeAutoPushConfig in TUI (Phase 2 config screens).
	sendDone(formatApplySummary(result), finalErr)
}

func runUpgradeFlow(
	ctx context.Context,
	m RunModel,
	runner discovery.Runner,
	progress engine.Progress,
	brewLog *logWriter,
	send func(tea.Msg),
	sendDone func(string, error),
) {
	upPlan, err := engine.BuildUpgradePlan(ctx, m.configPath, runner)
	if err != nil {
		sendDone("", err)
		return
	}
	send(phaseMsg{phase: "upgrade"})
	upResult, upErr := engine.UpgradePackages(ctx, runner, upPlan, engine.UpgradeOptions{
		Progress: progress,
		Stdout:   brewLog,
		Stderr:   brewLog,
	})

	applyPlan, err := engine.BuildPlan(ctx, m.configPath, runner)
	if err != nil {
		sendDone(formatUpgradeSummary(upResult), errors.Join(upErr, err))
		return
	}
	manifest, mode, stateDir, prepErr := prepareFinalize(m.configPath)
	if prepErr != nil {
		sendDone(formatUpgradeSummary(upResult), errors.Join(upErr, prepErr))
		return
	}
	send(phaseMsg{phase: "apply"})
	applyResult, applyErr := engine.Apply(ctx, runner, applyPlan, engine.ApplyOptions{
		ConfigPath:  m.configPath,
		ConfigDir:   filepath.Dir(m.configPath),
		StateDir:    stateDir,
		Mode:        mode,
		FileReplace: policy.ResolveFileReplaceFromManifest(manifest),
		FilesMode:   policy.ResolveFilesModeFromManifest(manifest),
		AutoYes:     false,
		Progress:    progress,
		Confirm:     m.confirmer,
		Stdout:      brewLog,
		Stderr:      brewLog,
		OnPhase: func(phase string) {
			send(phaseMsg{phase: phase})
		},
	})
	finalErr := engine.FinalizeApply(engine.FinalizeOptions{
		StateDir:  stateDir,
		ConfigDir: filepath.Dir(m.configPath),
		Manifest:  manifest,
	}, applyPlan, applyErr)
	// Skip maybeAutoPushConfig in TUI (Phase 2 config screens).
	summary := joinSummaries(formatUpgradeSummary(upResult), formatApplySummary(applyResult))
	sendDone(summary, errors.Join(upErr, finalErr))
}

func prepareFinalize(configPath string) (config.Manifest, config.UninstallMode, string, error) {
	manifest, err := config.LoadManifest(configPath)
	if err != nil {
		return config.Manifest{}, "", "", fmt.Errorf("load config: %w", err)
	}
	mode := policy.ResolveModeFromManifest(manifest)
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return manifest, mode, "", err
	}
	return manifest, mode, stateDir, nil
}

func joinSummaries(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || p == "No changes." || p == "No upgrades." {
			continue
		}
		out = append(out, strings.TrimPrefix(p, "Done: "))
	}
	if len(out) == 0 {
		return "No changes."
	}
	return "Done: " + strings.Join(out, "; ")
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
			return m, nil
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
	add(r.Managed, "managed copy", "managed copies")
	add(r.Unlinked, "unlink", "unlinks")
	add(r.Pruned, "prune", "prunes")
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
		parts = append(parts, fmt.Sprintf("%d upgraded", r.Upgraded))
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
