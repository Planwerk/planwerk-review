package report

import (
	"fmt"
	"strings"
)

// FormatInlineComment formats a Finding as a GitHub inline review comment body.
// For auto-fix findings with a SuggestedFix, it includes a GitHub suggestion block.
func FormatInlineComment(f Finding) string {
	var sb strings.Builder

	// Header: ID, title, severity, fix class
	header := fmt.Sprintf("**%s: %s** | %s", f.ID, f.Title, f.Severity)
	if f.FixClass != "" {
		header += fmt.Sprintf(" | %s", f.FixClass)
	}
	sb.WriteString(header)
	sb.WriteString("\n\n")

	// Problem
	sb.WriteString(f.Problem)
	sb.WriteString("\n")

	// For auto-fix findings with a suggested fix, use GitHub's suggestion syntax
	if f.Actionability == ActionabilityAutoFix && f.SuggestedFix != "" {
		sb.WriteString("\n```suggestion\n")
		sb.WriteString(f.SuggestedFix)
		sb.WriteString("\n```\n")
	} else if f.Action != "" {
		sb.WriteString("\n**Action**: ")
		sb.WriteString(f.Action)
		sb.WriteString("\n")
	}

	return sb.String()
}
