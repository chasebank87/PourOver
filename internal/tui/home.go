package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/discovery"
	"github.com/chasebank87/PourOver/internal/engine"
	"github.com/chasebank87/PourOver/internal/paths"
)

type menuID string

const (
	menuPlan          menuID = "plan"
	menuApply         menuID = "apply"
	menuUpgrade       menuID = "upgrade"
	menuDoctor        menuID = "doctor"
	menuHistory       menuID = "history"
	menuBackupRestore menuID = "backup"
	menuImport        menuID = "import"
	menuConfig        menuID = "config"
	menuQuit          menuID = "quit"
)

type menuItem struct {
	id    menuID
	label string
}

func defaultMenuItems() []menuItem {
	return []menuItem{
		{menuPlan, "Plan"},
		{menuApply, "Apply"},
		{menuUpgrade, "Upgrade"},
		{menuDoctor, "Doctor"},
		{menuHistory, "History"},
		{menuBackupRestore, "Backup/Restore"},
		{menuImport, "Import"},
		{menuConfig, "Config"},
		{menuQuit, "Quit"},
	}
}

// HomeModel is the TUI root: status strip + action menu.
type HomeModel struct {
	items      []menuItem
	cursor     int
	configPath string
	driftLine  string
	doctorLine string
	stub       string // non-empty when showing a stub screen
	width      int
	height     int
}

// NewHomeModel constructs the home screen. Summary fields fill in via Init.
func NewHomeModel() HomeModel {
	configPath, err := paths.ResolveConfigFile("")
	if err != nil {
		configPath = "(unknown)"
	}
	return HomeModel{
		items:      defaultMenuItems(),
		configPath: configPath,
		driftLine:  "drift: loading…",
		doctorLine: "doctor: not checked",
	}
}

type summaryMsg struct {
	configPath string
	driftLine  string
	doctorLine string
}

func (m HomeModel) Init() tea.Cmd {
	return loadHomeSummary
}

func loadHomeSummary() tea.Msg {
	configPath, err := paths.ResolveConfigFile("")
	if err != nil {
		return summaryMsg{
			configPath: "(unknown)",
			driftLine:  "drift: " + err.Error(),
			doctorLine: "doctor: not checked",
		}
	}

	driftLine := loadDriftLine(configPath)
	doctorLine := loadDoctorLine(configPath)
	return summaryMsg{
		configPath: configPath,
		driftLine:  driftLine,
		doctorLine: doctorLine,
	}
}

func loadDriftLine(configPath string) string {
	p, err := engine.BuildPlan(context.Background(), configPath, discovery.NewExecRunner())
	if err != nil {
		return "drift: " + friendlyErr(err)
	}
	n := len(p.Actions)
	if n == 0 {
		return "drift: in sync (0 actions)"
	}
	return fmt.Sprintf("drift: %d action(s)", n)
}

func loadDoctorLine(configPath string) string {
	stateDir, err := paths.DefaultStateDir()
	if err != nil {
		return "doctor: " + err.Error()
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

	report, err := engine.Doctor(engine.DoctorInputs{
		ConfigPath:  configPath,
		StateDir:    stateDir,
		Manifest:    manifest,
		BrewOK:      brewOK,
		BrewErr:     brewErr,
		PouroverOK:  pouroverOK,
		PouroverErr: pouroverErr,
	})
	if err != nil {
		return "doctor: " + err.Error()
	}
	fail := 0
	for _, c := range report.Checks {
		if !c.OK {
			fail++
		}
	}
	if fail == 0 {
		return fmt.Sprintf("doctor: ok (%d checks)", len(report.Checks))
	}
	return fmt.Sprintf("doctor: %d issue(s)", fail)
}

func friendlyErr(err error) string {
	msg := err.Error()
	if isMissingConfigErr(err, msg) {
		return "config missing — run pourover init"
	}
	return truncateErr(msg, 120)
}

func isMissingConfigErr(err error, msg string) bool {
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	lower := strings.ToLower(msg)
	if strings.HasPrefix(lower, "load config:") {
		return true
	}
	if strings.Contains(lower, "config not found") {
		return true
	}
	return false
}

func truncateErr(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if max > 0 && len(msg) > max {
		return msg[:max] + "…"
	}
	return msg
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case summaryMsg:
		m.configPath = msg.configPath
		m.driftLine = msg.driftLine
		m.doctorLine = msg.doctorLine
		return m, nil

	case tea.KeyMsg:
		if m.stub != "" {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q":
				m.stub = ""
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m.activate()
		}
	}
	return m, nil
}

func (m HomeModel) activate() (tea.Model, tea.Cmd) {
	if len(m.items) == 0 {
		return m, nil
	}
	item := m.items[m.cursor]
	switch item.id {
	case menuQuit:
		return m, tea.Quit
	case menuPlan:
		pm := NewPlanModel(m.configPath, m)
		return pm, pm.Init()
	case menuApply:
		rm := NewRunModel(RunApply, m.configPath, m)
		return rm, rm.Init()
	case menuUpgrade:
		rm := NewRunModel(RunUpgrade, m.configPath, m)
		return rm, rm.Init()
	case menuDoctor:
		dm := NewDoctorModel(m.configPath, m)
		return dm, dm.Init()
	case menuHistory:
		hm := NewHistoryModel(m)
		return hm, hm.Init()
	case menuBackupRestore:
		bm := NewBackupModel(m.configPath, m)
		return bm, bm.Init()
	case menuImport:
		im := NewImportModel(m.configPath, m)
		return im, im.Init()
	case menuConfig:
		cm := NewConfigModel(m.configPath, m)
		return cm, cm.Init()
	default:
		m.stub = stubTitle(item)
		return m, nil
	}
}

func stubTitle(item menuItem) string {
	return fmt.Sprintf("%s view: coming soon (esc to return)", item.label)
}

func (m HomeModel) View() string {
	if m.stub != "" {
		return styleTitle.Render("PourOver") + "\n\n" +
			styleMuted.Render(m.stub) + "\n"
	}

	var b strings.Builder
	b.WriteString(styleTitle.Render("PourOver"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("config: " + m.configPath))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render(m.driftLine))
	b.WriteString("\n")
	b.WriteString(styleSummary.Render(m.doctorLine))
	b.WriteString("\n\n")

	for i, item := range m.items {
		cursor := "  "
		label := item.label
		if i == m.cursor {
			cursor = styleAccent.Render("> ")
			label = styleSelected.Render(item.label)
		} else {
			label = styleMenu.Render(item.label)
		}
		b.WriteString(cursor)
		b.WriteString(label)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("↑/↓ or j/k · enter · q quit"))
	b.WriteString("\n")
	return b.String()
}
