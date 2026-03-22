package report

import (
	"encoding/json"
	"testing"
)

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input   string
		want    Severity
		wantErr bool
	}{
		{"BLOCKING", SeverityBlocking, false},
		{"blocking", SeverityBlocking, false},
		{"CRITICAL", SeverityCritical, false},
		{"critical", SeverityCritical, false},
		{"WARNING", SeverityWarning, false},
		{"warning", SeverityWarning, false},
		{"INFO", SeverityInfo, false},
		{"info", SeverityInfo, false},
		{"unknown", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSeverity(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseSeverity(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveFixClass(t *testing.T) {
	tests := []struct {
		input Actionability
		want  FixClass
	}{
		{ActionabilityAutoFix, FixClassAutoFix},
		{ActionabilityNeedsDiscussion, FixClassAsk},
		{ActionabilityArchitectural, FixClassAsk},
		{"", FixClassAsk},
		{"unknown", FixClassAsk},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := DeriveFixClass(tt.input)
			if got != tt.want {
				t.Errorf("DeriveFixClass(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSeverityUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Severity
		wantErr bool
	}{
		{"uppercase", `"CRITICAL"`, SeverityCritical, false},
		{"lowercase", `"warning"`, SeverityWarning, false},
		{"mixed case", `"Blocking"`, SeverityBlocking, false},
		{"with whitespace", `" info "`, SeverityInfo, false},
		{"unknown passes through", `"custom"`, Severity("CUSTOM"), false},
		{"invalid json", `123`, Severity(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Severity
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindingUnmarshalNormalizesSeverity(t *testing.T) {
	input := `{"severity": "critical", "title": "test", "problem": "p", "action": "a"}`
	var f Finding
	if err := json.Unmarshal([]byte(input), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Severity != SeverityCritical {
		t.Errorf("Finding.Severity = %q, want %q", f.Severity, SeverityCritical)
	}
}

func TestMeetsMinimum(t *testing.T) {
	tests := []struct {
		sev  Severity
		min  Severity
		want bool
	}{
		{SeverityBlocking, SeverityInfo, true},
		{SeverityCritical, SeverityInfo, true},
		{SeverityWarning, SeverityInfo, true},
		{SeverityInfo, SeverityInfo, true},
		{SeverityInfo, SeverityWarning, false},
		{SeverityWarning, SeverityCritical, false},
		{SeverityBlocking, SeverityBlocking, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.sev)+"_meets_"+string(tt.min), func(t *testing.T) {
			if got := tt.sev.MeetsMinimum(tt.min); got != tt.want {
				t.Errorf("%s.MeetsMinimum(%s) = %v, want %v", tt.sev, tt.min, got, tt.want)
			}
		})
	}
}
