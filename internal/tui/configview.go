package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
)

type configAction int

const (
	configActionICloud configAction = iota
	configActionPush
	configActionPull
	configActionRefresh
	configActionCount
)

// ConfigModel shows iCloud/git sync status and common actions.
type ConfigModel struct {
	configPath string
	home       HomeModel

	status     engine.ConfigStatus
	cursor     int
	loading    bool
	busy       bool
	err        string
	statusLine string

	width  int
	height int
}

// NewConfigModel constructs a config sync view, returning to home on esc when idle.
func NewConfigModel(configPath string, home HomeModel) ConfigModel {
	return ConfigModel{
		configPath: configPath,
		home:       home,
		loading:    true,
	}
}

type configStatusMsg struct {
	status engine.ConfigStatus
	err    error
}

type configActionDoneMsg struct {
	kind   string // icloud | push | pull | refresh
	status engine.ConfigStatus
	pushed bool
	remote string
	err    error
}

func (m ConfigModel) Init() tea.Cmd {
	return loadConfigStatusCmd(m.configPath)
}

func loadConfigStatusCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		st, err := engine.LoadConfigStatus(configPath)
		return configStatusMsg{status: st, err: err}
	}
}

func runEnableICloudCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		st, err := engine.EnableICloud(configPath, "", false)
		return configActionDoneMsg{kind: "icloud", status: st, err: err}
	}
}

func runDisableICloudCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		if err := engine.DisableICloud(configPath); err != nil {
			return configActionDoneMsg{kind: "icloud", err: err}
		}
		st, err := engine.LoadConfigStatus(configPath)
		return configActionDoneMsg{kind: "icloud", status: st, err: err}
	}
}

func runPushConfigCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		result, err := engine.PushConfig(configPath)
		if err != nil {
			return configActionDoneMsg{kind: "push", err: err}
		}
		st, _ := engine.LoadConfigStatus(configPath)
		return configActionDoneMsg{
			kind:   "push",
			status: st,
			pushed: result.Pushed,
			remote: result.Remote,
		}
	}
}

func runPullConfigCmd(configPath string) tea.Cmd {
	return func() tea.Msg {
		if err := engine.PullConfig(configPath); err != nil {
			return configActionDoneMsg{kind: "pull", err: err}
		}
		st, err := engine.LoadConfigStatus(configPath)
		return configActionDoneMsg{kind: "pull", status: st, err: err}
	}
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case configStatusMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.status = msg.status
		return m, nil

	case configActionDoneMsg:
		m.busy = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.statusLine = ""
			return m, nil
		}
		m.err = ""
		m.status = msg.status
		switch msg.kind {
		case "push":
			if msg.pushed {
				m.statusLine = fmt.Sprintf("Pushed config changes to %s", msg.remote)
			} else {
				m.statusLine = "Nothing to push (already synced)."
			}
		case "pull":
			m.statusLine = "Pulled config updates."
		case "icloud":
			if msg.status.ICloudEnabled {
				m.statusLine = "iCloud mirror enabled."
			} else {
				m.statusLine = "iCloud mirror disabled."
			}
		case "refresh":
			m.statusLine = "Status refreshed."
		}
		return m, nil

	case tea.KeyMsg:
		if m.busy {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < int(configActionCount)-1 {
				m.cursor++
			}
		case "enter":
			return m.activate()
		case "r":
			m.busy = true
			m.err = ""
			m.statusLine = ""
			return m, func() tea.Msg {
				st, err := engine.LoadConfigStatus(m.configPath)
				return configActionDoneMsg{kind: "refresh", status: st, err: err}
			}
		}
	}
	return m, nil
}

func (m ConfigModel) activate() (tea.Model, tea.Cmd) {
	m.busy = true
	m.err = ""
	m.statusLine = ""
	switch configAction(m.cursor) {
	case configActionICloud:
		if m.status.ICloudEnabled {
			return m, runDisableICloudCmd(m.configPath)
		}
		return m, runEnableICloudCmd(m.configPath)
	case configActionPush:
		return m, runPushConfigCmd(m.configPath)
	case configActionPull:
		return m, runPullConfigCmd(m.configPath)
	case configActionRefresh:
		return m, func() tea.Msg {
			st, err := engine.LoadConfigStatus(m.configPath)
			return configActionDoneMsg{kind: "refresh", status: st, err: err}
		}
	}
	m.busy = false
	return m, nil
}

func (m ConfigModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	if m.configPath != "" {
		b.WriteString(styleMuted.Render("config: " + m.configPath))
		b.WriteString("\n")
	}
	b.WriteString(styleSummary.Render("Config sync"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(styleMuted.Render("loading…"))
		b.WriteString("\n")
	} else {
		b.WriteString(m.renderStatus())
		b.WriteString("\n")
		if m.busy {
			b.WriteString(styleMuted.Render("working…"))
			b.WriteString("\n")
		} else {
			b.WriteString(m.renderActions())
		}
	}

	if m.statusLine != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleSummary.Render(m.statusLine))
		b.WriteString("\n")
	}
	if m.err != "" && !m.busy {
		b.WriteString("\n")
		b.WriteString(styleAccent.Render("error: " + m.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render(m.footer()))
	b.WriteString("\n")
	return b.String()
}

func (m ConfigModel) renderStatus() string {
	var b strings.Builder

	icloud := "iCloud: disabled"
	if m.status.ICloudEnabled {
		icloud = "iCloud: enabled"
		if m.status.ICloudPath != "" {
			icloud += " · " + m.status.ICloudPath
			if !m.status.ICloudAvailable {
				icloud += " (unavailable)"
			}
		}
	} else if m.status.ICloudPath != "" {
		icloud += " · path " + m.status.ICloudPath
	}
	b.WriteString(styleSummary.Render(icloud))
	b.WriteString("\n")

	git := "git: disabled"
	if m.status.GitEnabled {
		git = "git: enabled"
	}
	if m.status.GitRepo {
		git += " · repo"
	} else {
		git += " · not a repo"
	}
	if m.status.GitRemote != "" {
		git += " · " + m.status.GitRemote
	}
	if m.status.GitBranch != "" && m.status.GitRepo {
		git += " (" + m.status.GitBranch + ")"
	}
	if m.status.GitDirty {
		git += " · dirty"
	} else if m.status.GitRepo {
		git += " · clean"
	}
	b.WriteString(styleSummary.Render(git))
	b.WriteString("\n")

	if tip := strings.TrimSpace(m.status.GitSetupTip); tip != "" {
		b.WriteString(styleMuted.Render("tip: " + tip))
		b.WriteString("\n")
	}
	return b.String()
}

func (m ConfigModel) renderActions() string {
	labels := []string{
		"Enable iCloud",
		"Push",
		"Pull",
		"Refresh",
	}
	if m.status.ICloudEnabled {
		labels[0] = "Disable iCloud"
	}

	var b strings.Builder
	for i, label := range labels {
		cursor := "  "
		line := label
		if i == m.cursor {
			cursor = styleAccent.Render("> ")
			line = styleSelected.Render(label)
		} else {
			line = styleMenu.Render(label)
		}
		b.WriteString(cursor)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func (m ConfigModel) footer() string {
	if m.busy {
		return "please wait…"
	}
	return "↑/↓ or j/k · enter · r refresh · esc back · q quit"
}
