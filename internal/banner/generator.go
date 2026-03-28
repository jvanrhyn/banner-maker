package banner

import (
	_ "embed"
	"errors"
	"fmt"
	"go/format"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/mbndr/figlet4go"
)

//go:embed fonts/ansi-shadow.flf
var ansiFontBytes []byte

const fontName = "ANSI Shadow"

// ErrEmptyWord is returned when an empty word is passed to Generate.
var ErrEmptyWord = errors.New("empty word: please provide at least one character")

// shadowRunes is the set of box-drawing characters used as the drop-shadow in
// the ANSI Shadow figlet font.
var shadowRunes = map[rune]bool{
	'╗': true, '╔': true, '╝': true, '╚': true, '═': true, '║': true,
}

// ColorOptions controls the foreground colors applied when displaying a banner.
// Empty strings fall back to the defaults returned by DefaultColors.
type ColorOptions struct {
	TextColor   string // color for █ body characters  (default "#FFFFFF")
	ShadowColor string // color for box-drawing shadow chars (default "#555555")
}

// DefaultColors returns the out-of-the-box color scheme.
func DefaultColors() ColorOptions {
	return ColorOptions{
		TextColor:   "#FFFFFF",
		ShadowColor: "#555555",
	}
}

// Generate renders word as ANSI Shadow block art and returns the result as a
// multi-line string. Input is normalised to uppercase before rendering.
func Generate(word string) (string, error) {
	if strings.TrimSpace(word) == "" {
		return "", ErrEmptyWord
	}

	ascii := figlet4go.NewAsciiRender()
	if err := ascii.LoadBindataFont(ansiFontBytes, fontName); err != nil {
		return "", err
	}

	opts := figlet4go.NewRenderOptions()
	opts.FontName = fontName

	result, err := ascii.RenderOpts(strings.ToUpper(word), opts)
	if err != nil {
		return "", err
	}

	return result, nil
}

// ValidateColor reports whether s is an acceptable lipgloss color value.
// Valid inputs are:
//   - an empty string (the caller's default will be used)
//   - a hex color in #RGB or #RRGGBB form
//   - a decimal ANSI-256 index in the range 0–255
func ValidateColor(s string) error {
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "#") {
		hex := s[1:]
		if len(hex) != 3 && len(hex) != 6 {
			return fmt.Errorf("invalid hex color %q: must be #RGB or #RRGGBB", s)
		}
		for _, c := range hex {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return fmt.Errorf("invalid hex color %q: contains non-hex characters", s)
			}
		}
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return fmt.Errorf("invalid color %q: must be a hex color (#RRGGBB or #RGB) or an ANSI-256 index (0–255)", s)
	}
	return nil
}

// Colorize applies ANSI color codes to a raw banner string.
// '█' characters receive TextColor; box-drawing shadow characters receive
// ShadowColor. All other runes (spaces, newlines) are passed through unchanged.
// Empty color fields in opts fall back to DefaultColors.
func Colorize(raw string, opts ColorOptions) string {
	defaults := DefaultColors()

	textColor := defaults.TextColor
	if opts.TextColor != "" {
		textColor = opts.TextColor
	}
	shadowColor := defaults.ShadowColor
	if opts.ShadowColor != "" {
		shadowColor = opts.ShadowColor
	}

	// Guard against invalid user input — fall back to defaults rather than panic.
	if ValidateColor(textColor) != nil {
		textColor = defaults.TextColor
	}
	if ValidateColor(shadowColor) != nil {
		shadowColor = defaults.ShadowColor
	}

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(textColor))
	shadowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(shadowColor))

	var sb strings.Builder
	for _, r := range raw {
		switch {
		case r == '█':
			sb.WriteString(textStyle.Render(string(r)))
		case shadowRunes[r]:
			sb.WriteString(shadowStyle.Render(string(r)))
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Save writes content to the file at path, creating or truncating as needed.
func Save(content, path string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// GoIdent derives safe Go identifiers from word.
// varName is the []string variable name (e.g. "mycliLogo").
// funcName is the exported function name (e.g. "MycliBanner").
func GoIdent(word string) (varName, funcName string) {
	// Keep only alphanumeric, lowercase the whole thing
	var sb strings.Builder
	for _, r := range strings.ToLower(word) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	slug := sb.String()
	if slug == "" {
		slug = "banner"
	}
	// Ensure slug starts with a letter (prepend 'b' if it starts with a digit)
	if unicode.IsDigit([]rune(slug)[0]) {
		slug = "b" + slug
	}
	varName = slug + "Logo"
	runes := []rune(slug)
	funcName = string(unicode.ToUpper(runes[0])) + string(runes[1:]) + "Banner"
	return
}

// GenerateGoSource returns a gofmt-formatted Go source file containing:
//   - a var {word}Logo = []string{...} with each line of the banner+tagline
//   - a func {Word}Banner() string that joins and returns the lines
func GenerateGoSource(word, rawBanner string, tag Tagline) (string, error) {
	full := AppendTagline(rawBanner, tag)

	// Collect non-empty content lines
	var lines []string
	for _, l := range strings.Split(strings.TrimRight(full, "\n"), "\n") {
		lines = append(lines, l)
	}

	varName, funcName := GoIdent(word)

	// Build the []string literal entries
	var entries strings.Builder
	for _, l := range lines {
		entries.WriteString(fmt.Sprintf("\t\t%q,\n", l))
	}

	src := fmt.Sprintf(`package main

import "strings"

var %s = []string{
%s}

// %s returns the banner art as a single string.
func %s() string {
	return strings.Join(%s, "\n")
}
`, varName, entries.String(), funcName, funcName, varName)

	formatted, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("go/format: %w", err)
	}
	return string(formatted), nil
}

// SaveGoFile generates a Go source file for the given banner and writes it to path.
func SaveGoFile(word, rawBanner string, tag Tagline, path string) error {
	src, err := GenerateGoSource(word, rawBanner, tag)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(src), 0o644)
}

// Tagline represents an optional line of text appended below a banner.
type Tagline struct {
	Text  string // empty means no tagline
	Align string // "left" (default) or "right"; case-insensitive
}

// ValidateTaglineAlign reports whether s is a valid alignment value.
// Valid values are "" (defaults to "left"), "left", and "right".
func ValidateTaglineAlign(s string) error {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "left", "right":
		return nil
	default:
		return fmt.Errorf("invalid alignment %q: must be \"left\" or \"right\"", s)
	}
}

// AppendTagline appends a blank separator line and the tagline text to raw.
// If t.Text is empty, raw is returned unchanged.
// Right alignment is calculated relative to the widest line in the raw banner.
func AppendTagline(raw string, t Tagline) string {
	if strings.TrimSpace(t.Text) == "" {
		return raw
	}

	maxWidth := 0
	for _, line := range strings.Split(raw, "\n") {
		if w := len([]rune(line)); w > maxWidth {
			maxWidth = w
		}
	}

	tagline := t.Text
	if strings.EqualFold(strings.TrimSpace(t.Align), "right") {
		if pad := maxWidth - len([]rune(tagline)); pad > 0 {
			tagline = strings.Repeat(" ", pad) + tagline
		}
	}

	return strings.TrimRight(raw, "\n ") + "\n" + tagline + "\n"
}
