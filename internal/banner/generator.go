package banner

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
