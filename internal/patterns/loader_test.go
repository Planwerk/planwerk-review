package patterns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePattern(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goPattern = `# Review Pattern: Go Errors

**Review-Area**: quality
**Detection-Hint**: bare returns
**Severity**: WARNING
**Category**: technology
**Applies-When**: go

## What to check

Wrap errors.
`

const pythonPattern = `# Review Pattern: Python Types

**Review-Area**: quality
**Detection-Hint**: missing type hints
**Severity**: INFO
**Category**: technology
**Applies-When**: python

## What to check

Add type hints.
`

const yagniPattern = `# Review Pattern: YAGNI

**Review-Area**: architecture
**Detection-Hint**: premature abstractions
**Severity**: INFO
**Category**: design-principle

## What to check

No premature abstractions.
`

const legacyPattern = `# Review Pattern: Legacy Check

**Review-Area**: quality
**Detection-Hint**: check stuff
**Severity**: WARNING

## What to check

Check stuff.
`

func TestLoad_Recursive(t *testing.T) {
	dir := t.TempDir()
	writePattern(t, filepath.Join(dir, "technology", "go"), "go-errors.md", goPattern)
	writePattern(t, filepath.Join(dir, "design"), "yagni.md", yagniPattern)

	pats, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pats) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(pats))
	}

	names := map[string]bool{}
	for _, p := range pats {
		names[p.Name] = true
	}
	if !names["Go Errors"] {
		t.Error("missing Go Errors pattern")
	}
	if !names["YAGNI"] {
		t.Error("missing YAGNI pattern")
	}
}

func TestLoadFiltered_GoOnly(t *testing.T) {
	dir := t.TempDir()
	writePattern(t, filepath.Join(dir, "technology", "go"), "go-errors.md", goPattern)
	writePattern(t, filepath.Join(dir, "technology", "python"), "python-types.md", pythonPattern)
	writePattern(t, filepath.Join(dir, "design"), "yagni.md", yagniPattern)

	pats, err := LoadFiltered([]string{"go"}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include Go Errors + YAGNI (no AppliesWhen = always), but NOT Python Types
	names := map[string]bool{}
	for _, p := range pats {
		names[p.Name] = true
	}
	if !names["Go Errors"] {
		t.Error("missing Go Errors pattern")
	}
	if !names["YAGNI"] {
		t.Error("missing YAGNI (design pattern should always apply)")
	}
	if names["Python Types"] {
		t.Error("Python Types should be filtered out for go project")
	}
}

func TestLoadFiltered_NilTags(t *testing.T) {
	dir := t.TempDir()
	writePattern(t, dir, "go-errors.md", goPattern)
	writePattern(t, dir, "python-types.md", pythonPattern)

	pats, err := LoadFiltered(nil, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pats) != 2 {
		t.Errorf("nil tags should return all patterns, got %d", len(pats))
	}
}

func TestLoadFiltered_LegacyAlwaysApplies(t *testing.T) {
	dir := t.TempDir()
	writePattern(t, dir, "legacy.md", legacyPattern)
	writePattern(t, dir, "go-errors.md", goPattern)

	pats, err := LoadFiltered([]string{"python"}, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := map[string]bool{}
	for _, p := range pats {
		names[p.Name] = true
	}
	if !names["Legacy Check"] {
		t.Error("legacy pattern (no AppliesWhen) should always apply")
	}
	if names["Go Errors"] {
		t.Error("Go Errors should be filtered out for python project")
	}
}

func TestLoad_PriorityOverride(t *testing.T) {
	general := t.TempDir()
	repo := t.TempDir()

	// Same pattern name, different severity
	writePattern(t, general, "check.md", `# Review Pattern: Check

**Review-Area**: quality
**Detection-Hint**: check
**Severity**: INFO

## What to check

General version.
`)
	writePattern(t, repo, "check.md", `# Review Pattern: Check

**Review-Area**: quality
**Detection-Hint**: check
**Severity**: CRITICAL

## What to check

Repo-specific version.
`)

	pats, err := Load(general, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pats) != 1 {
		t.Fatalf("expected 1 pattern after dedup, got %d", len(pats))
	}
	if pats[0].Severity != "CRITICAL" {
		t.Errorf("repo-specific pattern should override: Severity = %q, want CRITICAL", pats[0].Severity)
	}
}

func TestLoad_SkipsSourcesMD(t *testing.T) {
	dir := t.TempDir()
	writePattern(t, dir, "SOURCES.md", "# Best Practice Sources\n\nNot a pattern.\n")
	writePattern(t, dir, "go-errors.md", goPattern)

	pats, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pats) != 1 {
		t.Errorf("expected 1 pattern (SOURCES.md skipped), got %d", len(pats))
	}
}

func TestLoad_NonexistentDir(t *testing.T) {
	pats, err := Load("/nonexistent/path")
	if err != nil {
		t.Fatalf("nonexistent dir should not error: %v", err)
	}
	if len(pats) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(pats))
	}
}

func TestFormatGroupedForPrompt(t *testing.T) {
	pats := []Pattern{
		{Name: "Go Errors", Category: "technology", ReviewArea: "quality", DetectionHint: "check", Severity: "WARNING", Body: "## Check\nWrap."},
		{Name: "YAGNI", Category: "design-principle", ReviewArea: "architecture", DetectionHint: "check", Severity: "INFO", Body: "## Check\nDon't."},
		{Name: "Legacy", Category: "", ReviewArea: "quality", DetectionHint: "check", Severity: "INFO", Body: "## Check\nStuff."},
	}

	out := FormatGroupedForPrompt(pats)

	if !strings.Contains(out, "<technology-patterns>") {
		t.Error("should contain technology-patterns tag")
	}
	if !strings.Contains(out, "<design-patterns>") {
		t.Error("should contain design-patterns tag")
	}
	if !strings.Contains(out, "<project-patterns>") {
		t.Error("should contain project-patterns tag")
	}
	if !strings.Contains(out, "Go Errors") {
		t.Error("should contain Go Errors")
	}
	if !strings.Contains(out, "YAGNI") {
		t.Error("should contain YAGNI")
	}
}

func TestFormatGroupedForPrompt_Empty(t *testing.T) {
	out := FormatGroupedForPrompt(nil)
	if out != "" {
		t.Errorf("empty patterns should return empty string, got %q", out)
	}
}

func TestTruncatePatterns(t *testing.T) {
	// Create more than MaxPatternsInPrompt patterns
	var pats []Pattern
	for i := 0; i < MaxPatternsInPrompt+10; i++ {
		sev := "INFO"
		if i < 5 {
			sev = "BLOCKING"
		} else if i < 15 {
			sev = "CRITICAL"
		}
		pats = append(pats, Pattern{
			Name:     "P" + string(rune('A'+i%26)),
			Severity: sev,
		})
	}

	result := truncatePatterns(pats)
	if len(result) != MaxPatternsInPrompt {
		t.Errorf("truncated to %d, want %d", len(result), MaxPatternsInPrompt)
	}

	// All BLOCKING patterns should be present
	blocking := 0
	for _, p := range result {
		if p.Severity == "BLOCKING" {
			blocking++
		}
	}
	if blocking != 5 {
		t.Errorf("expected all 5 BLOCKING patterns, got %d", blocking)
	}
}
