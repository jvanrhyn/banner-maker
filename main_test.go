package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
	return dir
}

func TestRun_Version(t *testing.T) {
	code, out, _ := runCLI(t, "--version")
	if code != 0 || !strings.Contains(out, "banner-maker") {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestRun_WritesFileAndReportsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.go")
	code, out, errs := runCLI(t, "--word", "HI", "--output", path)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.Contains(out, "Saved: "+path) {
		t.Errorf("stdout missing Saved line: %q", out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
}

func TestRun_RefusesToOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.go")
	os.WriteFile(path, []byte("keep me"), 0o644) //nolint:errcheck
	code, _, errs := runCLI(t, "--word", "HI", "--output", path)
	if code != 1 || !strings.Contains(errs, "--force") {
		t.Errorf("code=%d stderr=%q, want exit 1 mentioning --force", code, errs)
	}
	if data, _ := os.ReadFile(path); string(data) != "keep me" {
		t.Errorf("file modified: %q", data)
	}
}

func TestRun_ForceOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.go")
	os.WriteFile(path, []byte("old"), 0o644) //nolint:errcheck
	if code, _, errs := runCLI(t, "--word", "HI", "--output", path, "--force"); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "package main") {
		t.Errorf("file not overwritten: %q", data)
	}
}

func TestRun_PackageFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.go")
	if code, _, errs := runCLI(t, "--word", "HI", "--output", path, "--package", "ui"); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if data, _ := os.ReadFile(path); !strings.Contains(string(data), "package ui\n") {
		t.Errorf("package not applied: %q", data)
	}
}

func TestRun_InvalidPackage(t *testing.T) {
	dir := chdirTemp(t)
	code, _, errs := runCLI(t, "--word", "HI", "--package", "bad-name")
	if code != 1 || !strings.Contains(errs, "--package") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("no file should be written, found %d", len(entries))
	}
}

func TestRun_UnsupportedCharacters(t *testing.T) {
	chdirTemp(t)
	code, _, errs := runCLI(t, "--word", "héllo")
	if code != 1 || !strings.Contains(errs, "é") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
}

func TestRun_EmbedColor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.go")
	code, _, errs := runCLI(t, "--word", "HI", "--output", path, "--embed-color", "--color", "#FF0000")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "func HiBannerColor()") || !strings.Contains(string(data), `\x1b[38;2;255;0;0m`) {
		t.Errorf("colour not embedded:\n%s", data)
	}
}

func TestRun_InvalidTaglineColor(t *testing.T) {
	chdirTemp(t)
	code, _, errs := runCLI(t, "--word", "HI", "--tagline", "x", "--tagline-color", "999")
	if code != 1 || !strings.Contains(errs, "--tagline-color") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
}

func TestRun_StdoutPrintsSourceAndWritesNothing(t *testing.T) {
	dir := chdirTemp(t)
	code, out, errs := runCLI(t, "--word", "HI", "--stdout", "--package", "ui")
	if code != 0 {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if !strings.HasPrefix(out, "package ui\n") {
		t.Errorf("stdout should contain only Go source, got: %q", out)
	}
	if strings.Contains(out, "Saved:") {
		t.Errorf("stdout must not contain status text: %q", out)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("--stdout must not write files, found %d", len(entries))
	}
}

func TestRun_StdoutAndOutputConflict(t *testing.T) {
	chdirTemp(t)
	code, _, errs := runCLI(t, "--word", "HI", "--stdout", "--output", "x.go")
	if code != 1 || !strings.Contains(errs, "--stdout") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
}

func TestRun_InvalidColor(t *testing.T) {
	chdirTemp(t)
	if code, _, errs := runCLI(t, "--word", "HI", "--color", "zzz"); code != 1 || !strings.Contains(errs, "--color") {
		t.Errorf("code=%d stderr=%q", code, errs)
	}
}
