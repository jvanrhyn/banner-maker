# banner-maker

A Go CLI tool for generating ANSI Shadow block-character word art. Create beautiful ASCII banners interactively or via command line, then save them as reusable Go source files.

## Features

- **Interactive TUI** — Use Bubble Tea to design your banner with live preview
- **Beautiful Block Art** — ANSI Shadow figlet font with realistic drop-shadow effect
- **Customizable Colors** — Set text and shadow colors independently (hex or ANSI 256)
- **Optional Taglines** — Add left or right-aligned subtitles below banners
- **Generate Go Code** — Export as reusable Go source files with exported functions
- **Non-Interactive Mode** — Generate banners from CLI flags for scripting

## Installation

```bash
go install github.com/jvanrhyn/banner-maker@latest
```

Or build from source:

```bash
git clone https://github.com/jvanrhyn/banner-maker.git
cd banner-maker
go build .
```

## Usage

### Interactive Mode (Default)

Launch the TUI without arguments:

```bash
banner-maker
```

**TUI Controls:**
- **Tab** / **Shift+Tab** — Cycle between fields (word, text color, shadow color, tagline, alignment)
- **↑ / ↓ / Space** — Toggle tagline alignment (left/right)
- **Enter** — Submit word and preview the banner
- **y** — Confirm and save
- **n** — Reject and edit
- **Ctrl+C** — Quit

### Non-Interactive Mode

Generate a banner from command-line arguments:

```bash
banner-maker --word "MYCLI" --color 205 --shadow 60 --tagline "The best CLI" --align right
```

**Flags:**
- `--word` (required) — Text to render; enables non-interactive mode
- `--color` — Block/text color: hex (`#RGB` or `#RRGGBB`) or ANSI 0–255 (default: built-in color)
- `--shadow` — Drop-shadow color: hex or ANSI 0–255 (default: built-in color)
- `--tagline` — Optional subtitle text below the banner
- `--align` — Tagline alignment: `left` or `right` (default: `left`)
- `--output` — Output path for the generated `.go` file (default: `{word}banner.go`)

**Example:**

```bash
$ banner-maker --word "HELLO" --color "#FF00FF" --shadow "200"
```

Output:
- Displays the colored banner in the terminal
- Saves to `hellobanner.go` in the current directory
- The generated file contains a reusable Go function

### Generated Go Files

The tool generates self-contained Go source files. Example output for `--word "MYCLI"`:

```go
package main

import "strings"

var mycliLogo = []string{
	"███╗   ███╗██╗   ██╗ ██████╗██╗     ██╗",
	"████╗ ████║╚██╗ ██╔╝██╔════╝██║     ██║",
	"██╔████╔██║ ╚████╔╝ ██║     ██║     ██║",
	"██║╚██╔╝██║  ╚██╔╝  ██║     ██║     ██║",
	"██║ ╚═╝ ██║   ██║   ╚██████╗███████╗██║",
	"╚═╝     ╚═╝   ╚═╝    ╚═════╝╚══════╝╚═╝",
}

// MycliBanner returns the banner art as a single string.
func MycliBanner() string {
	return strings.Join(mycliLogo, "\n")
}
```

**Use it in your own project:**

```go
package main

import "fmt"

// Your generated banner file
// import "./banners"

func main() {
	fmt.Println(MycliBanner())
}
```

## Architecture

```
internal/
  banner/
    generator.go       — Core banner generation, colorization, Go code generation
    generator_test.go  — Unit tests (47 tests, 100% coverage)
    fonts/
      ansi-shadow.flf  — Embedded ANSI Shadow figlet font
  tui/
    model.go          — Bubble Tea model, TUI state, and views
    model_test.go     — TUI interaction tests
main.go               — CLI entry point (interactive/non-interactive modes)
```

### Key Components

**`banner.Generator(word string)`** — Generates raw ANSI Shadow art from figlet

**`banner.Colorize(raw string, opts ColorOptions)`** — Applies terminal colors to banner lines

**`banner.GenerateGoSource(word, rawBanner string, tag Tagline)`** — Produces `gofmt`-formatted Go code

**`banner.SaveGoFile(word, rawBanner string, tag Tagline, path string)`** — Writes the `.go` file to disk

**TUI Model** — Full Bubble Tea application with multi-screen flow:
- Input screen (word, colors, tagline, alignment)
- Preview screen (live rendering with formatting)
- Done screen (save confirmation)

## Color Format

**Hex colors:** `#RGB` (3 digits) or `#RRGGBB` (6 digits)
- Example: `#FF0000` (red), `#F0F` (magenta)

**ANSI 256 colors:** Integer 0–255
- Standard colors: 0–15 (black, red, green, yellow, blue, magenta, cyan, white)
- 216 extended colors: 16–231
- 24 grayscale: 232–255
- Example: `196` (bright red), `226` (yellow)

Empty strings use built-in defaults (purple text, dark blue shadow).

## Testing

Run all tests:

```bash
go test ./...
```

Run with coverage:

```bash
go test -cover ./...
```

47 unit tests cover:
- Banner generation and figlet parsing
- Color validation and ANSI code generation
- Go identifier sanitization and code formatting
- TUI state transitions and input handling
- Tagline rendering with alignment

## Requirements

- **Go** 1.24.2 or later
- **Terminal** with 256-color ANSI support (for full color output)

### Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components (text input)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [figlet4go](https://github.com/mbndr/figlet4go) — Figlet font rendering

## Development

### Debug Mode

Enable verbose TUI logging:

```bash
DEBUG=1 banner-maker
```

Logs go to `debug.log`.

### Project Structure

- TDD approach — tests written first, implementation follows
- Adversarial code review (gpt-5.3-codex) before merges
- 100% test coverage on business logic
- Clean separation: banner generation (testable package) vs. TUI (Bubble Tea integration)

## License

MIT — see [LICENSE.md](LICENSE.md)

## Contributing

Contributions welcome! Please ensure:
- Tests pass: `go test ./...`
- Build succeeds: `go build ./...`
- Code follows Go conventions

---

**Questions?** Open an issue or check the example in `.example/example.md`.
