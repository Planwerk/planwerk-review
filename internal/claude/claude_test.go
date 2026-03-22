package claude

import (
	"testing"

	"github.com/planwerk/planwerk-review/internal/report"
)

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fences",
			input: `{"findings": []}`,
			want:  `{"findings": []}`,
		},
		{
			name:  "json fences",
			input: "```json\n{\"findings\": []}\n```",
			want:  `{"findings": []}`,
		},
		{
			name:  "plain fences",
			input: "```\n{\"findings\": []}\n```",
			want:  `{"findings": []}`,
		},
		{
			name:  "with surrounding whitespace",
			input: "  \n```json\n{\"findings\": []}\n```\n  ",
			want:  `{"findings": []}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownFences(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownFences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssignIDs(t *testing.T) {
	result := &report.ReviewResult{
		Findings: []report.Finding{
			{Severity: "blocking"},
			{Severity: "critical"},
			{Severity: "critical"},
			{Severity: "warning"},
			{Severity: "info"},
		},
	}

	assignIDs(result)

	expected := []struct {
		id       string
		severity string
	}{
		{"B-001", "BLOCKING"},
		{"C-001", "CRITICAL"},
		{"C-002", "CRITICAL"},
		{"W-001", "WARNING"},
		{"I-001", "INFO"},
	}

	for i, exp := range expected {
		f := result.Findings[i]
		if f.ID != exp.id {
			t.Errorf("finding[%d].ID = %q, want %q", i, f.ID, exp.id)
		}
		if f.Severity != exp.severity {
			t.Errorf("finding[%d].Severity = %q, want %q", i, f.Severity, exp.severity)
		}
	}
}
