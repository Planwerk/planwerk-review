package report

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCoverageMap_Empty(t *testing.T) {
	var buf bytes.Buffer
	RenderCoverageMap(&buf, CoverageResult{})
	if !strings.Contains(buf.String(), "No changed functions found") {
		t.Error("expected 'No changed functions found' message")
	}
}

func TestRenderCoverageMap_WithEntries(t *testing.T) {
	result := CoverageResult{
		Entries: []CoverageEntry{
			{
				Function: "buildPrompt()",
				File:     "claude/claude.go",
				Rating:   "★★★",
				TestFile: "claude_test.go",
				TestFunc: "TestBuildPrompt",
			},
			{
				Function:       "Run()",
				File:           "review/reviewer.go",
				Rating:         "★",
				TestFile:       "reviewer_test.go",
				UncoveredPaths: []string{"error paths", "cache miss"},
			},
			{
				Function:       "Load()",
				File:           "checklist/checklist.go",
				Rating:         "GAP",
				UncoveredPaths: []string{"no tests"},
			},
		},
	}

	var buf bytes.Buffer
	RenderCoverageMap(&buf, result)
	output := buf.String()

	if !strings.Contains(output, "## Test Coverage Map") {
		t.Error("expected header")
	}
	if !strings.Contains(output, "buildPrompt()") {
		t.Error("expected function name")
	}
	if !strings.Contains(output, "★★★") {
		t.Error("expected star rating")
	}
	if !strings.Contains(output, "GAP") {
		t.Error("expected GAP rating")
	}
	if !strings.Contains(output, "2/3 functions tested (66%)") {
		t.Errorf("expected coverage summary, got: %s", output)
	}
}

func TestRenderCoverageMap_AllTested(t *testing.T) {
	result := CoverageResult{
		Entries: []CoverageEntry{
			{Function: "Foo()", File: "a.go", Rating: "★★★", TestFile: "a_test.go"},
			{Function: "Bar()", File: "b.go", Rating: "★★", TestFile: "b_test.go"},
		},
	}

	var buf bytes.Buffer
	RenderCoverageMap(&buf, result)
	if !strings.Contains(buf.String(), "2/2 functions tested (100%)") {
		t.Errorf("expected 100%% coverage, got: %s", buf.String())
	}
}
