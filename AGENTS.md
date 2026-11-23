# AGENTS.md — Working in banner-maker

This document captures essential context for agents working in this codebase. Focus on non-obvious patterns, gotchas, and conventions that aren't self-evident from reading the code.

## Project Overview

**banner-maker** is a Go CLI tool that generates ASCII art banners using the ANSI Shadow figlet font. It provides both interactive (Bubble Tea TUI) and non-interactive (flag-based) modes, then saves the output as reusable Go source files.

**Key characteristics:**
- Go 1.24.2+
- Charmbracelet stack (Bubble Tea, Bubbles, Lip Gloss) for TUI
- Test-first repo; `make coverage` for current numbers (banner ≈95%, tui ≈94%, root ≈74% — the interactive-launch branch of `run` can't be unit tested)
- Git-based SemVer 2.0 versioning via Makefile ldflags injection
- Branch-based development workflow with strong guardrails

## Essential Commands

### Build & Run
```bash
make build          # Build binary to ./banner-maker
make install        # Install to $GOPATH/bin (with version injected)
make uninstall      # Remove from $GOPATH/bin
make version        # Print current version string
```

### Testing & Quality
```bash
make test           # Run all tests (go test ./...)
make coverage       # Generate HTML coverage report (coverage.html)
make lint           # Run go vet + staticcheck (if installed)
make clean          # Remove build artifacts
```

### Interactive Mode
```bash
./banner-maker      # Launch TUI (no arguments)
DEBUG=1 ./banner-maker  # TUI with debug logging to debug.log
```

### Non-Interactive Mode (Flag-based)
```bash
./banner-maker --word "TEXT" --color "#FF00FF" --shadow "200" \
  --tagline "subtitle" --tagline-color 51 --align right --output mybanner.go
```

| Flag | Meaning |
|---|---|
| `--word` | Text to render; its presence selects non-interactive mode |
| `--color`, `--shadow`, `--tagline-color` | Hex (`#RGB`/`#RRGGBB`) or ANSI 0–255. Tagline color defaults to unstyled |
| `--tagline`, `--align` | Subtitle and `left`/`right` alignment |
| `--output` | Output path (default `{slug}banner.go`, see `DefaultOutputPath`) |
| `--package` | Package clause of the generated file (default `main`) |
| `--embed-color` | Also emit `{Word}BannerColor()` with ANSI escapes baked in |
| `--force` | Overwrite an existing output file (otherwise exit 1, file untouched) |
| `--stdout` | Print the generated Go source only; writes no file; conflicts with `--output` |

**Version flag:**
```bash
./banner-maker --version
./banner-maker -v
```

## Code Organization

```
banner-maker/
├── main.go                          # run(args, stdout, stderr) int — flag parsing, mode routing
├── main_test.go                     # CLI tests via run() (non-interactive paths only)
├── internal/banner/                 # Core banner generation logic
│   ├── generator.go                 # Generate, Colorize*, source generation, saving, validation
│   ├── generator_test.go            # Original unit tests
│   ├── options_test.go              # Package/overwrite/embed-color/tagline-color/validation tests
│   ├── ascii_all_test.go            # Every printable ASCII char must render
│   └── fonts/ansi-shadow.flf        # Embedded figlet font (via //go:embed)
├── internal/tui/                    # Bubble Tea interactive interface
│   ├── model.go                     # TUI state machine (3 screens)
│   └── model_test.go                # TUI interaction tests
├── Makefile                         # Build targets, version injection
└── [config]
    ├── go.mod, go.sum               # Dependencies
    ├── .github/copilot-instructions.md  # Copilot workflow rules
    └── README.md                    # User documentation
```

### Key Packages

**`internal/banner`** — Core generation and I/O
- `Generate(word)` — Figlet rendering with embedded ANSI Shadow font; rejects empty input (`ErrEmptyWord`) and non-printable-ASCII (`ErrUnsupportedChars`)
- `Colorize(raw, opts)` — Terminal styling via lipgloss (honours color profile / `NO_COLOR` / non-TTY)
- `ColorizeWithTagline(raw, tag, opts)` — Same, plus appends the tagline and styles it with `TaglineColor`
- `ColorizeANSI(raw, tag, opts)` — Always emits literal escapes (24-bit for hex, `38;5;n` for indices); used for embedding in generated source
- `AppendTagline(raw, tag)` — Add optional subtitle below banner
- `GenerateGoSource` / `GenerateGoSourceWithOptions` — Build gofmt'd source; `SaveGoFile` (overwrites) / `SaveGoFileWithOptions` (refuses to overwrite unless `SourceOptions.Overwrite`; returns `ErrFileExists`)
- `SourceOptions{Package, EmbedColor, Colors, Overwrite}` — Zero value = `package main`, plain banner, no overwrite
- `ValidateColor`, `ValidateTaglineAlign`, `ValidateWord`, `ValidatePackage` — Input validation (empty is valid for color/align/package)
- `GoIdent(word)` — Returns `(varName, funcName)`; **no error return** (see gotcha below)
- `DefaultOutputPath(word)` — Shared default filename for CLI and TUI
- `Save(content, path)` — Plain file write helper (not used by the generator)

**`internal/tui`** — Interactive banner builder (Bubble Tea)
- **Model** — Stateful Bubble Tea model with 3 screens:
  - `screenInput` (0) — Form: word, text/shadow colour, tagline, tagline colour, alignment, package, embed-colour toggle, output path
  - `screenPreview` (1) — Preview with colored banner (and a note when the colour func will be generated)
  - `screenDone` (2) — Confirmation screen showing save path
- **Tab navigation** — Cycles through 9 focus positions (fieldWord → … → fieldOutput)
- **Field indexing** — `fieldAlign` and `fieldEmbed` are toggles, not textinputs; map via `inputIdx()` helper
- The form is one line per field (label + control) with a `▸` focus marker, so it fits short terminals

## Important Gotchas & Non-Obvious Patterns

### Version Injection via Makefile

The Makefile uses **Git-based SemVer 2.0 versioning** (not a VERSION file):
- Derives version from highest git tag reachable from HEAD
- Fallback if no tag: `0.0.0+{commit_count}.{short_sha}`
- Injects via `LDFLAGS = -ldflags "-X main.version=$(VERSION)"`
- **Build without make** (e.g., `go build .`) sets version to `"dev"`
- **Always use make** for production builds to get real version strings

**Critical**: After tagging (e.g., `git tag v1.0.0`), clean and rebuild to see the new version:
```bash
make clean
make build
./banner-maker --version  # Must show e.g., "1.0.0+abc123" not "dev"
```

### Embedded Font File

The ANSI Shadow figlet font is embedded at compile time via `//go:embed`:
```go
//go:embed fonts/ansi-shadow.flf
var ansiFontBytes []byte
```

- **Don't rename** `fonts/ansi-shadow.flf` without updating the go:embed comment
- **Font must exist** at `internal/banner/fonts/ansi-shadow.flf` or build will fail
- Loaded at runtime via `figlet4go.NewAsciiRender().LoadBindataFont()`

### Color Validation & ANSI 256

Color inputs accept two formats; both must pass validation before use:
1. **Hex**: `#RGB` or `#RRGGBB` (lowercase or uppercase)
2. **ANSI 256**: decimal 0–255

**Gotcha**: Empty string is **valid** and means "use default" — don't confuse with error.

Default colors are defined in `banner.DefaultColors()`:
```go
TextColor:   "#FFFFFF"    // white
ShadowColor: "#555555"    // dark gray
```

Invalid colors in colorization **silently fall back** to defaults (no panic) via `resolveColors`; an invalid or empty `TaglineColor` means *unstyled* (it has no default). Callers (CLI/TUI) validate first so users still get errors.

**Two rendering paths — don't mix them up:** `Colorize`/`ColorizeWithTagline` go through lipgloss and produce *no escapes* in tests/non-TTY. `ColorizeANSI` always emits escapes, so tests asserting on escape sequences must use it (or `GenerateGoSourceWithOptions` with `EmbedColor`). Both paint *runs* of same-class runes with one escape each, not per rune.

### Box-Drawing Characters Are the Shadow

The ANSI Shadow font uses specific box-drawing runes for the drop-shadow effect:
```go
var shadowRunes = map[rune]bool{
  '╗': true, '╔': true, '╝': true, '╚': true, '═': true, '║': true,
}
```

- Block characters (`█`) receive `TextColor`
- Box-drawing chars receive `ShadowColor`
- All other runes (spaces, newlines) pass through unchanged

**Don't assume all non-block chars are shadow** — spaces and newlines must be preserved exactly.

### Go Identifier Sanitization (GoIdent)

`banner.GoIdent(word)` returns `(varName, funcName)` and never fails:
- Lowercases the word and **drops** every character that isn't a letter or digit (hyphens are removed, not replaced)
- Empty result → slug `"banner"`; leading digit → prefixed with `b`
- `varName = slug + "Logo"`, `funcName = Title(slug) + "Banner"`

**Examples**: `GoIdent("my-cli")` → `("mycliLogo", "MycliBanner")`; `GoIdent("2fast")` → `("b2fastLogo", "B2fastBanner")`.

With `EmbedColor` the generated file also gets `varName+"Color"` and `funcName+"Color"` (e.g. `mycliLogoColor`, `MycliBannerColor`).

### Unsupported Characters

Only printable ASCII (space–`~`) is accepted; `ValidateWord` lists offending runes in the error. Input is upper-cased before rendering. Whitespace-only input is `ErrEmptyWord`.

### TUI Screen State Machine

The TUI has strict screen progression:
1. **screenInput** → User fills form, hits Enter to validate & generate preview
2. **screenPreview** → Shows colored banner, asks "Save? (y/n)"
3. **screenDone** → Displays save path, returns to screenInput for next banner

**Key navigation**:
- **Tab** / **Shift+Tab** → Cycle focus (wraps through all 9 tab stops)
- **Enter** at screenInput → Validate, generate, move to screenPreview
- **y** at screenPreview → Write file, move to screenDone
- **n** at screenPreview → Return to screenInput (discard preview)
- **Ctrl+C** → Exit from any screen

`fieldAlign` and `fieldEmbed` are toggles rendered separately (←/→ pick a side, space flips); they have no `textinput`, so `inputIdx()` returns `-1` for them.

### TUI Field Indexing

Don't confuse **focusIdx** (tab stop, 0–8) with the **inputs array slot** (0–6):
```go
// tab stops (focus order)           // inputs slots
fieldWord        = 0                 slotWord        = 0
fieldTextColor   = 1                 slotTextColor   = 1
fieldShadowColor = 2                 slotShadowColor = 2
fieldTagline     = 3                 slotTagline     = 3
fieldTaglineColor= 4                 slotTaglineColor= 4
fieldAlign       = 5  // toggle      (no slot)
fieldPackage     = 6                 slotPackage     = 5
fieldEmbed       = 7  // toggle      (no slot)
fieldOutput      = 8                 slotOutput      = 6
```

`inputIdx(focusIdx)` converts via the `tabInput` table and returns `-1` for the two toggles (`fieldAlign`, `fieldEmbed`). Tests that tab to a field hardcode these counts — update them if you add a field.

**Save semantics:** the TUI never overwrites silently. `doSave` passes `SourceOptions.Overwrite=false`; on `banner.ErrFileExists` the model sets `confirmOverwrite` and the next `y` overwrites.

### Output Path Generation

If the user doesn't specify `--output` (or leaves the TUI field blank), `banner.DefaultOutputPath(word)` is used: the `GoIdent` slug (without `Logo`) + `banner.go`.

**Examples**: "MYCLI" → `myclibanner.go`; "my-app" → `myappbanner.go`; "2fast" → `b2fastbanner.go`.

**Gotcha:** `.gitignore` contains `/*banner.go`, so default-named output in the repo root is ignored by git. `coverage.html` is *not* ignored (only `*.out`) — delete it before committing.

**Overwrite behaviour:** the CLI and TUI never replace an existing file silently (`O_EXCL` create → `ErrFileExists`). CLI needs `--force`; TUI asks for a second `y`. Library callers of the legacy `SaveGoFile` still overwrite.

### Non-Interactive Mode Entry Point

`main()` is just `os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`. `run` uses its own `flag.NewFlagSet` and returns the exit code, which is what makes it testable — **write CLI tests against `run()`**, and use `chdirTemp` in tests so default-named output doesn't land in the repo.

- **Interactive** (default): `--word` empty → launch TUI
- **Non-interactive**: `--word "TEXT"` provided → validate, generate, save (or `--stdout`), exit

All flag validation (colors, align, package, flag conflicts) happens **before** any file I/O. Errors go to stderr with exit 1 (flag-parse errors exit 2). `--stdout` prints *only* the Go source, so keep status text out of that path.

## Code Patterns & Conventions

### Error Handling

- Explicit errors (not panics) for user input validation
- Sentinel errors exported from `banner`: `ErrEmptyWord`, `ErrUnsupportedChars`, `ErrFileExists` (test with `errors.Is`; the latter two are wrapped with detail)
- Validation functions return `fmt.Errorf()` with descriptive messages
- TUI errors display in `errorStyle` (red, bold) on screen; non-interactive errors go to stderr

### Function Naming

- **Generator functions** (banner package): `Generate`, `Colorize*`, `AppendTagline`, `GenerateGoSource*`, `SaveGoFile*`
- **Validators**: `ValidateColor`, `ValidateTaglineAlign`, `ValidateWord`, `ValidatePackage`
- **Helpers**: `GoIdent`, `DefaultOutputPath`, `DefaultColors`, `ColorOptions`/`Tagline`/`SourceOptions` (structs)
- New options go in `SourceOptions`/`ColorOptions` with a zero value that preserves old behaviour; keep thin wrappers for the old signatures

### Lipgloss Styling

All TUI styles are defined as module-level variables at the top of `tui/model.go`:
```go
var (
  titleStyle       // bold purple background
  bannerBoxStyle   // rounded border
  errorStyle       // red, bold
  helpStyle        // dim gray
  labelStyle       // light gray
  successStyle     // bright green
  toggleOnStyle    // purple background (focus)
  toggleOffStyle   // dim (unfocus)
  focusStyle       // purple bold marker for the focused form row
  docStyle         // padding
)
```

**Pattern**: Don't inline Lipgloss styles. Keep them centralized and reuse.

### Testing Conventions

- Tests use the `_test` package (black-box imports): `package banner_test`, `import "github.com/jvanrhyn/banner-maker/internal/banner"`
- Table-driven tests are minimal; most are single assertion tests
- Tests verify both happy path and error cases (empty input, invalid colors, etc.)
- TUI tests verify screen transitions, validation and saved-file contents, not rendering. Helpers: `tab(m, n)`, `typeRunes`, `saveVia`, `shortTempDir`
- **The output textinput has a 120-char limit**: `t.TempDir()` paths derived from long test names can exceed it and get silently truncated, so TUI tests that type a path use `shortTempDir`
- Tests that tab to a field hardcode tab-stop counts — update them when adding fields
- Anything that writes files must use a temp dir or `chdirTemp`

### Generated Go Code Format

Generated banner files use `gofmt`-formatted output (built by `GenerateGoSourceWithOptions`):
```go
package main // or --package

import "strings"

var mycliLogo = []string{
	"line1",
	"line2",
}

// MycliBanner returns the banner art as a single string.
func MycliBanner() string {
	return strings.Join(mycliLogo, "\n")
}

// only with --embed-color:
var mycliLogoColor = []string{"\x1b[38;2;255;0;0m██\x1b[0m…"}

// MycliBannerColor returns the banner art with ANSI color escape sequences embedded.
func MycliBannerColor() string { return strings.Join(mycliLogoColor, "\n") }
```

**Key details**:
- Variable name: slug + "Logo"; function name: title-cased slug + "Banner" (see `GoIdent`)
- Slice type: `[]string`; imports only `"strings"`
- Embedded escapes are unconditional — callers must check `NO_COLOR`/TTY themselves
- The package name is not derived from the output directory; it's `main` unless `--package`/the TUI field says otherwise

## Branch & Commit Workflow

**Branch Safety (from .github/copilot-instructions.md):**
- Always work in feature branches (e.g., `feature/xyz`, `fix/abc`, `docs/readme`, `build/makefile`)
- Never commit directly to `main` (exception: explicit user approval)
- Merge to `main` requires explicit user confirmation ("merge to main")

**Push Policy:**
- Do NOT push by default
- Only run `git push` if explicitly instructed
- Confirm target remote/branch before pushing

**Commit Style:**
- Conventional commits encouraged (feat:, fix:, docs:, build:, etc.)
- TDD: write tests first, then implementation
- Each commit should be logically coherent

## Development Approach (TDD)

From copilot instructions:
1. **Write tests first** (or update existing tests)
2. **Implement minimal code** to pass tests
3. **Keep tests focused, deterministic, and fast**

This is a TDD-first repo. If adding a feature:
- Add tests to `*_test.go` files first
- Run `make test` to verify they fail
- Implement the feature
- Rerun tests until all pass
- Run `make coverage` to verify coverage
- Run `make lint` to check for vet/staticcheck issues

## Color & Terminal Support

- Previews assume 256-color ANSI terminal support; lipgloss downsamples for fewer colors and drops color for `NO_COLOR`/non-TTY
- Hex colors are converted by lipgloss for previews, but emitted as 24-bit escapes (`38;2;r;g;b`) in embedded output
- ANSI 256 indices (0–255) pass directly to the terminal (`38;5;n` when embedded)

## Useful Files to Know

- **main.go** — flag parsing, `run()`, mode routing, debug logging setup
- **internal/banner/generator.go** — all generation, colorization, validation, source generation
- **internal/tui/model.go** — full Bubble Tea state machine and views
- **Makefile** — version injection, build targets
- **go.mod** — Go 1.24.2; bubbletea, bubbles, lipgloss, figlet4go
- **README.md** — User-facing documentation (keep flag/field lists in sync)

## Common Tasks for Agents

### Adding a New CLI Flag
1. Write a failing test in `main_test.go` (call `run()`), plus banner tests if library behaviour changes
2. Declare the flag on the `FlagSet` in `run()` in `main.go`
3. Validate it in the validation block, before any I/O; add a `Validate*` in `internal/banner` if reusable
4. Thread it through `SourceOptions`/`ColorOptions`/`Tagline` rather than adding positional parameters
5. Consider TUI parity (new field + `validate()` + `sourceOpts()`), then update README.md (flags and TUI sections) and this file

### Modifying Banner Generation
1. Changes live in `internal/banner/generator.go`
2. Update tests in `generator_test.go` / `options_test.go` (black-box package)
3. Keep coverage high (`make coverage`)
4. Test interactively: `make build && ./banner-maker --word TEST --color "#FF00FF"`

### Adding TUI Features
1. Modify `internal/tui/model.go`: add a `field*` tab stop (and a `slot*` + `tabInput` entry if it's a text input), keep `tabStops` last
2. Add validation to `validate()`, state to `sourceOpts()`/`colorOpts()`, and a `m.row(...)` line in `viewInput`
3. Update tab-count assumptions in `model_test.go`
4. Test: `make build && ./banner-maker` (manual interactive testing)

### Releasing a Version
1. Commit all changes
2. Create git tag: `git tag v1.2.3` (SemVer format)
3. Run `make clean && make build` to inject version
4. Test: `./banner-maker --version` shows correct version
5. Push tag if approved: `git push origin v1.2.3`

## Debugging

**Enable debug logging:**
```bash
DEBUG=1 ./banner-maker
# Writes to debug.log (Bubble Tea internal logs)
```

**Check generated Go code:**
```bash
./banner-maker --word TEST --stdout          # source to stdout, no file written
./banner-maker --word TEST --output debug-banner.go --force
```

**Inspect figlet output (before coloring):**
```go
raw, _ := banner.Generate("HELLO")
fmt.Println(raw)  // Raw ANSI Shadow art
```

## LSP/Diagnostics

This is a standard Go module. Use:
- `gopls` for diagnostics (should auto-detect)
- Agents with LSP can run `lsp_diagnostics` on individual files
- Run `make lint` for static analysis

## See Also

- **README.md** — Comprehensive user guide, examples, feature list
- **go.mod** — Dependency versions
- **.github/copilot-instructions.md** — Workflow guardrails (branch safety, testing, communication)
