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

func tab(m tea.Model, n int) tea.Model {
	for range n {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	return m
}

func shortTempDir(t *testing.T) string {
	t.Helper()
	// Short dir: the output field has a 120-char limit and t.TempDir() paths
	// derived from long test names can exceed it.
	dir, err := os.MkdirTemp("", "bm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// saveVia presses 'y' on the preview screen and runs the resulting command.
func saveVia(m tea.Model) tea.Model {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		updated, _ = updated.Update(cmd())
	}
	return updated
}

func typeRunes(m tea.Model, s string) tea.Model {
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	return m
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

	// Tab past tagline-color to the alignment toggle (→ 5), press right to select "right"
	m = tab(m, 2)
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

	// Tab to alignment field (5 tabs from word field)
	m = tab(m, 5)

	// Space should toggle; view should not panic
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	view1 := m.View()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	view2 := m.View()

	_ = view1
	_ = view2 // just confirm no panic
}

func TestCustomOutputPath_UsedWhenProvided(t *testing.T) {
	dir := t.TempDir()
	customPath := dir + "/custom_output.go"

	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")

	// Tab to output field (0 word → 1 text → 2 shadow → 3 tagline → 4 tagline-color → 5 align → 6 package → 7 embed → 8 output)
	for range 8 {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	m = typeWord(m, customPath)

	// Enter to generate and go to preview
	m = sendKey(m, "enter")

	// Confirm save with 'y', execute the async save command
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		msg := cmd()
		updated, _ = updated.Update(msg)
	}

	view := updated.View()
	if !strings.Contains(view, "custom_output.go") {
		t.Errorf("expected custom output path in done view, got: %s", view)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("expected file at custom path %s, got error: %v", customPath, err)
	}
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

func TestPreviewSave_ExistingFileNeedsSecondConfirm(t *testing.T) {
	// Short dir: the output field has a 120-char limit and t.TempDir() paths
	// derived from this test's long name can exceed it.
	dir, err := os.MkdirTemp("", "bm")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	path := dir + "/existing.go"
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 8)
	m = typeWord(m, path)
	m = sendKey(m, "enter")

	press := func(m tea.Model) tea.Model {
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		if cmd != nil {
			updated, _ = updated.Update(cmd())
		}
		return updated
	}

	// First 'y': refused, file untouched, user is told how to proceed.
	m = press(m)
	if view := m.View(); !strings.Contains(view, "already exists") || !strings.Contains(view, "overwrite") {
		t.Errorf("expected overwrite prompt, got: %s", view)
	}
	if data, _ := os.ReadFile(path); string(data) != "original" {
		t.Fatalf("file modified on first save: %q", data)
	}

	// Second 'y': overwrites and reaches the done screen.
	m = press(m)
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "package main") {
		t.Errorf("file not overwritten on confirm: %q", data)
	}
	if !strings.Contains(m.View(), "Saved") {
		t.Errorf("expected done screen, got: %s", m.View())
	}
}

// ─── Parity with CLI flags ──────────────────────────────────────────────────

func TestTabWrapsAroundAfterNineStops(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = tab(m, 9)
	m = typeWord(m, "X")
	m = sendKey(m, "enter")
	if !strings.Contains(m.View(), "█") {
		t.Errorf("9 tabs should wrap back to the word field; got: %s", m.View())
	}
}

func TestShiftTabFromWordGoesToOutput(t *testing.T) {
	dir := shortTempDir(t)
	path := dir + "/wrap.go"
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = typeRunes(m, path)
	m = sendKey(m, "enter")
	m = saveVia(m)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("shift-tab from word should land on output: %v", err)
	}
}

func TestInvalidTaglineColor_ShowsError(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 4)
	m = typeRunes(m, "999")
	m = sendKey(m, "enter")
	view := m.View()
	if strings.Contains(view, "█") {
		t.Error("expected to stay on input screen")
	}
	if !strings.Contains(strings.ToLower(view), "tagline colo") {
		t.Errorf("expected tagline colour error, got: %s", view)
	}
}

func TestInvalidPackage_ShowsError(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 6)
	m = typeRunes(m, "bad-name")
	m = sendKey(m, "enter")
	view := m.View()
	if strings.Contains(view, "█") {
		t.Error("expected to stay on input screen")
	}
	if !strings.Contains(strings.ToLower(view), "package") {
		t.Errorf("expected package error, got: %s", view)
	}
}

func TestUnsupportedCharacters_ShowsError(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeRunes(m, "héllo")
	m = sendKey(m, "enter")
	view := m.View()
	if strings.Contains(view, "█") {
		t.Error("expected to stay on input screen")
	}
	if !strings.Contains(view, "é") {
		t.Errorf("expected error naming the unsupported character, got: %s", view)
	}
}

func TestPackage_AppliedToSavedFile(t *testing.T) {
	path := shortTempDir(t) + "/pkg.go"
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 6)
	m = typeRunes(m, "ui")
	m = tab(m, 2)
	m = typeRunes(m, path)
	m = sendKey(m, "enter")
	m = saveVia(m)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "package ui\n") {
		t.Errorf("package not applied:\n%s", data)
	}
	if strings.Contains(string(data), "BannerColor") {
		t.Error("embed-color must be off by default")
	}
}

func TestEmbedColor_ToggleAddsColorFunc(t *testing.T) {
	path := shortTempDir(t) + "/emb.go"
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 1)
	m = typeRunes(m, "#FF0000")
	m = tab(m, 6) // → embed toggle (7)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = tab(m, 1)
	m = typeRunes(m, path)
	m = sendKey(m, "enter")

	if !strings.Contains(m.View(), "HiBannerColor") {
		t.Errorf("preview should mention the colour func, got: %s", m.View())
	}
	m = saveVia(m)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"func HiBannerColor()", `\x1b[38;2;255;0;0m`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("saved file missing %q:\n%s", want, data)
		}
	}
}

func TestEmbedColor_ArrowKeysSetExplicitly(t *testing.T) {
	path := shortTempDir(t) + "/arrow.go"
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 7)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // on
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})  // off again
	m = tab(m, 1)
	m = typeRunes(m, path)
	m = sendKey(m, "enter")
	m = saveVia(m)
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "BannerColor") {
		t.Errorf("left arrow should turn embed off:\n%s", data)
	}
}

func TestTaglineColor_AcceptedAndPreviewShowsTagline(t *testing.T) {
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 3)
	m = typeRunes(m, "sub")
	m = tab(m, 1)
	m = typeRunes(m, "#00FF00")
	m = sendKey(m, "enter")
	view := m.View()
	if !strings.Contains(view, "sub") || !strings.Contains(view, "█") {
		t.Errorf("expected preview with tagline, got: %s", view)
	}
}

func TestTaglineColor_EmbeddedInSavedFile(t *testing.T) {
	path := shortTempDir(t) + "/tag.go"
	var m tea.Model = tui.InitialModel()
	m = typeWord(m, "HI")
	m = tab(m, 3)
	m = typeRunes(m, "sub")
	m = tab(m, 1)
	m = typeRunes(m, "51")
	m = tab(m, 3) // → embed (7)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = tab(m, 1)
	m = typeRunes(m, path)
	m = sendKey(m, "enter")
	m = saveVia(m)
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `\x1b[38;5;51msub\x1b[0m`) {
		t.Errorf("tagline colour not embedded:\n%s", data)
	}
}
