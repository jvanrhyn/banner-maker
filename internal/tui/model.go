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
	fieldTagline     = 3
	fieldAlign       = 4  // not a textinput — rendered as a toggle
	inputCount       = 4  // number of textinput fields (0..3)
	tabStops         = 5  // total tab positions (0..4)
)

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

	toggleOnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	toggleOffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

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
	inputs    [inputCount]textinput.Model
	focusIdx  int
	align     string // "left" or "right"
	word      string // the input word (used for filename + Go identifier)
	banner    string // raw figlet output
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

	taglineInput := textinput.New()
	taglineInput.Placeholder = "Optional tagline text…"
	taglineInput.CharLimit = 80
	taglineInput.Width = 60

	return Model{
		screen:   screenInput,
		inputs:   [inputCount]textinput.Model{word, textColor, shadowColor, taglineInput},
		focusIdx: fieldWord,
		align:    "left",
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

// taglineOpts builds a Tagline from the current input values.
func (m Model) taglineOpts() banner.Tagline {
	return banner.Tagline{
		Text:  strings.TrimSpace(m.inputs[fieldTagline].Value()),
		Align: m.align,
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
		if msg.Type == tea.KeyTab {
			m.focusIdx = (m.focusIdx + 1) % tabStops
		} else {
			m.focusIdx = (m.focusIdx - 1 + tabStops) % tabStops
		}
		for i := range m.inputs {
			if i == m.focusIdx {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		// When focusIdx == fieldAlign, all textinputs are blurred (correct)
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
		m.word = word
		m.banner = art
		m.screen = screenPreview
		return m, nil
	}

	// Handle alignment toggle when that field is focused
	if m.focusIdx == fieldAlign {
		switch msg.Type {
		case tea.KeyLeft:
			m.align = "left"
		case tea.KeyRight:
			m.align = "right"
		case tea.KeySpace:
			if m.align == "left" {
				m.align = "right"
			} else {
				m.align = "left"
			}
		}
		return m, nil
	}

	// Route other keystrokes to the focused textinput
	var cmd tea.Cmd
	m.inputs[m.focusIdx], cmd = m.inputs[m.focusIdx].Update(msg)
	m.errMsg = ""
	return m, cmd
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, doSave(m.word, m.banner, m.taglineOpts())
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

// doSave is an async Cmd that generates a Go source file and writes it to disk.
func doSave(word, rawBanner string, tag banner.Tagline) tea.Cmd {
	return func() tea.Msg {
		varName, _ := banner.GoIdent(word)
		// Strip the "Logo" suffix to get the slug for the filename
		slug := strings.TrimSuffix(varName, "Logo")
		path := slug + "banner.go"
		if err := banner.SaveGoFile(word, rawBanner, tag, path); err != nil {
			return saveErrMsg{err}
		}
		return savedMsg{path: path}
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

// renderAlignToggle renders the left/right toggle, highlighting the active choice.
func (m Model) renderAlignToggle() string {
	leftS, rightS := toggleOffStyle, toggleOffStyle
	if m.align == "left" {
		leftS = toggleOnStyle
	} else {
		rightS = toggleOnStyle
	}
	return leftS.Render("◀ left") + "  " + rightS.Render("right ▶")
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
	sb.WriteString(labelStyle.Render("  Tagline  (optional)") + "\n")
	sb.WriteString("  " + m.inputs[fieldTagline].View() + "\n\n")
	sb.WriteString(labelStyle.Render("  Tagline alignment") + "\n")
	sb.WriteString("  " + m.renderAlignToggle() + "\n\n")

	if m.errMsg != "" {
		sb.WriteString("  " + errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  tab: next field  •  ← →: alignment  •  enter: generate  •  ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewPreview() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Preview") + "\n\n")

	// Display with colors and tagline; saved file contains the same content
	colorized := banner.Colorize(banner.AppendTagline(m.banner, m.taglineOpts()), m.colorOpts())
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
	// Show colorized banner with tagline on the done screen too
	sb.WriteString(banner.Colorize(banner.AppendTagline(m.banner, m.taglineOpts()), m.colorOpts()) + "\n")
	sb.WriteString(helpStyle.Render("  Press any key to exit"))
	return docStyle.Render(sb.String())
}

