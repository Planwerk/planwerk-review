package report

import (
	"encoding/json"
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
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "BLOCKING":
		return SeverityBlocking, nil
	case "CRITICAL":
		return SeverityCritical, nil
	case "WARNING":
		return SeverityWarning, nil
	case "INFO":
		return SeverityInfo, nil
	default:
		return "", fmt.Errorf("unknown severity: %q", s)
	}
}

// UnmarshalJSON normalizes severity values during JSON parsing,
// so downstream code always sees uppercase canonical values.
func (s *Severity) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	// Normalize to uppercase; unknown values pass through for downstream handling.
	*s = Severity(strings.ToUpper(strings.TrimSpace(raw)))
	return nil
}

func (s Severity) MeetsMinimum(min Severity) bool {
	return severityOrder[s] <= severityOrder[min]
}

type Actionability string

const (
	ActionabilityAutoFix         Actionability = "auto-fix"
	ActionabilityNeedsDiscussion Actionability = "needs-discussion"
	ActionabilityArchitectural   Actionability = "architectural"
)

// FixClass indicates whether a finding can be auto-fixed or requires user input.
type FixClass string

const (
	FixClassAutoFix FixClass = "AUTO-FIX"
	FixClassAsk     FixClass = "ASK"
)

// DeriveFixClass maps an Actionability value to a FixClass.
// auto-fix → AUTO-FIX, everything else → ASK.
func DeriveFixClass(a Actionability) FixClass {
	if a == ActionabilityAutoFix {
		return FixClassAutoFix
	}
	return FixClassAsk
}

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
// Unknown values default to needs-discussion.
func NormalizeActionability(s string) Actionability {
	if a, ok := validActionability[strings.ToLower(strings.TrimSpace(s))]; ok {
		return a
	}
	return ActionabilityNeedsDiscussion
}

type Finding struct {
	ID            string        `json:"id"`
	Severity      Severity      `json:"severity"`
	Title         string        `json:"title"`
	File          string        `json:"file"`
	Line          int           `json:"line,omitempty"`
	Pattern       string        `json:"pattern,omitempty"`
	Actionability Actionability `json:"actionability,omitempty"`
	FixClass      FixClass      `json:"fix_class,omitempty"`
	Problem       string        `json:"problem"`
	Action        string        `json:"action"`
}

type ReviewResult struct {
	Findings       []Finding `json:"findings"`
	Summary        string    `json:"summary"`
	Recommendation string    `json:"recommendation"`
}
