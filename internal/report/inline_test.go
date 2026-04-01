package report

import (
	"strings"
	"testing"
)

func TestFormatInlineComment_Normal(t *testing.T) {
	f := Finding{
		ID:       "W-001",
		Severity: SeverityWarning,
		Title:    "Missing error check",
		FixClass: FixClassAsk,
		Problem:  "The returned error from db.Close() is discarded.",
		Action:   "Wrap the error and return it.",
	}

	got := FormatInlineComment(f)

	if !strings.Contains(got, "**W-001: Missing error check**") {
		t.Error("should contain ID and title")
	}
	if !strings.Contains(got, "WARNING") {
		t.Error("should contain severity")
	}
	if !strings.Contains(got, "ASK") {
		t.Error("should contain fix class")
	}
	if !strings.Contains(got, "db.Close()") {
		t.Error("should contain problem text")
	}
	if !strings.Contains(got, "**Action**:") {
		t.Error("should contain action for non-autofix findings")
	}
	if strings.Contains(got, "```suggestion") {
		t.Error("should NOT contain suggestion block for non-autofix findings")
	}
}

func TestFormatInlineComment_AutoFix(t *testing.T) {
	f := Finding{
		ID:            "C-001",
		Severity:      SeverityCritical,
		Title:         "SQL injection",
		Actionability: ActionabilityAutoFix,
		FixClass:      FixClassAutoFix,
		Problem:       "User input interpolated into SQL query.",
		Action:        "Use parameterized query.",
		SuggestedFix:  `rows, err := db.Query("SELECT * FROM users WHERE id = ?", userID)`,
	}

	got := FormatInlineComment(f)

	if !strings.Contains(got, "```suggestion") {
		t.Error("should contain suggestion block for autofix findings")
	}
	if !strings.Contains(got, "db.Query") {
		t.Error("should contain the suggested fix code")
	}
	if strings.Contains(got, "**Action**:") {
		t.Error("should NOT contain action text when suggestion is present")
	}
}

func TestFormatInlineComment_AutoFixNoSuggestedFix(t *testing.T) {
	f := Finding{
		ID:            "W-002",
		Severity:      SeverityWarning,
		Title:         "Dead code",
		Actionability: ActionabilityAutoFix,
		FixClass:      FixClassAutoFix,
		Problem:       "Function is never called.",
		Action:        "Remove the function.",
	}

	got := FormatInlineComment(f)

	// Without a suggested fix, should fall back to action text
	if strings.Contains(got, "```suggestion") {
		t.Error("should NOT contain suggestion block when SuggestedFix is empty")
	}
	if !strings.Contains(got, "**Action**:") {
		t.Error("should contain action text as fallback")
	}
}
