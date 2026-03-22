package report

import (
	"fmt"
	"strings"
)

type Severity string

const (
	SeverityBlocking Severity = "BLOCKING"
	SeverityCritical Severity = "CRITICAL"
	SeverityWarning  Severity = "WARNING"
	SeverityInfo     Severity = "INFO"
)

var severityOrder = map[Severity]int{
	SeverityBlocking: 0,
	SeverityCritical: 1,
	SeverityWarning:  2,
	SeverityInfo:     3,
}

func ParseSeverity(s string) (Severity, error) {
	switch s {
	case "BLOCKING", "blocking":
		return SeverityBlocking, nil
	case "CRITICAL", "critical":
		return SeverityCritical, nil
	case "WARNING", "warning":
		return SeverityWarning, nil
	case "INFO", "info":
		return SeverityInfo, nil
	default:
		return "", fmt.Errorf("unknown severity: %q", s)
	}
}

func (s Severity) MeetsMinimum(min Severity) bool {
	return severityOrder[s] <= severityOrder[min]
}

type Actionability string

const (
	ActionabilityAutoFix          Actionability = "auto-fix"
	ActionabilityNeedsDiscussion  Actionability = "needs-discussion"
	ActionabilityArchitectural    Actionability = "architectural"
)

var validActionability = map[string]Actionability{
	"auto-fix":         ActionabilityAutoFix,
	"autofix":          ActionabilityAutoFix,
	"auto_fix":         ActionabilityAutoFix,
	"needs-discussion": ActionabilityNeedsDiscussion,
	"needs_discussion": ActionabilityNeedsDiscussion,
	"needsdiscussion":  ActionabilityNeedsDiscussion,
	"architectural":    ActionabilityArchitectural,
}

// NormalizeActionability maps common variants to the canonical value.
// Unknown values are returned as empty string.
func NormalizeActionability(s string) Actionability {
	if a, ok := validActionability[strings.ToLower(strings.TrimSpace(s))]; ok {
		return a
	}
	return ""
}

type Finding struct {
	ID            string        `json:"id"`
	Severity      string        `json:"severity"`
	Title         string        `json:"title"`
	File          string        `json:"file"`
	Line          int           `json:"line,omitempty"`
	Pattern       string        `json:"pattern,omitempty"`
	Actionability Actionability `json:"actionability,omitempty"`
	Problem       string        `json:"problem"`
	Action        string        `json:"action"`
}

type ReviewResult struct {
	Findings      []Finding `json:"findings"`
	Summary       string    `json:"summary"`
	Recommendation string   `json:"recommendation"`
}
