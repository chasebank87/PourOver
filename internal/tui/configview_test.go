package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/engine"
)

func TestConfigView_RendersStatus(t *testing.T) {
	t.Parallel()

	m := ConfigModel{
		status: engine.ConfigStatus{
			ICloudEnabled:   true,
			ICloudPath:      "/tmp/icloud",
			ICloudAvailable: true,
			GitEnabled:      true,
			GitRemote:       "git@github.com:ex/pourover.git",
			GitBranch:       "main",
			GitRepo:         true,
			GitDirty:        true,
		},
	}
	view := m.View()
	for _, want := range []string{
		"iCloud: enabled",
		"/tmp/icloud",
		"git: enabled",
		"git@github.com:ex/pourover.git",
		"dirty",
		"Push",
		"Pull",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() = %q, want %q", view, want)
		}
	}
}

func TestConfigView_ShowsSetupTipWhenNoRepo(t *testing.T) {
	t.Parallel()

	m := ConfigModel{
		status: engine.ConfigStatus{
			GitSetupTip: "use pourover config git setup <url> for first-time setup",
		},
	}
	view := m.View()
	if !strings.Contains(view, "pourover config git setup") {
		t.Fatalf("View() = %q, want git setup tip", view)
	}
}

func TestConfigView_ICloudToggleLabel(t *testing.T) {
	t.Parallel()

	m := ConfigModel{status: engine.ConfigStatus{ICloudEnabled: false}}
	if !strings.Contains(m.View(), "Enable iCloud") {
		t.Fatalf("View() = %q, want Enable iCloud", m.View())
	}

	m.status.ICloudEnabled = true
	if !strings.Contains(m.View(), "Disable iCloud") {
		t.Fatalf("View() = %q, want Disable iCloud", m.View())
	}
}

func TestConfigUpdate_StatusLoadedMsg(t *testing.T) {
	m := ConfigModel{loading: true}
	next, _ := m.Update(configStatusMsg{
		status: engine.ConfigStatus{
			ICloudEnabled: true,
			ICloudPath:    "/x",
			GitEnabled:    true,
			GitRemote:     "origin-url",
			GitDirty:      false,
			GitRepo:       true,
		},
	})
	cm := next.(ConfigModel)
	if cm.loading {
		t.Fatal("loading should clear")
	}
	view := cm.View()
	if !strings.Contains(view, "iCloud: enabled") || !strings.Contains(view, "origin-url") {
		t.Fatalf("View() = %q after status load", view)
	}
}

func TestConfigUpdate_JKNavigateActions(t *testing.T) {
	m := ConfigModel{status: engine.ConfigStatus{ICloudEnabled: false}}
	if m.cursor != 0 {
		t.Fatalf("cursor = %d", m.cursor)
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	cm := next.(ConfigModel)
	if cm.cursor != 1 {
		t.Fatalf("j: cursor = %d, want 1", cm.cursor)
	}
	next, _ = cm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	cm = next.(ConfigModel)
	if cm.cursor != 0 {
		t.Fatalf("k: cursor = %d, want 0", cm.cursor)
	}
}

func TestConfigUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := ConfigModel{home: home}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestConfigUpdate_QQuitsWhenIdle(t *testing.T) {
	m := ConfigModel{}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestConfigUpdate_EnterEnableICloudStartsCmd(t *testing.T) {
	m := ConfigModel{
		configPath: "/tmp/pourover.lua",
		status:     engine.ConfigStatus{ICloudEnabled: false},
		cursor:     0, // Enable iCloud
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := next.(ConfigModel)
	if !cm.busy {
		t.Fatal("expected busy")
	}
	if cmd == nil {
		t.Fatal("expected enable command")
	}
}

func TestConfigUpdate_ICloudDoneRefreshesStatus(t *testing.T) {
	m := ConfigModel{busy: true}
	next, _ := m.Update(configActionDoneMsg{
		status: engine.ConfigStatus{ICloudEnabled: true, ICloudPath: "/p", ICloudAvailable: true},
		kind:   "icloud",
	})
	cm := next.(ConfigModel)
	if cm.busy {
		t.Fatal("busy should clear")
	}
	if !cm.status.ICloudEnabled {
		t.Fatal("expected status updated")
	}
	if !strings.Contains(cm.View(), "iCloud: enabled") {
		t.Fatalf("View() = %q", cm.View())
	}
}

func TestConfigUpdate_PushDoneShowsMessage(t *testing.T) {
	m := ConfigModel{busy: true}
	next, _ := m.Update(configActionDoneMsg{
		kind:   "push",
		pushed: true,
		remote: "git@github.com:ex/x.git",
	})
	cm := next.(ConfigModel)
	if !strings.Contains(cm.statusLine, "Pushed") {
		t.Fatalf("statusLine = %q", cm.statusLine)
	}
}

func TestEnableDisableICloud_ViaEngineTempLua(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "pourover.lua")
	icloudDir := filepath.Join(root, "icloud")
	if err := os.MkdirAll(icloudDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `return {
  policy = { uninstall_mode = "safe" },
  backup = {
    icloud = {
      enabled = false,
      path = "` + icloudDir + `",
    },
  },
}
`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := engine.EnableICloud(configPath, "", false)
	if err != nil {
		t.Fatal(err)
	}
	m := ConfigModel{status: st}
	if !strings.Contains(m.View(), "iCloud: enabled") {
		t.Fatalf("View() = %q after enable", m.View())
	}

	if err := engine.DisableICloud(configPath); err != nil {
		t.Fatal(err)
	}
	st, err = engine.LoadConfigStatus(configPath)
	if err != nil {
		t.Fatal(err)
	}
	m.status = st
	if !strings.Contains(m.View(), "iCloud: disabled") {
		t.Fatalf("View() = %q after disable", m.View())
	}
}

func TestHomeUpdate_EnterOnConfigOpensConfigView(t *testing.T) {
	m := newTestHome()
	m.cursor = 8 // Config

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm, ok := next.(ConfigModel)
	if !ok {
		t.Fatalf("got %T, want ConfigModel", next)
	}
	if cmd == nil {
		t.Fatal("expected Init/load command when opening config view")
	}
	if cm.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", cm.configPath, m.configPath)
	}
}
