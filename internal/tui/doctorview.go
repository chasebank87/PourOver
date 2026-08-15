package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

// DoctorModel shows a read-only engine.Doctor checklist.
type DoctorModel struct {
	configPath string
	home       HomeModel
	report     engine.DoctorReport
	err        string
	loading    bool
	width      int
	height     int
}

// NewDoctorModel constructs a doctor view for configPath, returning to home on esc.
func NewDoctorModel(configPath string, home HomeModel) DoctorModel {
	return DoctorModel{
		configPath: configPath,
		home:       home,
		loading:    true,
	}
}

type doctorLoadedMsg struct {
	report engine.DoctorReport
	err    error
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

func (m DoctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case doctorLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			m.report = engine.DoctorReport{}
			return m, nil
		}
		m.err = ""
		m.report = msg.report
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.home, nil
		case "r":
			m.loading = true
			m.err = ""
			return m, loadDoctorReport(m.configPath)
		}
	}
	return m, nil
}

func (m DoctorModel) View() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render("Doctor"))
	b.WriteString("\n\n")

	switch {
	case m.loading:
		b.WriteString(styleMuted.Render("loading doctor…"))
		b.WriteString("\n")
	case m.err != "":
		b.WriteString(styleMuted.Render("error: " + m.err))
		b.WriteString("\n")
	case len(m.report.Checks) == 0:
		b.WriteString(styleMuted.Render("no checks"))
		b.WriteString("\n")
	default:
		for _, c := range m.report.Checks {
			status := "PASS"
			if !c.OK {
				status = "FAIL"
			}
			line := fmt.Sprintf("[%s] %s: %s", status, c.Name, c.Detail)
			if c.OK {
				b.WriteString(styleSummary.Render(line))
			} else {
				b.WriteString(styleAccent.Render(line))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		if m.report.OK() {
			b.WriteString(styleSummary.Render("All checks passed."))
		} else {
			b.WriteString(styleAccent.Render("Doctor found problems."))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("r refresh · esc back · q quit"))
	b.WriteString("\n")
	return b.String()
}
