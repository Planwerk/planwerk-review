package report

import "fmt"

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

type Finding struct {
	ID       string  `json:"id"`
	Severity string  `json:"severity"`
	Title    string  `json:"title"`
	File     string  `json:"file"`
	Line     int     `json:"line,omitempty"`
	Pattern  string  `json:"pattern,omitempty"`
	Problem  string  `json:"problem"`
	Action   string  `json:"action"`
}

type ReviewResult struct {
	Findings      []Finding `json:"findings"`
	Summary       string    `json:"summary"`
	Recommendation string   `json:"recommendation"`
}
