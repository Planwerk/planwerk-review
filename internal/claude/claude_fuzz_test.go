package claude

import (
	"strings"
	"testing"
)

func FuzzStripMarkdownFences(f *testing.F) {
	f.Add(`{"findings": []}`)
	f.Add("```json\n{\"findings\": []}\n```")
	f.Add("```\n{\"findings\": []}\n```")
	f.Add("  \n```json\n{\"findings\": []}\n```\n  ")
	f.Add("```")
	f.Add("``````")
	f.Add("```json")
	f.Add("```no-newline-after-fence")
	f.Add("")

	f.Fuzz(func(t *testing.T, in string) {
		out := stripMarkdownFences(in)
		// Output must be a TrimSpace fixed point.
		if out != strings.TrimSpace(out) {
			t.Fatalf("output not trimmed: %q", out)
		}
		// Output must never be longer than the input.
		if len(out) > len(in) {
			t.Fatalf("output longer than input: %q -> %q", in, out)
		}
	})
}
