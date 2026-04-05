package audit

import (
	"testing"

	"github.com/planwerk/planwerk-review/internal/report"
)

func TestGroupFindings_SamePatternAndFile(t *testing.T) {
	findings := []report.Finding{
		{Pattern: "err-handling", File: "a.go", Line: 10, Severity: report.SeverityWarning, Title: "t1"},
		{Pattern: "err-handling", File: "a.go", Line: 20, Severity: report.SeverityWarning, Title: "t2"},
		{Pattern: "err-handling", File: "a.go", Line: 5, Severity: report.SeverityCritical, Title: "t3"},
	}

	groups := GroupFindings(findings)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	g := groups[0]
	if len(g.Findings) != 3 {
		t.Errorf("expected 3 findings in group, got %d", len(g.Findings))
	}
	if g.MaxSeverity != report.SeverityCritical {
		t.Errorf("MaxSeverity = %s, want CRITICAL", g.MaxSeverity)
	}
	// Findings should be sorted by line
	wantLines := []int{5, 10, 20}
	for i, f := range g.Findings {
		if f.Line != wantLines[i] {
			t.Errorf("findings[%d].Line = %d, want %d", i, f.Line, wantLines[i])
		}
	}
}

func TestGroupFindings_DifferentFilesProduceSeparateGroups(t *testing.T) {
	findings := []report.Finding{
		{Pattern: "p", File: "a.go", Line: 1, Severity: report.SeverityInfo},
		{Pattern: "p", File: "b.go", Line: 1, Severity: report.SeverityInfo},
	}

	groups := GroupFindings(findings)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Deterministic ordering: file a.go before b.go
	if groups[0].File != "a.go" || groups[1].File != "b.go" {
		t.Errorf("unexpected ordering: %q, %q", groups[0].File, groups[1].File)
	}
}

func TestGroupFindings_MissingPatternFallsBackToTitle(t *testing.T) {
	findings := []report.Finding{
		{Pattern: "", Title: "Unused var", File: "x.go", Line: 3, Severity: report.SeverityInfo},
		{Pattern: "", Title: "Unused var", File: "x.go", Line: 7, Severity: report.SeverityInfo},
	}

	groups := GroupFindings(findings)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group (title fallback), got %d", len(groups))
	}
	if groups[0].Pattern != "Unused var" {
		t.Errorf("Pattern = %q, want title fallback %q", groups[0].Pattern, "Unused var")
	}
}

func TestGroupFindings_SortingBySeverity(t *testing.T) {
	findings := []report.Finding{
		{Pattern: "a", File: "a.go", Line: 1, Severity: report.SeverityInfo},
		{Pattern: "b", File: "b.go", Line: 1, Severity: report.SeverityBlocking},
		{Pattern: "c", File: "c.go", Line: 1, Severity: report.SeverityWarning},
	}

	groups := GroupFindings(findings)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	want := []report.Severity{report.SeverityBlocking, report.SeverityWarning, report.SeverityInfo}
	for i, g := range groups {
		if g.MaxSeverity != want[i] {
			t.Errorf("groups[%d].MaxSeverity = %s, want %s", i, g.MaxSeverity, want[i])
		}
	}
}

func TestFilterBySeverity(t *testing.T) {
	groups := []FindingGroup{
		{MaxSeverity: report.SeverityBlocking},
		{MaxSeverity: report.SeverityCritical},
		{MaxSeverity: report.SeverityWarning},
		{MaxSeverity: report.SeverityInfo},
	}

	got := FilterBySeverity(groups, report.SeverityWarning)
	if len(got) != 3 {
		t.Errorf("len(got) = %d, want 3 (blocking, critical, warning)", len(got))
	}

	got = FilterBySeverity(groups, report.SeverityBlocking)
	if len(got) != 1 {
		t.Errorf("len(got) = %d, want 1 (only blocking)", len(got))
	}

	// Empty severity passes all through
	got = FilterBySeverity(groups, "")
	if len(got) != 4 {
		t.Errorf("empty severity should pass all, got %d", len(got))
	}
}
