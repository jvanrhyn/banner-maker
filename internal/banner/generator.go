package banner

import (
	_ "embed"
	"errors"
	"fmt"
	"go/format"
	"go/token"
	"io/fs"
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

// ErrUnsupportedChars is returned when a word contains characters the ANSI
// Shadow font cannot render (anything other than printable ASCII).
var ErrUnsupportedChars = errors.New("unsupported characters (only printable ASCII is supported)")

// ErrFileExists is returned by SaveGoFileWithOptions when the target file
// already exists and SourceOptions.Overwrite is false.
var ErrFileExists = errors.New("file already exists")

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
	// TaglineColor colors the tagline text. Empty (or invalid) leaves the
	// tagline unstyled, which is also the default.
	TaglineColor string
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
	if err := ValidateWord(word); err != nil {
		return "", err
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

// Colorize applies terminal color styling (via lipgloss, so it honours the
// terminal's color profile, NO_COLOR and non-TTY output) to a raw banner string.
// '█' characters receive TextColor; box-drawing shadow characters receive
// ShadowColor. All other runes (spaces, newlines) are passed through unchanged.
// Empty color fields in opts fall back to DefaultColors.
func Colorize(raw string, opts ColorOptions) string {
	return colorize(raw, opts, lipglossPaint)
}

// ColorizeWithTagline is like Colorize but also appends the tagline (see
// AppendTagline) and styles it with opts.TaglineColor when that is set.
func ColorizeWithTagline(raw string, tag Tagline, opts ColorOptions) string {
	return colorizeWithTagline(raw, tag, opts, lipglossPaint)
}

// ColorizeANSI is like ColorizeWithTagline but always emits literal ANSI escape
// sequences (24-bit for hex colors, 256-color for indices), independent of the
// terminal. It is used to embed colors in generated Go source.
func ColorizeANSI(raw string, tag Tagline, opts ColorOptions) string {
	return colorizeWithTagline(raw, tag, opts, ansiPaint)
}

// paintFunc returns a function that wraps text in the given color.
type paintFunc func(color string) func(string) string

func lipglossPaint(color string) func(string) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return func(s string) string { return style.Render(s) }
}

func ansiPaint(color string) func(string) string {
	prefix := "\x1b[" + sgrForeground(color) + "m"
	return func(s string) string { return prefix + s + "\x1b[0m" }
}

// sgrForeground converts an already-validated color to an SGR parameter string.
func sgrForeground(color string) string {
	if hex, ok := strings.CutPrefix(color, "#"); ok {
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		v, _ := strconv.ParseUint(hex, 16, 32)
		return fmt.Sprintf("38;2;%d;%d;%d", v>>16&0xff, v>>8&0xff, v&0xff)
	}
	return "38;5;" + color
}

func colorize(raw string, opts ColorOptions, paint paintFunc) string {
	text, shadow, _ := resolveColors(opts)
	return paintRuns(raw, paint(text), paint(shadow))
}

func colorizeWithTagline(raw string, tag Tagline, opts ColorOptions, paint paintFunc) string {
	if strings.TrimSpace(tag.Text) == "" {
		return colorize(raw, opts, paint)
	}
	body := strings.TrimRight(raw, "\n ")
	tagLine := strings.TrimSuffix(AppendTagline(raw, tag)[len(body)+1:], "\n")

	_, _, tagColor := resolveColors(opts)
	if tagColor != "" {
		text := strings.TrimLeft(tagLine, " ")
		tagLine = tagLine[:len(tagLine)-len(text)] + paint(tagColor)(text)
	}
	return colorize(body, opts, paint) + "\n" + tagLine + "\n"
}

// resolveColors applies defaults and drops invalid values. The tagline color
// has no default: "" means unstyled.
func resolveColors(opts ColorOptions) (text, shadow, tagline string) {
	defaults := DefaultColors()

	text, shadow = defaults.TextColor, defaults.ShadowColor
	// Invalid user input falls back to defaults rather than panicking.
	if opts.TextColor != "" && ValidateColor(opts.TextColor) == nil {
		text = opts.TextColor
	}
	if opts.ShadowColor != "" && ValidateColor(opts.ShadowColor) == nil {
		shadow = opts.ShadowColor
	}
	if opts.TaglineColor != "" && ValidateColor(opts.TaglineColor) == nil {
		tagline = opts.TaglineColor
	}
	return text, shadow, tagline
}

// paintRuns styles consecutive runs of block / shadow runes with a single
// escape sequence each, leaving every other rune untouched.
func paintRuns(raw string, text, shadow func(string) string) string {
	const (
		other = iota
		block
		shade
	)
	var out, run strings.Builder
	class := other
	flush := func() {
		if run.Len() == 0 {
			return
		}
		switch s := run.String(); class {
		case block:
			out.WriteString(text(s))
		case shade:
			out.WriteString(shadow(s))
		default:
			out.WriteString(s)
		}
		run.Reset()
	}
	for _, r := range raw {
		c := other
		switch {
		case r == '█':
			c = block
		case shadowRunes[r]:
			c = shade
		}
		if c != class {
			flush()
			class = c
		}
		run.WriteRune(r)
	}
	flush()
	return out.String()
}

// ValidateWord reports whether every rune of word can be rendered by the font.
// Only printable ASCII (space through '~') is supported.
func ValidateWord(word string) error {
	var bad []string
	seen := map[rune]bool{}
	for _, r := range word {
		if (r < ' ' || r > '~') && !seen[r] {
			seen[r] = true
			bad = append(bad, strconv.QuoteRune(r))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("%w: %s", ErrUnsupportedChars, strings.Join(bad, ", "))
	}
	return nil
}

// ValidatePackage reports whether s is usable as a Go package name. An empty
// string is valid and means "main".
func ValidatePackage(s string) error {
	if s == "" {
		return nil
	}
	if s == "_" || !token.IsIdentifier(s) {
		return fmt.Errorf("invalid package name %q: must be a valid Go identifier and not a keyword", s)
	}
	return nil
}

// DefaultOutputPath returns the file name used when no output path is given.
func DefaultOutputPath(word string) string {
	varName, _ := GoIdent(word)
	return strings.TrimSuffix(varName, "Logo") + "banner.go"
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

// SourceOptions customises the generated Go source. The zero value produces
// `package main` with a plain (uncolored) banner and refuses to overwrite files.
type SourceOptions struct {
	Package    string       // package clause; "" means "main"
	EmbedColor bool         // also emit a {Word}BannerColor() func with ANSI colors baked in
	Colors     ColorOptions // colors used when EmbedColor is set
	Overwrite  bool         // SaveGoFileWithOptions: replace an existing file
}

// GenerateGoSource returns a gofmt-formatted Go source file containing:
//   - a var {word}Logo = []string{...} with each line of the banner+tagline
//   - a func {Word}Banner() string that joins and returns the lines
func GenerateGoSource(word, rawBanner string, tag Tagline) (string, error) {
	return GenerateGoSourceWithOptions(word, rawBanner, tag, SourceOptions{})
}

// GenerateGoSourceWithOptions is GenerateGoSource with a configurable package
// name and optional embedded ANSI color variant.
func GenerateGoSourceWithOptions(word, rawBanner string, tag Tagline, o SourceOptions) (string, error) {
	if err := ValidatePackage(o.Package); err != nil {
		return "", err
	}
	pkg := o.Package
	if pkg == "" {
		pkg = "main"
	}

	varName, funcName := GoIdent(word)

	var sb strings.Builder
	fmt.Fprintf(&sb, "package %s\n\nimport \"strings\"\n\n", pkg)
	writeBanner(&sb, varName, funcName, "returns the banner art as a single string.", AppendTagline(rawBanner, tag))
	if o.EmbedColor {
		writeBanner(&sb, varName+"Color", funcName+"Color",
			"returns the banner art with ANSI color escape sequences embedded.",
			ColorizeANSI(rawBanner, tag, o.Colors))
	}

	formatted, err := format.Source([]byte(sb.String()))
	if err != nil {
		return "", fmt.Errorf("go/format: %w", err)
	}
	return string(formatted), nil
}

// writeBanner emits `var varName = []string{...}` plus an accessor function.
func writeBanner(sb *strings.Builder, varName, funcName, doc, content string) {
	fmt.Fprintf(sb, "var %s = []string{\n", varName)
	for _, l := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		fmt.Fprintf(sb, "\t%q,\n", l)
	}
	fmt.Fprintf(sb, "}\n\n// %s %s\nfunc %s() string {\n\treturn strings.Join(%s, \"\\n\")\n}\n\n",
		funcName, doc, funcName, varName)
}

// SaveGoFile generates a Go source file for the given banner and writes it to
// path, replacing any existing file.
func SaveGoFile(word, rawBanner string, tag Tagline, path string) error {
	return SaveGoFileWithOptions(word, rawBanner, tag, path, SourceOptions{Overwrite: true})
}

// SaveGoFileWithOptions generates Go source and writes it to path. Unless
// o.Overwrite is set it fails with ErrFileExists instead of replacing an
// existing file. Nothing is written if the options are invalid.
func SaveGoFileWithOptions(word, rawBanner string, tag Tagline, path string, o SourceOptions) error {
	src, err := GenerateGoSourceWithOptions(word, rawBanner, tag, o)
	if err != nil {
		return err
	}
	if o.Overwrite {
		return os.WriteFile(path, []byte(src), 0o644)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%w: %s", ErrFileExists, path)
	}
	if err != nil {
		return err
	}
	if _, err := f.WriteString(src); err != nil {
		f.Close()
		os.Remove(path)
		return err
	}
	return f.Close()
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

// AppendTagline appends the tagline text directly after the banner content.
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
