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
