package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jvanrhyn/banner-maker/internal/banner"
	"github.com/jvanrhyn/banner-maker/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=<semver>".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("banner-maker", flag.ContinueOnError)
	fs.SetOutput(stderr)

	word := fs.String("word", "", "Text to render (enables non-interactive mode)")
	color := fs.String("color", "", "Block/text color: hex (#RGB or #RRGGBB) or ANSI 0–255")
	shadow := fs.String("shadow", "", "Shadow color: hex (#RGB or #RRGGBB) or ANSI 0–255")
	tagline := fs.String("tagline", "", "Optional subtitle text")
	taglineColor := fs.String("tagline-color", "", "Tagline color: hex or ANSI 0–255 (default: terminal default)")
	align := fs.String("align", "left", "Tagline alignment: left or right")
	output := fs.String("output", "", "Output path for the generated .go file (default: {word}banner.go)")
	pkg := fs.String("package", "", "Package name for the generated file (default: main)")
	embedColor := fs.Bool("embed-color", false, "Also generate a {Word}BannerColor() func with ANSI colors embedded")
	force := fs.Bool("force", false, "Overwrite the output file if it already exists")
	toStdout := fs.Bool("stdout", false, "Print the generated Go source to stdout instead of writing a file")
	showVersion := fs.Bool("version", false, "Print version and exit")
	fs.BoolVar(showVersion, "v", false, "Print version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "banner-maker", version)
		return 0
	}

	if os.Getenv("DEBUG") != "" {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Fprintln(stderr, "could not open debug log:", err)
			return 1
		}
		defer f.Close()
	}

	if *word == "" {
		// Interactive TUI mode
		p := tea.NewProgram(tui.InitialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		return 0
	}

	// Non-interactive mode: validate everything before any I/O.
	fail := func(prefix string, err error) int {
		fmt.Fprintln(stderr, prefix, err)
		return 1
	}
	for _, c := range []struct{ flag, value string }{
		{"--color", *color}, {"--shadow", *shadow}, {"--tagline-color", *taglineColor},
	} {
		if err := banner.ValidateColor(c.value); err != nil {
			return fail("Invalid "+c.flag+":", err)
		}
	}
	if err := banner.ValidateTaglineAlign(*align); err != nil {
		return fail("Invalid --align:", err)
	}
	if err := banner.ValidatePackage(*pkg); err != nil {
		return fail("Invalid --package:", err)
	}
	if *toStdout && *output != "" {
		return fail("Invalid flags:", errors.New("--stdout cannot be combined with --output"))
	}

	raw, err := banner.Generate(*word)
	if err != nil {
		return fail("Error generating banner:", err)
	}

	colors := banner.ColorOptions{TextColor: *color, ShadowColor: *shadow, TaglineColor: *taglineColor}
	tag := banner.Tagline{Text: *tagline, Align: *align}
	srcOpts := banner.SourceOptions{Package: *pkg, EmbedColor: *embedColor, Colors: colors, Overwrite: *force}

	if *toStdout {
		src, err := banner.GenerateGoSourceWithOptions(*word, raw, tag, srcOpts)
		if err != nil {
			return fail("Error generating source:", err)
		}
		fmt.Fprint(stdout, src)
		return 0
	}

	outPath := *output
	if outPath == "" {
		outPath = banner.DefaultOutputPath(*word)
	}

	if err := banner.SaveGoFileWithOptions(*word, raw, tag, outPath, srcOpts); err != nil {
		if errors.Is(err, banner.ErrFileExists) {
			return fail("Error saving file:", fmt.Errorf("%w (use --force to overwrite)", err))
		}
		return fail("Error saving file:", err)
	}

	fmt.Fprintln(stdout, banner.ColorizeWithTagline(raw, tag, colors))
	fmt.Fprintln(stdout, "Saved:", outPath)
	return 0
}
