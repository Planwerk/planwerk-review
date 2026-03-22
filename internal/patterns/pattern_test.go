package patterns

import "testing"

func TestParse(t *testing.T) {
	input := `# Review Pattern: Test Pattern

**Review-Area**: testing
**Detection-Hint**: look for tests
**Severity**: WARNING
**Occurrences**: 3

## What to check

Check for missing tests.
`

	p, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Test Pattern" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Pattern")
	}
	if p.ReviewArea != "testing" {
		t.Errorf("ReviewArea = %q, want %q", p.ReviewArea, "testing")
	}
	if p.DetectionHint != "look for tests" {
		t.Errorf("DetectionHint = %q, want %q", p.DetectionHint, "look for tests")
	}
	if p.Severity != "WARNING" {
		t.Errorf("Severity = %q, want %q", p.Severity, "WARNING")
	}
	if p.Occurrences != 3 {
		t.Errorf("Occurrences = %d, want %d", p.Occurrences, 3)
	}
	if p.Body == "" {
		t.Error("Body should not be empty")
	}
}

func TestParse_NoName(t *testing.T) {
	_, err := Parse("just some content\nwithout a header")
	if err == nil {
		t.Fatal("expected error for pattern without name")
	}
}
