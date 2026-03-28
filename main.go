package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jvanrhyn/banner-maker/internal/banner"
	"github.com/jvanrhyn/banner-maker/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=<semver>".
var version = "dev"

func main() {
	word := flag.String("word", "", "Text to render (enables non-interactive mode)")
	color := flag.String("color", "", "Block/text color: hex (#RGB or #RRGGBB) or ANSI 0–255")
	shadow := flag.String("shadow", "", "Shadow color: hex (#RGB or #RRGGBB) or ANSI 0–255")
	tagline := flag.String("tagline", "", "Optional subtitle text")
	align := flag.String("align", "left", "Tagline alignment: left or right")
	output := flag.String("output", "", "Output path for the generated .go file (default: {word}banner.go)")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("banner-maker", version)
		return
	}

	if os.Getenv("DEBUG") != "" {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not open debug log:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	if *word == "" {
		// Interactive TUI mode
		p := tea.NewProgram(
			tui.InitialModel(),
			tea.WithAltScreen(),
		)
		if _, err := p.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}

	// Non-interactive mode
	if err := banner.ValidateColor(*color); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid --color:", err)
		os.Exit(1)
	}
	if err := banner.ValidateColor(*shadow); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid --shadow:", err)
		os.Exit(1)
	}
	if err := banner.ValidateTaglineAlign(*align); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid --align:", err)
		os.Exit(1)
	}

	raw, err := banner.Generate(*word)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating banner:", err)
		os.Exit(1)
	}

	colorOpts := banner.ColorOptions{TextColor: *color, ShadowColor: *shadow}
	tag := banner.Tagline{Text: *tagline, Align: *align}

	outPath := *output
	if outPath == "" {
		varName, _ := banner.GoIdent(*word)
		outPath = strings.TrimSuffix(varName, "Logo") + "banner.go"
	}

	if err := banner.SaveGoFile(*word, raw, tag, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "Error saving file:", err)
		os.Exit(1)
	}

	colored := banner.Colorize(banner.AppendTagline(raw, tag), colorOpts)
	fmt.Println(colored)
	fmt.Println("Saved:", outPath)
}
