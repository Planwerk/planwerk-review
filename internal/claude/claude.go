package claude

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/report"
)

// Review invokes `claude /review` in the given directory and returns structured findings.
// It runs two Claude calls:
//  1. `claude /review` to get the unstructured review output
//  2. `claude -p` to structure the output into JSON
func Review(dir string, pats []patterns.Pattern) (*report.ReviewResult, error) {
	// Step 1: Run /review
	rawReview, err := runReview(dir, pats)
	if err != nil {
		return nil, fmt.Errorf("running /review: %w", err)
	}

	// Step 2: Structure the output into JSON
	result, err := structureReview(rawReview)
	if err != nil {
		return nil, fmt.Errorf("structuring review output: %w", err)
	}

	assignIDs(result)
	return result, nil
}

// runReview invokes `claude -p` with a prompt that includes patterns and the /review command.
func runReview(dir string, pats []patterns.Pattern) (string, error) {
	prompt := buildReviewPrompt(pats)

	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude /review: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("claude /review: %w", err)
	}

	text, err := extractText(out)
	if err != nil {
		return "", err
	}

	return text, nil
}

// buildReviewPrompt constructs a prompt that includes patterns and triggers /review.
func buildReviewPrompt(pats []patterns.Pattern) string {
	var sb strings.Builder

	if len(pats) > 0 {
		sb.WriteString("Before running the review, consider these additional review patterns. Flag violations of these patterns in your review:\n\n")
		sb.WriteString("<review-patterns>\n")
		sb.WriteString(patterns.FormatAllForPrompt(pats))
		sb.WriteString("</review-patterns>\n\n")
	}

	sb.WriteString("IMPORTANT: Completely ignore all changes in the .planwerk/ directory. Do not create any findings for files inside .planwerk/. These are project management artifacts that are always expected in the diff.\n\n")
	sb.WriteString("/review")

	return sb.String()
}

// structureReview calls Claude to convert unstructured review text into JSON.
func structureReview(rawReview string) (*report.ReviewResult, error) {
	prompt := buildStructurePrompt(rawReview)

	cmd := exec.Command("claude", "-p", prompt, "--output-format", "json")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("claude structuring call: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("claude structuring call: %w", err)
	}

	text, err := extractText(out)
	if err != nil {
		return nil, err
	}

	text = stripMarkdownFences(text)

	var result report.ReviewResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("parsing structured review as JSON: %w\nraw output:\n%s", err, text)
	}

	return &result, nil
}

func buildStructurePrompt(rawReview string) string {
	return `Convert the following code review output into structured JSON. Extract every finding mentioned.

Output ONLY valid JSON matching this exact schema (no markdown fences, no surrounding text):

{
  "findings": [
    {
      "id": "",
      "severity": "BLOCKING|CRITICAL|WARNING|INFO",
      "title": "Short title",
      "file": "path/to/file.go",
      "line": 42,
      "pattern": "Pattern name if triggered by a review pattern, otherwise omit",
      "problem": "Description of the problem",
      "action": "What should be done to fix it"
    }
  ],
  "summary": "Brief overall summary of the review",
  "recommendation": "Whether the PR should be merged and under what conditions"
}

Severity levels:
- BLOCKING: Fundamental architecture or security issues — PR must not be merged
- CRITICAL: Bugs, security vulnerabilities, severe problems — must be fixed before merge
- WARNING: Code quality issues, potential problems — should be fixed
- INFO: Style suggestions, minor improvements — optional

Leave the "id" field as an empty string — it will be assigned automatically.
If there are no findings, return an empty findings array.

<review-output>
` + rawReview + `
</review-output>`
}

type claudeResponse struct {
	Result string `json:"result"`
}

// extractText extracts the text content from Claude's JSON output envelope.
func extractText(raw []byte) (string, error) {
	var resp claudeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		// Fall back to treating the entire output as the response text
		return string(raw), nil
	}
	return resp.Result, nil
}

// stripMarkdownFences removes ```json ... ``` wrapping that LLMs frequently add.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (```json or ```)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// Remove closing fence
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

func assignIDs(result *report.ReviewResult) {
	counters := map[string]int{
		"BLOCKING": 0,
		"CRITICAL": 0,
		"WARNING":  0,
		"INFO":     0,
	}
	prefixes := map[string]string{
		"BLOCKING": "B",
		"CRITICAL": "C",
		"WARNING":  "W",
		"INFO":     "I",
	}

	for i := range result.Findings {
		sev := strings.ToUpper(result.Findings[i].Severity)
		result.Findings[i].Severity = sev
		counters[sev]++
		prefix := prefixes[sev]
		if prefix == "" {
			prefix = "X"
		}
		result.Findings[i].ID = fmt.Sprintf("%s-%03d", prefix, counters[sev])
	}
}
