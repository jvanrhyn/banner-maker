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
	input     textinput.Model
	banner    string
	errMsg    string
	savedPath string
}

// InitialModel returns a freshly initialised Model ready to run.
func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a word…"
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40

	return Model{
		screen: screenInput,
		input:  ti,
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
	case tea.KeyEnter:
		word := strings.TrimSpace(m.input.Value())
		if word == "" {
			m.errMsg = "Please enter at least one character."
			return m, nil
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

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.errMsg = ""
	return m, cmd
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, doSave(m.banner)
	case "n", "esc":
		m.screen = screenInput
		m.input.SetValue("")
		m.input.Focus()
		m.errMsg = ""
		return m, textinput.Blink
	}
	return m, nil
}

// doSave is an async Cmd that writes the banner to disk.
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
	sb.WriteString("  Enter a word to generate block-character art:\n\n")
	sb.WriteString("  " + m.input.View() + "\n\n")

	if m.errMsg != "" {
		sb.WriteString("  " + errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  enter: generate • ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewPreview() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Preview") + "\n\n")

	bannerContent := strings.TrimRight(m.banner, "\n ")
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
	sb.WriteString(m.banner + "\n")
	sb.WriteString(helpStyle.Render("  Press any key to exit"))
	return docStyle.Render(sb.String())
}
