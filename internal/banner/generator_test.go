package banner_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvanrhyn/banner-maker/internal/banner"
)

func TestGenerate_MYCLI(t *testing.T) {
	result, err := banner.Generate("MYCLI")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}

	// Count non-blank content lines (ANSI Shadow renders 6 lines per character row)
	var contentLines []string
	for _, l := range strings.Split(result, "\n") {
		if strings.TrimSpace(l) != "" {
			contentLines = append(contentLines, l)
		}
	}
	if len(contentLines) != 6 {
		t.Errorf("expected 6 content lines, got %d", len(contentLines))
	}

	// Verify box-drawing characters present (ANSI Shadow font signature)
	if !strings.Contains(result, "█") {
		t.Error("expected block character █ in output")
	}
	if !strings.Contains(result, "╗") {
		t.Error("expected box-drawing character ╗ in output")
	}
	if !strings.Contains(result, "╚") {
		t.Error("expected box-drawing character ╚ in output")
	}
}

func TestGenerate_EmptyWord(t *testing.T) {
	_, err := banner.Generate("")
	if err == nil {
		t.Fatal("expected error for empty word, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected error message to mention 'empty', got: %v", err)
	}
}

func TestGenerate_Uppercase(t *testing.T) {
	lower, err := banner.Generate("hello")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}
	upper, err := banner.Generate("HELLO")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}
	if lower != upper {
		t.Error("lowercase and uppercase inputs should produce the same output")
	}
}

func TestGenerate_SingleChar(t *testing.T) {
	result, err := banner.Generate("A")
	if err != nil {
		t.Fatalf("Generate returned unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty output for single character input")
	}
}

func TestSave_WritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "banner.txt")

	content := "test banner content"
	if err := banner.Save(content, path); err != nil {
		t.Fatalf("Save returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read saved file: %v", err)
	}
	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestSave_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "banner.txt")

	if err := banner.Save("original", path); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	if err := banner.Save("overwritten", path); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read saved file: %v", err)
	}
	if string(data) != "overwritten" {
		t.Errorf("expected overwritten content, got %q", string(data))
	}
}

func TestColorize_DefaultColors(t *testing.T) {
	raw, _ := banner.Generate("A")
	colorized := banner.Colorize(raw, banner.ColorOptions{})

	// Output must still contain the original characters regardless of terminal color support
	if !strings.Contains(colorized, "█") {
		t.Error("colorized output missing block character █")
	}
	if !strings.Contains(colorized, "╗") {
		t.Error("colorized output missing shadow character ╗")
	}
}

func TestColorize_CustomTextColor(t *testing.T) {
	raw, _ := banner.Generate("A")
	opts := banner.ColorOptions{TextColor: "#FF0000", ShadowColor: "#0000FF"}
	colorized := banner.Colorize(raw, opts)

	// Content characters must be preserved regardless of color support
	if !strings.Contains(colorized, "█") {
		t.Error("colorized output missing block character █")
	}
}

func TestColorize_EmptyOptsUsesDefaults(t *testing.T) {
	raw, _ := banner.Generate("A")
	withDefaults := banner.Colorize(raw, banner.DefaultColors())
	withEmpty := banner.Colorize(raw, banner.ColorOptions{})

	// Both should produce identical output since empty fields fall back to defaults
	if withDefaults != withEmpty {
		t.Error("empty ColorOptions should produce the same output as explicit DefaultColors()")
	}
}

func TestColorize_PreservesNewlines(t *testing.T) {
	raw, _ := banner.Generate("A")
	colorized := banner.Colorize(raw, banner.ColorOptions{})

	rawLines := strings.Count(raw, "\n")
	colorizedLines := strings.Count(colorized, "\n")
	if rawLines != colorizedLines {
		t.Errorf("colorization changed newline count: got %d, want %d", colorizedLines, rawLines)
	}
}

func TestValidateColor_EmptyString(t *testing.T) {
	if err := banner.ValidateColor(""); err != nil {
		t.Errorf("expected nil for empty string, got %v", err)
	}
}

func TestValidateColor_ValidHex6(t *testing.T) {
	cases := []string{"#FFFFFF", "#000000", "#7D56F4", "#abcdef", "#ABCDef"}
	for _, c := range cases {
		if err := banner.ValidateColor(c); err != nil {
			t.Errorf("ValidateColor(%q) = %v, want nil", c, err)
		}
	}
}

func TestValidateColor_ValidHex3(t *testing.T) {
	if err := banner.ValidateColor("#FFF"); err != nil {
		t.Errorf("ValidateColor(\"#FFF\") = %v, want nil", err)
	}
}

func TestValidateColor_ValidANSI256(t *testing.T) {
	cases := []string{"0", "7", "128", "255"}
	for _, c := range cases {
		if err := banner.ValidateColor(c); err != nil {
			t.Errorf("ValidateColor(%q) = %v, want nil", c, err)
		}
	}
}

func TestValidateColor_InvalidANSI_OutOfRange(t *testing.T) {
	cases := []string{"256", "999", "-1", "1000"}
	for _, c := range cases {
		if err := banner.ValidateColor(c); err == nil {
			t.Errorf("ValidateColor(%q): expected error, got nil", c)
		}
	}
}

func TestValidateColor_InvalidHex(t *testing.T) {
	cases := []string{"#GGGGGG", "#1234", "#", "red", "blue", "hello"}
	for _, c := range cases {
		if err := banner.ValidateColor(c); err == nil {
			t.Errorf("ValidateColor(%q): expected error, got nil", c)
		}
	}
}

func TestColorize_InvalidColorFallsBackToDefaults(t *testing.T) {
	raw, _ := banner.Generate("A")
	// An out-of-range ANSI value must not panic; output must still contain characters
	colorized := banner.Colorize(raw, banner.ColorOptions{TextColor: "999", ShadowColor: "999"})
	if !strings.Contains(colorized, "█") {
		t.Error("colorized output missing block character █ after invalid color fallback")
	}
}

