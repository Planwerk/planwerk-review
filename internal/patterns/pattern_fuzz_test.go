package patterns

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add(`# Review Pattern: Seed

**Review-Area**: testing
**Detection-Hint**: seed
**Severity**: INFO
**Category**: design-principle
**Applies-When**: go, python
**Sources**: Title One (https://example.com), Title Two

## What to check

Something.
`)
	f.Add("# Review Pattern: Minimal\n")
	f.Add("")
	f.Add("not a pattern\n")
	f.Add("# Review Pattern:\n**Sources**: a (not-a-url), b, ()\n")

	// Seed with real pattern files if reachable from this package's directory.
	for _, dir := range []string{
		filepath.Join("..", "..", "patterns", "design"),
		filepath.Join("..", "..", "patterns", "technology", "go"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			f.Add(string(data))
		}
	}

	f.Fuzz(func(t *testing.T, content string) {
		p, err := Parse(content)
		if err != nil {
			return
		}
		// On success the pattern must have a name, and downstream helpers
		// must not panic on the parsed value.
		if p.Name == "" {
			t.Fatalf("Parse returned no error but Name is empty")
		}
		_ = p.FormatForPrompt()
		_ = p.AppliesTo(nil)
		_ = p.AppliesTo([]string{"go", "python"})
	})
}
