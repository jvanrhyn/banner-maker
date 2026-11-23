package banner_test

import (
	"testing"

	"github.com/jvanrhyn/banner-maker/internal/banner"
)

// Every character ValidateWord accepts must actually render.
func TestGenerate_AllPrintableASCIIRenders(t *testing.T) {
	for r := ' '; r <= '~'; r++ {
		if r == ' ' {
			continue // whitespace-only input is ErrEmptyWord by design
		}
		out, err := banner.Generate("A" + string(r))
		if err != nil || out == "" {
			t.Errorf("Generate(%q) failed: %v", "A"+string(r), err)
		}
	}
}
