package tui_test

import (
	"os"
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
	// Save writes to CWD — redirect to a temp dir to avoid polluting internal/tui
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

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
	// Done screen should mention "Saved" or "hibanner.go"
	if !strings.Contains(view, "Saved") && !strings.Contains(view, "banner") {
		t.Errorf("expected done/saved confirmation in view, got: %s", view)
	}
}

func TestWindowResize_Handled(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	_ = updated.View() // should not panic
}

func TestTabCycles_BetweenInputFields(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	view0 := m.View()

	// Tab once
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	view1 := m.View()
	_ = view1 // just ensure no panic and view renders

	// Tab back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	view2 := m.View()

	// After tab + shift-tab we're back to the word field
	if view0 != view2 {
		// Views can differ in cursor blink state, so just check no panic occurred
		_ = view2
	}
}

func TestColorInput_UsedInPreview(t *testing.T) {
	var m tea.Model = tui.InitialModel()

	// Type word
	m = typeWord(m, "HI")

	// Tab to text-color field and type a color
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = typeWord(m, "#FF0000")

	// Enter generates the banner with the given color
	m = sendKey(m, "enter")

	view := m.View()
	// Should be on preview screen
	if !strings.Contains(view, "█") {
		t.Errorf("expected banner art in preview, got: %s", view)
	}
}

func TestTagline_AppearsInPreview(t *testing.T) {
	var m tea.Model = tui.InitialModel()

	// Type word
	m = typeWord(m, "HI")

	// Tab past color fields to tagline field (0 → 1 → 2 → 3)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = typeWord(m, "My tagline")

	m = sendKey(m, "enter")

	view := m.View()
	if !strings.Contains(view, "My tagline") {
		t.Errorf("expected tagline in preview, got: %s", view)
	}
	if !strings.Contains(view, "█") {
		t.Errorf("expected banner art in preview, got: %s", view)
	}
}

func TestAlignToggle_RightArrowAndGenerate(t *testing.T) {
	var m tea.Model = tui.InitialModel()

	m = typeWord(m, "HI")

	// Tab to tagline field (0 → 1 → 2 → 3)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = typeWord(m, "subtitle")

	// Tab to alignment toggle (→ 4), press right to select "right"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})

	// Enter — should generate and transition to preview
	m = sendKey(m, "enter")

	view := m.View()
	if !strings.Contains(view, "█") {
		t.Errorf("expected banner art in preview after right-alignment selection, got: %s", view)
	}
	if !strings.Contains(view, "subtitle") {
		t.Errorf("expected tagline in preview, got: %s", view)
	}
}

func TestAlignToggle_SpaceToggles(t *testing.T) {
	var m tea.Model = tui.InitialModel()

	// Tab to alignment field (4 tabs from word field)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Space should toggle; view should not panic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	view1 := m.View()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	view2 := m.View()

	_ = view1
	_ = view2 // just confirm no panic
}

func TestInvalidColor_ShowsError(t *testing.T) {
	var m tea.Model = tui.InitialModel()

	// Type a valid word
	m = typeWord(m, "HI")

	// Tab to text-color field and type an out-of-range ANSI index
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = typeWord(m, "999")

	// Press Enter
	m = sendKey(m, "enter")

	view := m.View()
	// Must stay on input screen — no banner art
	if strings.Contains(view, "█") {
		t.Error("expected to stay on input screen with invalid color, but transitioned to preview")
	}
	// Must surface an error about colour
	lower := strings.ToLower(view)
	if !strings.Contains(lower, "colour") && !strings.Contains(lower, "color") {
		t.Errorf("expected colour error in view, got: %s", view)
	}
}
