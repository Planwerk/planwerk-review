package report

import "testing"

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
