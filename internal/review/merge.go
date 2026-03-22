package review

import (
	"strings"

	"github.com/planwerk/planwerk-review/internal/report"
)

// mergeResults combines findings from a primary review and an adversarial review.
// It deduplicates by file+line+title, keeping the higher severity finding.
// Adversarial-only findings are appended to the primary result.
func mergeResults(primary, adversarial *report.ReviewResult) *report.ReviewResult {
	if adversarial == nil || len(adversarial.Findings) == 0 {
		return primary
	}

	// Build index of primary findings by file+line+title
	type key struct {
		file  string
		line  int
		title string
	}
	existing := make(map[key]int) // key -> index in primary.Findings
	for i, f := range primary.Findings {
		if f.File != "" {
			existing[key{f.File, f.Line, normalizeTitle(f.Title)}] = i
		}
	}

	severityRank := map[report.Severity]int{
		report.SeverityBlocking: 0,
		report.SeverityCritical: 1,
		report.SeverityWarning:  2,
		report.SeverityInfo:     3,
	}

	for _, af := range adversarial.Findings {
		k := key{af.File, af.Line, normalizeTitle(af.Title)}
		if idx, found := existing[k]; found && af.File != "" {
			// Duplicate — keep higher severity
			existingRank, ok1 := severityRank[primary.Findings[idx].Severity]
			newRank, ok2 := severityRank[af.Severity]
			if ok1 && ok2 && newRank < existingRank {
				// Adversarial finding is higher severity — replace
				af.ID = primary.Findings[idx].ID // preserve ID
				primary.Findings[idx] = af
			}
		} else {
			// New finding from adversarial pass
			primary.Findings = append(primary.Findings, af)
		}
	}

	// Update summary to mention adversarial pass
	if primary.Summary != "" {
		primary.Summary += " (includes adversarial review pass)"
	}

	return primary
}

// normalizeTitle lowercases and trims a title for dedup comparison.
func normalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}
