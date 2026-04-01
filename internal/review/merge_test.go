package review

import (
	"testing"

	"github.com/planwerk/planwerk-review/internal/report"
)

func TestMergeResults_NilAdversarial(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{{ID: "C-001", Severity: report.SeverityCritical, File: "main.go"}},
		Summary:  "Test",
	}
	result := mergeResults(primary, nil)
	if len(result.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestMergeResults_EmptyAdversarial(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{{ID: "C-001", Severity: report.SeverityCritical, File: "main.go"}},
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
			{ID: "C-001", Severity: report.SeverityCritical, File: "main.go", Line: 10},
		},
		Summary: "Primary review",
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: report.SeverityWarning, File: "handler.go", Line: 20, Title: "SSRF vector"},
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
			{ID: "W-001", Severity: report.SeverityWarning, File: "auth.go", Line: 42, Title: "Weak check"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: report.SeverityCritical, File: "auth.go", Line: 42, Title: "Weak check"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (deduplicated), got %d", len(result.Findings))
	}
	if result.Findings[0].Severity != report.SeverityCritical {
		t.Errorf("expected higher severity CRITICAL, got %s", result.Findings[0].Severity)
	}
	if result.Findings[0].ID != "W-001" {
		t.Errorf("expected preserved ID W-001, got %s", result.Findings[0].ID)
	}
}

func TestMergeResults_DuplicateKeepsPrimaryWhenEqual(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{ID: "C-001", Severity: report.SeverityCritical, File: "db.go", Line: 10, Title: "SQL injection"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: report.SeverityCritical, File: "db.go", Line: 10, Title: "SQL injection"},
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
			{ID: "W-001", Severity: report.SeverityWarning, File: "auth.go", Line: 42, Title: "Missing nil check"},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: report.SeverityCritical, File: "auth.go", Line: 42, Title: "Auth bypass"},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings (different titles), got %d", len(result.Findings))
	}
}

func TestMergeResults_PreservesEnrichment(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{
				ID:          "W-001",
				Severity:    report.SeverityWarning,
				File:        "auth.go",
				Line:        42,
				Title:       "Weak check",
				CodeSnippet: "if user != nil {",
				SuggestedFix: "if user != nil && user.IsActive() {",
			},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{
				Severity: report.SeverityCritical,
				File:     "auth.go",
				Line:     42,
				Title:    "Weak check",
				// Adversarial has no code snippet or suggested fix
			},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	f := result.Findings[0]
	if f.Severity != report.SeverityCritical {
		t.Errorf("expected upgraded severity CRITICAL, got %s", f.Severity)
	}
	if f.CodeSnippet != "if user != nil {" {
		t.Errorf("expected preserved CodeSnippet from primary, got %q", f.CodeSnippet)
	}
	if f.SuggestedFix != "if user != nil && user.IsActive() {" {
		t.Errorf("expected preserved SuggestedFix from primary, got %q", f.SuggestedFix)
	}
}

func TestMergeResults_MergesRelatedTo(t *testing.T) {
	primary := &report.ReviewResult{
		Findings: []report.Finding{
			{
				ID:        "W-001",
				Severity:  report.SeverityWarning,
				File:      "auth.go",
				Line:      42,
				Title:     "Weak check",
				RelatedTo: []string{"C-002"},
			},
		},
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{
				Severity:  report.SeverityCritical,
				File:      "auth.go",
				Line:      42,
				Title:     "Weak check",
				RelatedTo: []string{"C-003"},
			},
		},
	}

	result := mergeResults(primary, adv)
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	related := result.Findings[0].RelatedTo
	if len(related) != 2 {
		t.Fatalf("expected 2 related_to entries, got %d: %v", len(related), related)
	}
	// Should contain both C-003 (from adversarial) and C-002 (merged from primary)
	has := map[string]bool{}
	for _, r := range related {
		has[r] = true
	}
	if !has["C-002"] || !has["C-003"] {
		t.Errorf("expected C-002 and C-003, got %v", related)
	}
}

func TestMergeResults_SummaryUpdated(t *testing.T) {
	primary := &report.ReviewResult{
		Summary: "Clean review",
	}
	adv := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: report.SeverityInfo, File: "util.go", Line: 1},
		},
	}

	result := mergeResults(primary, adv)
	if result.Summary != "Clean review (includes adversarial review pass)" {
		t.Errorf("expected updated summary, got %q", result.Summary)
	}
}
