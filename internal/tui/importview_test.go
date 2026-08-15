package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chasebank87/PourOver/internal/config"
	"github.com/chasebank87/PourOver/internal/engine"
)

func TestImportView_DefaultToggles(t *testing.T) {
	t.Parallel()

	m := NewImportModel("/tmp/pourover.lua", newTestHome())
	if !m.packages || !m.files {
		t.Fatalf("packages/files = %v/%v, want true/true", m.packages, m.files)
	}
	if m.force || m.dryRun {
		t.Fatalf("force/dryRun = %v/%v, want false/false", m.force, m.dryRun)
	}
	view := m.View()
	if !strings.Contains(view, "Packages") || !strings.Contains(view, "Files") {
		t.Fatalf("View() = %q, want Packages/Files toggles", view)
	}
	if !strings.Contains(view, "Force") || !strings.Contains(view, "Dry-run") {
		t.Fatalf("View() = %q, want Force/Dry-run toggles", view)
	}
	if !strings.Contains(view, "Run import") {
		t.Fatalf("View() = %q, want Run import action", view)
	}
}

func TestImportUpdate_SpaceTogglesOption(t *testing.T) {
	m := NewImportModel("/tmp/pourover.lua", newTestHome())
	m.cursor = importOptForce

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	im := next.(ImportModel)
	if !im.force {
		t.Fatal("space on Force should enable it")
	}

	next, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	im = next.(ImportModel)
	if im.force {
		t.Fatal("space again should disable Force")
	}
}

func TestImportUpdate_EnterTogglesOption(t *testing.T) {
	m := NewImportModel("/tmp/pourover.lua", newTestHome())
	m.cursor = importOptDryRun

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	im := next.(ImportModel)
	if !im.dryRun {
		t.Fatal("enter on Dry-run should enable it")
	}
}

func TestImportUpdate_JKNavigate(t *testing.T) {
	m := NewImportModel("/tmp/pourover.lua", newTestHome())

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	im := next.(ImportModel)
	if im.cursor != importOptFiles {
		t.Fatalf("j: cursor = %d, want Files", im.cursor)
	}

	next, _ = im.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	im = next.(ImportModel)
	if im.cursor != importOptPackages {
		t.Fatalf("k: cursor = %d, want Packages", im.cursor)
	}
}

func TestImportUpdate_EnterOnRunStartsImport(t *testing.T) {
	m := NewImportModel("/tmp/pourover.lua", newTestHome())
	m.cursor = importOptRun

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	im := next.(ImportModel)
	if !im.busy {
		t.Fatal("expected busy during import")
	}
	if cmd == nil {
		t.Fatal("expected import command")
	}
}

func TestImportUpdate_ImportDoneShowsResult(t *testing.T) {
	m := ImportModel{busy: true, packages: true, files: true}

	next, _ := m.Update(importDoneMsg{
		result: engine.ImportResult{
			DryRun:        true,
			PackagesDone:  true,
			Taps:          []string{"homebrew/core"},
			Formulae:      []string{"git", "fzf"},
			Casks:         []string{"ghostty"},
			FilesDone:     true,
			Links:         []config.FileLink{{Source: "a", Target: "~/.a"}, {Source: "b", Target: "~/.b"}},
			SkippedLinks:  []string{"~/.zshrc"},
			FileLines:     []engine.ImportFileLine{{TargetDecl: "~/.gitconfig", RelSource: "config/home/gitconfig"}},
		},
	})
	im := next.(ImportModel)
	if im.busy {
		t.Fatal("busy should clear after done")
	}
	view := im.View()
	if !strings.Contains(view, "packages") {
		t.Fatalf("View() = %q, want packages summary", view)
	}
	if !strings.Contains(view, "files") {
		t.Fatalf("View() = %q, want files summary", view)
	}
	if !strings.Contains(view, "dry-run") && !strings.Contains(view, "Dry run") {
		t.Fatalf("View() = %q, want dry-run note", view)
	}
	if !strings.Contains(view, "skipped") {
		t.Fatalf("View() = %q, want skipped links count", view)
	}
}

func TestImportUpdate_EscReturnsHome(t *testing.T) {
	home := newTestHome()
	m := NewImportModel(home.configPath, home)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	hm, ok := next.(HomeModel)
	if !ok {
		t.Fatalf("got %T, want HomeModel", next)
	}
	if hm.configPath != home.configPath {
		t.Fatalf("home configPath = %q, want %q", hm.configPath, home.configPath)
	}
}

func TestImportUpdate_QQuitsWhenIdle(t *testing.T) {
	m := NewImportModel("/tmp/pourover.lua", newTestHome())

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("got %T, want tea.QuitMsg", msg)
	}
}

func TestImportUpdate_QIgnoredWhenBusy(t *testing.T) {
	m := ImportModel{busy: true}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q must not quit while busy")
	}
}

func TestHomeUpdate_EnterOnImportOpensImportView(t *testing.T) {
	m := newTestHome()
	m.cursor = 6 // Import

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	im, ok := next.(ImportModel)
	if !ok {
		t.Fatalf("got %T, want ImportModel", next)
	}
	if cmd != nil {
		// Init may be nil; import view has no async load on open
		_ = cmd
	}
	if im.configPath != m.configPath {
		t.Fatalf("configPath = %q, want %q", im.configPath, m.configPath)
	}
}
