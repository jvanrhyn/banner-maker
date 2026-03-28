package banner

import (
	_ "embed"
	"errors"
	"os"
	"strings"

	"github.com/mbndr/figlet4go"
)

//go:embed fonts/ansi-shadow.flf
var ansiFontBytes []byte

const fontName = "ANSI Shadow"

// ErrEmptyWord is returned when an empty word is passed to Generate.
var ErrEmptyWord = errors.New("empty word: please provide at least one character")

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

// Save writes content to the file at path, creating or truncating as needed.
func Save(content, path string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
