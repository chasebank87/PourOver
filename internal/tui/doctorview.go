package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

// DoctorModel shows an engine.Doctor checklist with opt-in safe fixes.
type DoctorModel struct {
	configPath string
	stateDir   string
	home       HomeModel
	report     engine.DoctorReport
	err        string
	tip        string
	loading    bool
	fixing     bool
	cursor     int
	pendingFix string
	confirm    ConfirmModel
	width      int
	height     int
}

// NewDoctorModel constructs a doctor view for configPath, returning to home on esc.
func NewDoctorModel(configPath string, home HomeModel) DoctorModel {
	m := DoctorModel{
		configPath: configPath,
		home:       home,
		loading:    true,
	}
	if stateDir, err := paths.DefaultStateDir(); err == nil {
		m.stateDir = stateDir
	}
	return m
}

type doctorLoadedMsg struct {
	report engine.DoctorReport
	err    error
}

type doctorFixDoneMsg struct {
	err error
}

func (m DoctorModel) Init() tea.Cmd {
	return loadDoctorReport(m.configPath)
}

func loadDoctorReport(configPath string) tea.Cmd {
	return func() tea.Msg {
		report, err := runDoctorChecks(configPath)
		return doctorLoadedMsg{report: report, err: err}
	}
}

func runDoctorChecks(configPath string) (engine.DoctorReport, error) {
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return engine.DoctorReport{}, err
	}

	var manifest config.Manifest
	if m, err := config.LoadManifest(configPath); err == nil {
		manifest = m
	}

	brewOK := true
	brewErr := ""
	if _, err := exec.LookPath("brew"); err != nil {
		brewOK = false
		brewErr = "brew not found on PATH"
	}

	pouroverOK := true
	pouroverErr := ""
	if _, err := exec.LookPath("pourover"); err != nil {
		pouroverOK = false
		pouroverErr = "pourover not found on PATH"
	}

	return engine.Doctor(engine.DoctorInputs{
		ConfigPath:  configPath,
		StateDir:    stateDir,
		Manifest:    manifest,
		BrewOK:      brewOK,
		BrewErr:     brewErr,
		PouroverOK:  pouroverOK,
		PouroverErr: pouroverErr,
	})
}

func runDoctorFix(fix, configPath, stateDir string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch fix {
		case "state":
			err = engine.EnsureStateDir(stateDir)
		case "config":
			err = engine.InitConfig(filepath.Dir(configPath), false)
		default:
			err = fmt.Errorf("unknown fix %q", fix)
		}
		return doctorFixDoneMsg{err: err}
	}
}

func (m DoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case doctorLoadedMsg:
		m.loading = false
		m.fixing = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.report = engine.DoctorReport{}
			return m, nil
		}
		m.err = ""
		m.report = msg.report
		if m.cursor >= len(m.report.Checks) && len(m.report.Checks) > 0 {
			m.cursor = len(m.report.Checks) - 1
		}
		if len(m.report.Checks) == 0 {
			m.cursor = 0
		}
		return m, nil

	case doctorFixDoneMsg:
		m.fixing = false
		m.pendingFix = ""
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.err = ""
		m.tip = ""
		m.loading = true
		return m, loadDoctorReport(m.configPath)

	case tea.KeyMsg:
		if m.loading || m.fixing {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		if m.confirm.Active {
			switch msg.String() {
			case "esc":
				m.confirm.Active = false
				m.pendingFix = ""
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			var answered *bool
			m.confirm, answered = m.confirm.Update(msg)
			if answered == nil {
				return m, nil
			}
			if !*answered {
				m.pendingFix = ""
				return m, nil
			}
			fix := m.pendingFix
			m.fixing = true
			m.err = ""
			m.tip = ""
			return m, runDoctorFix(fix, m.configPath, m.stateDir)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "r":
			m.loading = true
			m.err = ""
			m.tip = ""
			return m, loadDoctorReport(m.configPath)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.report.Checks)-1 {
				m.cursor++
			}
			return m, nil
		case "f":
			return m.offerFix()
		}
	}
	return m, nil
}

func (m DoctorModel) offerFix() (tea.Model, tea.Cmd) {
	if len(m.report.Checks) == 0 || m.cursor < 0 || m.cursor >= len(m.report.Checks) {
		return m, nil
	}
	c := m.report.Checks[m.cursor]
	if c.OK {
		m.tip = ""
		return m, nil
	}

	switch {
	case c.Name == "state":
		m.tip = ""
		m.pendingFix = "state"
		m.confirm = ConfirmModel{
			Prompt: fmt.Sprintf("Create state directory at %s?", m.stateDir),
			Active: true,
		}
		return m, nil
	case c.Name == "config" && configMissing(c):
		m.tip = ""
		m.pendingFix = "config"
		m.confirm = ConfirmModel{
			Prompt: fmt.Sprintf("Initialize config in %s? (force=false)", filepath.Dir(m.configPath)),
			Active: true,
		}
		return m, nil
	default:
		m.pendingFix = ""
		m.tip = tipForCheck(c)
		return m, nil
	}
}

func configMissing(c engine.DoctorCheck) bool {
	return strings.Contains(strings.ToLower(c.Detail), "not found")
}

func tipForCheck(c engine.DoctorCheck) string {
	switch c.Name {
	case "brew":
		return "Install Homebrew (https://brew.sh) and ensure brew is on PATH."
	case "pourover":
		return "Install PourOver and ensure pourover is on PATH."
	case "icloud":
		return "Sign in to iCloud Drive or adjust backup.icloud.path in pourover.lua."
	case "git":
		return "Run `pourover config git setup <url>` to initialize the config repo."
	case "packages":
		return "Fix package names in packages.lua (lowercase Homebrew tokens)."
	case "config":
		return "Fix pourover.lua (or run `pourover init` if missing), then refresh."
	default:
		return fmt.Sprintf("No automatic fix for %s — resolve manually, then press r.", c.Name)
	}
}

func (m DoctorModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render("Doctor"))
	b.WriteString("\n\n")

	if m.confirm.Active {
		b.WriteString(m.confirm.View())
		return b.String()
	}

	switch {
	case m.loading:
		b.WriteString(styleMuted.Render("loading doctor…"))
		b.WriteString("\n")
	case m.fixing:
		b.WriteString(styleMuted.Render("applying fix…"))
		b.WriteString("\n")
	case m.err != "":
		b.WriteString(styleFail.Render("error: " + m.err))
		b.WriteString("\n")
	case len(m.report.Checks) == 0:
		b.WriteString(styleMuted.Render("no checks"))
		b.WriteString("\n")
	default:
		for i, c := range m.report.Checks {
			marker := "  "
			if i == m.cursor {
				marker = styleAccent.Render("> ")
			}
			status := styleSuccess.Render("[PASS]")
			if !c.OK {
				status = styleFail.Render("[FAIL]")
			} else if c.Warn {
				status = styleWarning.Render("[WARN]")
			}
			detail := fmt.Sprintf("%s: %s", c.Name, c.Detail)
			if i == m.cursor {
				detail = styleSelected.Render(detail)
			}
			b.WriteString(marker)
			b.WriteString(status)
			b.WriteString(" ")
			b.WriteString(detail)
			b.WriteString("\n")
		}
		b.WriteString("\n")
		if m.report.OK() {
			b.WriteString(styleSuccess.Render("All checks passed."))
		} else {
			b.WriteString(styleFail.Render("Doctor found problems."))
		}
		b.WriteString("\n")
	}

	if tip := strings.TrimSpace(m.tip); tip != "" {
		b.WriteString("\n")
		b.WriteString(styleWarning.Render("tip: " + tip))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("↑↓ select · f fix · r refresh · esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}
