package tui

import (
	"errors"
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

// Tab stops, in focus order. fieldAlign and fieldEmbed are toggle widgets and
// have no textinput behind them.
const (
	fieldWord = iota
	fieldTextColor
	fieldShadowColor
	fieldTagline
	fieldTaglineColor
	fieldAlign // toggle
	fieldPackage
	fieldEmbed // toggle
	fieldOutput
	tabStops // total number of tab stops
)

// Slots in Model.inputs. These differ from tab stops because toggles don't
// occupy a slot; use inputIdx to convert.
const (
	slotWord = iota
	slotTextColor
	slotShadowColor
	slotTagline
	slotTaglineColor
	slotPackage
	slotOutput
	inputCount
)

// tabInput maps each tab stop to its inputs slot, or -1 for toggles.
var tabInput = [tabStops]int{
	fieldWord:         slotWord,
	fieldTextColor:    slotTextColor,
	fieldShadowColor:  slotShadowColor,
	fieldTagline:      slotTagline,
	fieldTaglineColor: slotTaglineColor,
	fieldAlign:        -1,
	fieldPackage:      slotPackage,
	fieldEmbed:        -1,
	fieldOutput:       slotOutput,
}

// inputIdx maps a tab-stop focusIdx to the corresponding inputs array index.
// Returns -1 if focusIdx is a toggle (fieldAlign, fieldEmbed).
func inputIdx(focusIdx int) int { return tabInput[focusIdx] }

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

	focusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
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
	inputs    [inputCount]textinput.Model
	focusIdx  int
	align     string // "left" or "right"
	embed     bool   // also generate the {Word}BannerColor() func
	word      string // the input word (used for filename + Go identifier)
	banner    string // raw figlet output
	errMsg    string
	savedPath string
	// confirmOverwrite is set after a save hit an existing file; the next
	// save attempt then replaces it.
	confirmOverwrite bool
}

// InitialModel returns a freshly initialised Model ready to run.
func InitialModel() Model {
	defaults := banner.DefaultColors()

	newInput := func(placeholder string, charLimit, width int) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = charLimit
		ti.Width = width
		return ti
	}

	var inputs [inputCount]textinput.Model
	inputs[slotWord] = newInput("Enter a word…", 50, 40)
	inputs[slotWord].Focus()
	inputs[slotTextColor] = newInput(defaults.TextColor+"  (default)", 20, 30)
	inputs[slotShadowColor] = newInput(defaults.ShadowColor+"  (default)", 20, 30)
	inputs[slotTagline] = newInput("Optional tagline text…", 80, 40)
	inputs[slotTaglineColor] = newInput("none  (default)", 20, 30)
	inputs[slotPackage] = newInput("main  (default)", 60, 30)
	inputs[slotOutput] = newInput("{word}banner.go  (default)", 120, 40)

	return Model{
		screen:   screenInput,
		inputs:   inputs,
		focusIdx: fieldWord,
		align:    "left",
	}
}

// value returns the trimmed contents of an input slot.
func (m Model) value(slot int) string { return strings.TrimSpace(m.inputs[slot].Value()) }

// colorOpts builds a ColorOptions from the current input values, falling back
// to defaults for any empty fields.
func (m Model) colorOpts() banner.ColorOptions {
	return banner.ColorOptions{
		TextColor:    m.value(slotTextColor),
		ShadowColor:  m.value(slotShadowColor),
		TaglineColor: m.value(slotTaglineColor),
	}
}

// taglineOpts builds a Tagline from the current input values.
func (m Model) taglineOpts() banner.Tagline {
	return banner.Tagline{Text: m.value(slotTagline), Align: m.align}
}

// sourceOpts builds the generated-file options from the current form state.
func (m Model) sourceOpts(overwrite bool) banner.SourceOptions {
	return banner.SourceOptions{
		Package:    m.value(slotPackage),
		EmbedColor: m.embed,
		Colors:     m.colorOpts(),
		Overwrite:  overwrite,
	}
}

// validate checks every form field and returns the first problem found.
func (m Model) validate() error {
	if m.value(slotWord) == "" {
		return errors.New("Please enter at least one character.")
	}
	for _, c := range []struct {
		label string
		slot  int
	}{
		{"Text colour", slotTextColor},
		{"Shadow colour", slotShadowColor},
		{"Tagline colour", slotTaglineColor},
	} {
		if err := banner.ValidateColor(m.value(c.slot)); err != nil {
			return fmt.Errorf("%s: %w", c.label, err)
		}
	}
	if err := banner.ValidatePackage(m.value(slotPackage)); err != nil {
		return fmt.Errorf("Package: %w", err)
	}
	return nil
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
		if errors.Is(msg.err, banner.ErrFileExists) {
			m.confirmOverwrite = true
			m.errMsg = fmt.Sprintf("%v — press y again to overwrite, n to cancel", msg.err)
			return m, nil
		}
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
		active := inputIdx(m.focusIdx)
		for i := range m.inputs {
			if i == active {
				m.inputs[i].Focus()
			} else {
				m.inputs[i].Blur()
			}
		}
		// When focusIdx == fieldAlign, active == -1 so all textinputs are blurred (correct)
		return m, textinput.Blink

	case tea.KeyEnter:
		if err := m.validate(); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		word := m.value(slotWord)
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

	// Toggle widgets have no textinput; arrows pick a side, space flips it.
	switch m.focusIdx {
	case fieldAlign:
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
	case fieldEmbed:
		switch msg.Type {
		case tea.KeyLeft:
			m.embed = false
		case tea.KeyRight:
			m.embed = true
		case tea.KeySpace:
			m.embed = !m.embed
		}
		return m, nil
	}

	// Route other keystrokes to the focused textinput
	if idx := inputIdx(m.focusIdx); idx >= 0 {
		var cmd tea.Cmd
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
		m.errMsg = ""
		return m, cmd
	}
	return m, nil
}

func (m Model) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, doSave(m.word, m.banner, m.taglineOpts(), m.value(slotOutput), m.sourceOpts(m.confirmOverwrite))
	case "n", "esc":
		m.confirmOverwrite = false
		m.screen = screenInput
		m.inputs[slotWord].SetValue("")
		for i := range m.inputs {
			if i == slotWord {
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
// outputPath overrides the default {word}banner.go filename if non-empty.
// An existing file is only replaced when opts.Overwrite is true.
func doSave(word, rawBanner string, tag banner.Tagline, outputPath string, opts banner.SourceOptions) tea.Cmd {
	return func() tea.Msg {
		path := outputPath
		if path == "" {
			path = banner.DefaultOutputPath(word)
		}
		if err := banner.SaveGoFileWithOptions(word, rawBanner, tag, path, opts); err != nil {
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

// renderToggle renders a two-choice toggle, highlighting the active side.
func renderToggle(left, right string, leftActive bool) string {
	leftS, rightS := toggleOffStyle, toggleOffStyle
	if leftActive {
		leftS = toggleOnStyle
	} else {
		rightS = toggleOnStyle
	}
	return leftS.Render("◀ "+left) + "  " + rightS.Render(right+" ▶")
}

// row renders one form line: a focus marker, a padded label and the control.
func (m Model) row(field int, label, control string) string {
	marker := "  "
	if m.focusIdx == field {
		marker = focusStyle.Render("▸ ")
	}
	return marker + labelStyle.Render(fmt.Sprintf("%-18s", label)) + control + "\n"
}

func (m Model) viewInput() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Maker") + "\n\n")
	sb.WriteString(m.row(fieldWord, "Word", m.inputs[slotWord].View()))
	sb.WriteString(m.row(fieldTextColor, "Text colour", m.inputs[slotTextColor].View()))
	sb.WriteString(m.row(fieldShadowColor, "Shadow colour", m.inputs[slotShadowColor].View()))
	sb.WriteString(m.row(fieldTagline, "Tagline", m.inputs[slotTagline].View()))
	sb.WriteString(m.row(fieldTaglineColor, "Tagline colour", m.inputs[slotTaglineColor].View()))
	sb.WriteString(m.row(fieldAlign, "Tagline align", renderToggle("left", "right", m.align == "left")))
	sb.WriteString(m.row(fieldPackage, "Package", m.inputs[slotPackage].View()))
	sb.WriteString(m.row(fieldEmbed, "Embed colour", renderToggle("off", "on", !m.embed)))
	sb.WriteString(m.row(fieldOutput, "Output path", m.inputs[slotOutput].View()))
	sb.WriteString("\n")

	if m.errMsg != "" {
		sb.WriteString("  " + errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  colours: hex (#RGB, #RRGGBB) or 0–255  •  leave optional fields blank for defaults") + "\n")
	sb.WriteString(helpStyle.Render("  tab/shift+tab: move  •  ← → space: toggles  •  enter: generate  •  ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewPreview() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Preview") + "\n\n")

	// Display with colors and tagline; saved file contains the same content
	colorized := banner.ColorizeWithTagline(m.banner, m.taglineOpts(), m.colorOpts())
	bannerContent := strings.TrimRight(colorized, "\n ")
	boxWidth := m.width - 8
	if boxWidth < 40 {
		boxWidth = 40
	}

	box := bannerBoxStyle.Width(boxWidth).Render(bannerContent)
	sb.WriteString(box + "\n\n")

	if m.embed {
		_, funcName := banner.GoIdent(m.word)
		sb.WriteString(helpStyle.Render("  Also generating "+funcName+"Color() with ANSI colours embedded") + "\n\n")
	}

	if m.errMsg != "" {
		sb.WriteString(errorStyle.Render("⚠ "+m.errMsg) + "\n\n")
	}

	sb.WriteString(helpStyle.Render("  y / enter: save to {word}banner.go  •  n / esc: try again  •  ctrl+c: quit"))
	return docStyle.Render(sb.String())
}

func (m Model) viewDone() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Banner Maker") + "\n\n")
	sb.WriteString(successStyle.Render(fmt.Sprintf("  ✓ Saved to %s", m.savedPath)) + "\n\n")
	// Show colorized banner with tagline on the done screen too
	sb.WriteString(banner.ColorizeWithTagline(m.banner, m.taglineOpts(), m.colorOpts()) + "\n")
	sb.WriteString(helpStyle.Render("  Press any key to exit"))
	return docStyle.Render(sb.String())
}
