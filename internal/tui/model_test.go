package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jvanrhyn/banner-maker/internal/tui"
)

func sendKey(m tea.Model, key string) tea.Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == "enter" {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	}
	if key == "backspace" {
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	}
	updated, _ := m.Update(msg)
	return updated
}

func typeWord(m tea.Model, word string) tea.Model {
	for _, ch := range word {
		m = sendKey(m, string(ch))
	}
	return m
}

func TestInitialScreen(t *testing.T) {
	m := tui.InitialModel()
	view := m.View()

	if !strings.Contains(view, "Enter") && !strings.Contains(view, "word") {
		t.Errorf("expected input prompt in initial view, got: %s", view)
	}
}

func TestSubmitEmptyWord_StaysOnInput(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	// Press enter without typing anything
	m = sendKey(m, "enter")

	view := m.View()
	// Should still be on input screen with an error hint
	if strings.Contains(view, "╚═╝") {
		t.Error("should not transition to preview with empty input")
	}
}

func TestSubmitWord_TransitionsToPreview(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = sendKey(m, "enter")

	view := m.View()
	if !strings.Contains(view, "█") {
		t.Errorf("expected banner art in preview screen, got: %s", view)
	}
}

func TestPreviewReject_BackToInput(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = sendKey(m, "enter")

	// Reject by pressing 'n'
	m = sendKey(m, "n")

	view := m.View()
	if strings.Contains(view, "█") {
		t.Errorf("pressing 'n' should return to input screen, but still shows banner art: %s", view)
	}
}

func TestPreviewConfirm_TransitionsToDone(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = sendKey(m, "enter")

	// Confirm by pressing 'y'
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	// Execute the save command synchronously to get the savedMsg back
	if cmd != nil {
		msg := cmd()
		updated, _ = updated.Update(msg)
	}

	view := updated.View()
	// Done screen should mention "Saved" or "banner.txt"
	if !strings.Contains(view, "Saved") && !strings.Contains(view, "banner") {
		t.Errorf("expected done/saved confirmation in view, got: %s", view)
	}
}

func TestWindowResize_Handled(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_ = updated.View() // should not panic
}
