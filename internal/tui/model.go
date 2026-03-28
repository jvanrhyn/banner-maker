package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jvanrhyn/banner-maker/internal/banner"
)

// screen identifies which view is active.
type screen int

const (
	screenInput screen = iota
	screenPreview
	screenDone
)

// inputField indices
const (
	fieldWord        = 0
	fieldTextColor   = 1
	fieldShadowColor = 2
	fieldCount       = 3
)

const saveFile = "banner.txt"

// ─── Styles ─────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	bannerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true)

	docStyle = lipgloss.NewStyle().Padding(1, 2)
)

// ─── Messages ────────────────────────────────────────────────────────────────

type savedMsg struct{ path string }
type saveErrMsg struct{ err error }

// ─── Model ───────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for banner-maker.
type Model struct {
	screen    screen
	width     int
	height    int
	inputs    [fieldCount]textinput.Model
	focusIdx  int
	banner    string // raw text (saved to file)
	errMsg    string
	savedPath string
}

// InitialModel returns a freshly initialised Model ready to run.
func InitialModel() Model {
	defaults := banner.DefaultColors()

	word := textinput.New()
	word.Placeholder = "Enter a word…"
	word.Focus()
	word.CharLimit = 50
	word.Width = 40

	textColor := textinput.New()
	textColor.Placeholder = defaults.TextColor + "  (default)"
	textColor.CharLimit = 20
	textColor.Width = 30

	shadowColor := textinput.New()
	shadowColor.Placeholder = defaults.ShadowColor + "  (default)"
	shadowColor.CharLimit = 20
	shadowColor.Width = 30

	return Model{
		screen:   screenInput,
		inputs:   [fieldCount]textinput.Model{word, textColor, shadowColor},
		focusIdx: fieldWord,
	}
}

// colorOpts builds a ColorOptions from the current input values, falling back
// to defaults for any empty fields.
func (m Model) colorOpts() banner.ColorOptions {
	return banner.ColorOptions{
		TextColor:   strings.TrimSpace(m.inputs[fieldTextColor].Value()),
		ShadowColor: strings.TrimSpace(m.inputs[fieldShadowColor].Value()),
	}
}

// ─── Init ────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// ─── Update ──────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.screen {
		case screenInput:
			return m.updateInput(msg)
		case screenPreview:
			return m.updatePreview(msg)
		case screenDone:
			return m, tea.Quit
		}

	case savedMsg:
		m.savedPath = msg.path
		m.screen = screenDone
		return m, nil

	case saveErrMsg:
		m.errMsg = fmt.Sprintf("Save failed: %v", msg.err)
		return m, nil
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyTab, tea.KeyShiftTab:
		// Cycle focus between the three inputs
		if msg.Type == tea.KeyTab {
			m.focusIdx = (m.focusIdx + 1) % fieldCount
		} else {
			m.focusIdx = (m.focusIdx - 1 + fieldCount) % fieldCount
		}
		for i := range m.inputs {
			if i == m.focusIdx {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		return m, textinput.Blink

	case tea.KeyEnter:
		word := strings.TrimSpace(m.inputs[fieldWord].Value())
		if word == "" {
			m.errMsg = "Please enter at least one character."
			return m, nil
		}
		opts := m.colorOpts()
		if opts.TextColor != "" {
			if err := banner.ValidateColor(opts.TextColor); err != nil {
				m.errMsg = "Text colour: " + err.Error()
				return m, nil
			}
		}
		if opts.ShadowColor != "" {
			if err := banner.ValidateColor(opts.ShadowColor); err != nil {
				m.errMsg = "Shadow colour: " + err.Error()
				return m, nil
			}
		}
		art, err := banner.Generate(word)
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.errMsg = ""
		m.banner = art
		m.screen = screenPreview
		return m, nil
	}

	// Route keystrokes to the focused input
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	m.errMsg = ""
	return m, cmd
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, doSave(m.banner)
	case "n", "esc":
		m.screen = screenInput
		m.inputs[fieldWord].SetValue("")
		for i := range m.inputs {
			if i == fieldWord {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		m.focusIdx = fieldWord
		m.errMsg = ""
		return m, textinput.Blink
	}
	return m, nil
}

// doSave is an async Cmd that writes the raw banner to disk.
func doSave(content string) tea.Cmd {
	return func() tea.Msg {
		if err := banner.Save(content, saveFile); err != nil {
			return saveErrMsg{err}
		}
		return savedMsg{path: saveFile}
	}
}

// ─── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.screen {
	case screenInput:
		return m.viewInput()
	case screenPreview:
		return m.viewPreview()
	case screenDone:
		return m.viewDone()
	}
	return ""
}

func (m Model) viewInput() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Maker") + "\n\n")
	sb.WriteString(labelStyle.Render("  Word") + "\n")
	sb.WriteString("  " + m.inputs[fieldWord].View() + "\n\n")
	sb.WriteString(labelStyle.Render("  Text colour  (hex or 256-colour code)") + "\n")
	sb.WriteString("  " + m.inputs[fieldTextColor].View() + "\n\n")
	sb.WriteString(labelStyle.Render("  Shadow colour  (hex or 256-colour code)") + "\n")
	sb.WriteString("  " + m.inputs[fieldShadowColor].View() + "\n\n")

	if m.errMsg != "" {
		sb.WriteString("  " + errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  tab: next field • enter: generate • ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewPreview() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Preview") + "\n\n")

	// Display with colors; save file will contain the raw version
	colorized := banner.Colorize(m.banner, m.colorOpts())
	bannerContent := strings.TrimRight(colorized, "\n ")
	boxWidth := m.width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}

	box := bannerBoxStyle.Width(boxWidth).Render(bannerContent)
	sb.WriteString(box + "\n\n")

	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  y / enter: save to banner.txt  •  n / esc: try again  •  ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewDone() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Maker") + "\n\n")
	sb.WriteString(successStyle.Render(fmt.Sprintf("  ✓ Saved to %s", m.savedPath)) + "\n\n")
	// Show colorized version on the done screen too
	sb.WriteString(banner.Colorize(m.banner, m.colorOpts()) + "\n")
	sb.WriteString(helpStyle.Render("  Press any key to exit"))
	return docStyle.Render(sb.String())
}

