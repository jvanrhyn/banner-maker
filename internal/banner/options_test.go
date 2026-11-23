package banner_test

import (
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvanrhyn/banner-maker/internal/banner"
)

// ─── ValidatePackage ────────────────────────────────────────────────────────

func TestValidatePackage(t *testing.T) {
	for _, ok := range []string{"", "main", "ui", "foo_bar", "v2"} {
		if err := banner.ValidatePackage(ok); err != nil {
			t.Errorf("ValidatePackage(%q) = %v, want nil", ok, err)
		}
	}
	for _, bad := range []string{"1x", "func", "a-b", "_", "a b", "pkg.name"} {
		if err := banner.ValidatePackage(bad); err == nil {
			t.Errorf("ValidatePackage(%q) = nil, want error", bad)
		}
	}
}

// ─── ValidateWord / unsupported characters ──────────────────────────────────

func TestValidateWord_Supported(t *testing.T) {
	for _, w := range []string{"HELLO", "Hello World-1!", "a_b"} {
		if err := banner.ValidateWord(w); err != nil {
			t.Errorf("ValidateWord(%q) = %v, want nil", w, err)
		}
	}
}

func TestValidateWord_Unsupported(t *testing.T) {
	for _, w := range []string{"héllo", "a\nb", "😀", "tab\there"} {
		err := banner.ValidateWord(w)
		if !errors.Is(err, banner.ErrUnsupportedChars) {
			t.Errorf("ValidateWord(%q) = %v, want ErrUnsupportedChars", w, err)
		}
	}
}

func TestValidateWord_ErrorNamesOffendingChars(t *testing.T) {
	err := banner.ValidateWord("caf\u00e9")
	if err == nil || !strings.Contains(err.Error(), "é") {
		t.Errorf("expected error naming 'é', got %v", err)
	}
}

func TestGenerate_UnsupportedCharsRejected(t *testing.T) {
	if _, err := banner.Generate("héllo"); !errors.Is(err, banner.ErrUnsupportedChars) {
		t.Errorf("Generate with accented char = %v, want ErrUnsupportedChars", err)
	}
}

// ─── Package option ─────────────────────────────────────────────────────────

func TestGenerateGoSourceWithOptions_Package(t *testing.T) {
	raw, _ := banner.Generate("A")
	src, err := banner.GenerateGoSourceWithOptions("A", raw, banner.Tagline{}, banner.SourceOptions{Package: "ui"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "package ui\n") {
		t.Errorf("expected 'package ui', got:\n%s", src)
	}
}

func TestGenerateGoSourceWithOptions_DefaultPackageIsMain(t *testing.T) {
	raw, _ := banner.Generate("A")
	src, err := banner.GenerateGoSourceWithOptions("A", raw, banner.Tagline{}, banner.SourceOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "package main\n") {
		t.Errorf("expected 'package main', got:\n%s", src)
	}
}

func TestGenerateGoSourceWithOptions_InvalidPackage(t *testing.T) {
	raw, _ := banner.Generate("A")
	_, err := banner.GenerateGoSourceWithOptions("A", raw, banner.Tagline{}, banner.SourceOptions{Package: "not-valid"})
	if err == nil {
		t.Error("expected error for invalid package name")
	}
}

// ─── Embedded colour ────────────────────────────────────────────────────────

func TestGenerateGoSourceWithOptions_NoEmbedHasNoColorFunc(t *testing.T) {
	raw, _ := banner.Generate("A")
	src, _ := banner.GenerateGoSourceWithOptions("A", raw, banner.Tagline{}, banner.SourceOptions{})
	if strings.Contains(src, "BannerColor") || strings.Contains(src, `\x1b`) {
		t.Errorf("colour output must be opt-in, got:\n%s", src)
	}
}

func TestGenerateGoSourceWithOptions_EmbedColor(t *testing.T) {
	raw, _ := banner.Generate("A")
	opts := banner.SourceOptions{
		EmbedColor: true,
		Colors:     banner.ColorOptions{TextColor: "#FF0000", ShadowColor: "205", TaglineColor: "#0f0"},
	}
	src, err := banner.GenerateGoSourceWithOptions("A", raw, banner.Tagline{Text: "hi"}, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"func ABanner() string",
		"func ABannerColor() string",
		`\x1b[38;2;255;0;0m`, // text: #FF0000 truecolor
		`\x1b[38;5;205m`,     // shadow: ANSI-256 index
		`\x1b[38;2;0;255;0m`, // tagline: #0f0 expanded
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated source missing %q:\n%s", want, src)
		}
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "x.go", src, parser.AllErrors); err != nil {
		t.Errorf("generated source does not parse: %v", err)
	}
}

func TestColorizeANSI_AlwaysEmitsEscapes(t *testing.T) {
	out := banner.ColorizeANSI("█╗ \n", banner.Tagline{}, banner.ColorOptions{TextColor: "#102030", ShadowColor: "5"})
	if !strings.Contains(out, "\x1b[38;2;16;32;48m█\x1b[0m") {
		t.Errorf("text run not coloured: %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;5m╗\x1b[0m") {
		t.Errorf("shadow run not coloured: %q", out)
	}
}

func TestColorizeANSI_TaglineColor(t *testing.T) {
	raw := "█\n"
	plain := banner.ColorizeANSI(raw, banner.Tagline{Text: "sub"}, banner.ColorOptions{})
	if strings.Contains(plain, "sub\x1b") || !strings.Contains(plain, "\nsub\n") {
		t.Errorf("tagline must be unstyled without TaglineColor: %q", plain)
	}
	coloured := banner.ColorizeANSI(raw, banner.Tagline{Text: "sub"}, banner.ColorOptions{TaglineColor: "9"})
	if !strings.Contains(coloured, "\x1b[38;5;9msub\x1b[0m") {
		t.Errorf("tagline not coloured: %q", coloured)
	}
}

func TestColorizeANSI_RightAlignedTaglinePaddingNotStyled(t *testing.T) {
	out := banner.ColorizeANSI("████\n", banner.Tagline{Text: "ab", Align: "right"}, banner.ColorOptions{TaglineColor: "9"})
	if !strings.Contains(out, "\n  \x1b[38;5;9mab\x1b[0m\n") {
		t.Errorf("padding should precede the styled text: %q", out)
	}
}

func TestColorizeWithTagline_NoTaglineMatchesColorize(t *testing.T) {
	raw, _ := banner.Generate("A")
	if banner.ColorizeWithTagline(raw, banner.Tagline{}, banner.ColorOptions{}) != banner.Colorize(raw, banner.ColorOptions{}) {
		t.Error("ColorizeWithTagline without tagline should equal Colorize")
	}
}

func TestColorizeWithTagline_KeepsTaglineText(t *testing.T) {
	raw, _ := banner.Generate("A")
	out := banner.ColorizeWithTagline(raw, banner.Tagline{Text: "my sub"}, banner.ColorOptions{TaglineColor: "#FF00FF"})
	if !strings.Contains(out, "my sub") {
		t.Errorf("tagline text missing: %q", out)
	}
}

func TestValidateColor_UsedForTaglineColor(t *testing.T) {
	if err := banner.ValidateColor("nope"); err == nil {
		t.Error("expected invalid colour error")
	}
}

// ─── Overwrite protection ───────────────────────────────────────────────────

func TestSaveGoFileWithOptions_RefusesToOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.go")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := banner.Generate("A")
	err := banner.SaveGoFileWithOptions("A", raw, banner.Tagline{}, path, banner.SourceOptions{})
	if !errors.Is(err, banner.ErrFileExists) {
		t.Fatalf("err = %v, want ErrFileExists", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "original" {
		t.Errorf("existing file was modified: %q", data)
	}
}

func TestSaveGoFileWithOptions_OverwriteReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.go")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _ := banner.Generate("A")
	if err := banner.SaveGoFileWithOptions("A", raw, banner.Tagline{}, path, banner.SourceOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "package main") {
		t.Errorf("file not replaced: %q", data)
	}
}

func TestSaveGoFileWithOptions_NewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.go")
	raw, _ := banner.Generate("A")
	if err := banner.SaveGoFileWithOptions("A", raw, banner.Tagline{}, path, banner.SourceOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestSaveGoFileWithOptions_InvalidPackageWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.go")
	raw, _ := banner.Generate("A")
	if err := banner.SaveGoFileWithOptions("A", raw, banner.Tagline{}, path, banner.SourceOptions{Package: "1bad"}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("file must not be created when options are invalid")
	}
}

// ─── DefaultOutputPath ──────────────────────────────────────────────────────

func TestDefaultOutputPath(t *testing.T) {
	cases := map[string]string{
		"MYCLI":  "mycli" + "banner.go",
		"my-cli": "mycli" + "banner.go",
		"2fast":  "b2fast" + "banner.go",
	}
	for word, want := range cases {
		if got := banner.DefaultOutputPath(word); got != want {
			t.Errorf("DefaultOutputPath(%q) = %q, want %q", word, got, want)
		}
	}
}
