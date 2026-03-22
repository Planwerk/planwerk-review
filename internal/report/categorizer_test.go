package report

import "testing"

func TestCategorize(t *testing.T) {
	findings := []Finding{
		{Severity: "BLOCKING", Title: "b1"},
		{Severity: "CRITICAL", Title: "c1"},
		{Severity: "CRITICAL", Title: "c2"},
		{Severity: "WARNING", Title: "w1"},
		{Severity: "INFO", Title: "i1"},
		{Severity: "INFO", Title: "i2"},
	}

	cf := Categorize(findings, SeverityInfo)
	if len(cf.Blocking) != 1 {
		t.Errorf("Blocking = %d, want 1", len(cf.Blocking))
	}
	if len(cf.Critical) != 2 {
		t.Errorf("Critical = %d, want 2", len(cf.Critical))
	}
	if len(cf.Warning) != 1 {
		t.Errorf("Warning = %d, want 1", len(cf.Warning))
	}
	if len(cf.Info) != 2 {
		t.Errorf("Info = %d, want 2", len(cf.Info))
	}
	if cf.Total() != 6 {
		t.Errorf("Total = %d, want 6", cf.Total())
	}
}

func TestCategorize_MinSeverity(t *testing.T) {
	findings := []Finding{
		{Severity: "BLOCKING", Title: "b1"},
		{Severity: "WARNING", Title: "w1"},
		{Severity: "INFO", Title: "i1"},
	}

	cf := Categorize(findings, SeverityWarning)
	if len(cf.Blocking) != 1 {
		t.Errorf("Blocking = %d, want 1", len(cf.Blocking))
	}
	if len(cf.Warning) != 1 {
		t.Errorf("Warning = %d, want 1", len(cf.Warning))
	}
	if len(cf.Info) != 0 {
		t.Errorf("Info = %d, want 0 (filtered by min severity)", len(cf.Info))
	}
}

func TestHasBlockersOrCritical(t *testing.T) {
	empty := CategorizedFindings{}
	if empty.HasBlockersOrCritical() {
		t.Error("empty should not have blockers")
	}

	withBlocking := CategorizedFindings{Blocking: []Finding{{}}}
	if !withBlocking.HasBlockersOrCritical() {
		t.Error("should have blockers when Blocking is non-empty")
	}

	withCritical := CategorizedFindings{Critical: []Finding{{}}}
	if !withCritical.HasBlockersOrCritical() {
		t.Error("should have blockers when Critical is non-empty")
	}

	warningsOnly := CategorizedFindings{Warning: []Finding{{}}}
	if warningsOnly.HasBlockersOrCritical() {
		t.Error("warnings-only should not have blockers")
	}
}
