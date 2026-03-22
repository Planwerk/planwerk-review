package patterns

import (
	"bufio"
	"fmt"
	"strings"
)

type Pattern struct {
	Name          string
	ReviewArea    string
	DetectionHint string
	Severity      string
	Occurrences   int
	Body          string
}

// Parse parses a pattern from its markdown content.
// The format uses a simple frontmatter-like header with key-value pairs.
func Parse(content string) (Pattern, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var p Pattern
	var bodyLines []string
	inBody := false

	for scanner.Scan() {
		line := scanner.Text()

		if !inBody {
			if strings.HasPrefix(line, "# Review Pattern:") {
				p.Name = strings.TrimSpace(strings.TrimPrefix(line, "# Review Pattern:"))
				continue
			}
			if strings.HasPrefix(line, "**Review-Area**:") {
				p.ReviewArea = extractValue(line, "**Review-Area**:")
				continue
			}
			if strings.HasPrefix(line, "**Detection-Hint**:") {
				p.DetectionHint = extractValue(line, "**Detection-Hint**:")
				continue
			}
			if strings.HasPrefix(line, "**Severity**:") {
				p.Severity = extractValue(line, "**Severity**:")
				continue
			}
			if strings.HasPrefix(line, "**Occurrences**:") {
				val := extractValue(line, "**Occurrences**:")
				_, _ = fmt.Sscanf(val, "%d", &p.Occurrences)
				continue
			}
			if strings.HasPrefix(line, "## ") {
				inBody = true
				bodyLines = append(bodyLines, line)
				continue
			}
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	p.Body = strings.TrimSpace(strings.Join(bodyLines, "\n"))

	if p.Name == "" {
		return p, fmt.Errorf("pattern has no name (missing '# Review Pattern: ...' header)")
	}

	return p, nil
}

func extractValue(line, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(line, prefix))
}

// FormatForPrompt returns a concise representation of the pattern suitable for inclusion in a prompt.
func (p Pattern) FormatForPrompt() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### Pattern: %s\n", p.Name)
	fmt.Fprintf(&sb, "- Area: %s\n", p.ReviewArea)
	fmt.Fprintf(&sb, "- Detection: %s\n", p.DetectionHint)
	fmt.Fprintf(&sb, "- Severity: %s\n", p.Severity)
	sb.WriteString(p.Body)
	sb.WriteString("\n")
	return sb.String()
}
