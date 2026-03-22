package review

import (
	"testing"

	"github.com/planwerk/planwerk-review/internal/report"
)

func TestMergeResults_NilAdversarial(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{{ID: "C-001", Severity: "CRITICAL", File: "main.go"}},
		Summary:  "Test",
	}
	result := mergeResults(primary, nil)
	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestMergeResults_EmptyAdversarial(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{{ID: "C-001", Severity: "CRITICAL", File: "main.go"}},
	}
	adv := &report.ReviewResult{}
	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestMergeResults_NewFindings(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{ID: "C-001", Severity: "CRITICAL", File: "main.go", Line: 10},
		},
		Summary: "Primary review",
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "WARNING", File: "handler.go", Line: 20, Title: "SSRF vector"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(result.Findings))
	}
	if result.Findings[1].Title != "SSRF vector" {
		t.Errorf("expected adversarial finding appended, got %q", result.Findings[1].Title)
	}
}

func TestMergeResults_DuplicateKeepsHigherSeverity(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{ID: "W-001", Severity: "WARNING", File: "auth.go", Line: 42, Title: "Weak check"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "CRITICAL", File: "auth.go", Line: 42, Title: "Weak check"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (deduplicated), got %d", len(result.Findings))
	}
	if result.Findings[0].Severity != "CRITICAL" {
		t.Errorf("expected higher severity CRITICAL, got %s", result.Findings[0].Severity)
	}
	if result.Findings[0].ID != "W-001" {
		t.Errorf("expected preserved ID W-001, got %s", result.Findings[0].ID)
	}
}

func TestMergeResults_DuplicateKeepsPrimaryWhenEqual(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{ID: "C-001", Severity: "CRITICAL", File: "db.go", Line: 10, Title: "SQL injection"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "CRITICAL", File: "db.go", Line: 10, Title: "SQL injection"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (deduplicated), got %d", len(result.Findings))
	}
	// Same severity — primary should be kept
	if result.Findings[0].Title != "SQL injection" {
		t.Errorf("expected primary finding preserved, got %q", result.Findings[0].Title)
	}
}

func TestMergeResults_DifferentTitlesSameLocation(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{ID: "W-001", Severity: "WARNING", File: "auth.go", Line: 42, Title: "Missing nil check"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "CRITICAL", File: "auth.go", Line: 42, Title: "Auth bypass"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings (different titles), got %d", len(result.Findings))
	}
}

func TestMergeResults_SummaryUpdated(t *testing.T) {
	primary := &report.ReviewResult{
		Summary: "Clean review",
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "INFO", File: "util.go", Line: 1},
		},
	}

	result := mergeResults(primary, adv)
	if result.Summary != "Clean review (includes adversarial review pass)" {
		t.Errorf("expected updated summary, got %q", result.Summary)
	}
}
