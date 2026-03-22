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
func Review(dir string, pats []patterns.Pattern, prTitle, prBody string) (*report.ReviewResult, error) {
	// Step 1: Run /review
	rawReview, err := runReview(dir, pats, prTitle, prBody)
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
func runReview(dir string, pats []patterns.Pattern, prTitle, prBody string) (string, error) {
	prompt := buildReviewPrompt(pats, prTitle, prBody)

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
func buildReviewPrompt(pats []patterns.Pattern, prTitle, prBody string) string {
	var sb strings.Builder

	// Staff Engineer persona
	sb.WriteString(`You are a Staff Engineer performing a thorough code review. Apply these thinking patterns:
- "What happens at 10x scale?" — Consider load, data volume, and concurrent users
- "What's the blast radius?" — If this code fails, what else breaks?
- "What happens at 3am?" — Is the error path clear? Will oncall understand the logs?
- "Would a new team member understand this?" — Is the intent clear from the code?

`)

	// Review patterns
	if len(pats) > 0 {
		sb.WriteString("Before running the review, consider these additional review patterns. Flag violations of these patterns in your review:\n\n")
		sb.WriteString("<review-patterns>\n")
		sb.WriteString(patterns.FormatAllForPrompt(pats))
		sb.WriteString("</review-patterns>\n\n")
	}

	// Scope Drift Detection
	if prTitle != "" || prBody != "" {
		sb.WriteString("## Scope Analysis (run FIRST, before code quality review)\n\n")
		if prTitle != "" {
			fmt.Fprintf(&sb, "PR Title: %s\n", prTitle)
		}
		if prBody != "" {
			sb.WriteString("<pr-body>\n")
			sb.WriteString(prBody)
			sb.WriteString("\n</pr-body>\n")
		}
		sb.WriteString(`
Before reviewing code quality, check:
1. SCOPE CREEP: Are there files changed that seem unrelated to the PR title/description? Flag each as WARNING with title "Scope Creep: <file or area>"
2. MISSING REQUIREMENTS: Are there requirements mentioned in the PR description that are NOT addressed in the diff? Flag each as WARNING with title "Missing Requirement: <requirement>"

`)
	}

	// Two-pass review checklist
	sb.WriteString(`## Review Checklist (work through systematically)

### Pass 1 — CRITICAL (always check these)
- [ ] SQL & Data Safety: raw queries, missing parameterization, unsafe migrations
- [ ] Race Conditions: shared mutable state, missing locks, concurrent map access
- [ ] Error Handling: swallowed errors, missing nil checks, panic-worthy paths
- [ ] Security: hardcoded secrets, injection vectors, auth/authz gaps
- [ ] Input Validation: unvalidated user input at system boundaries

### Pass 2 — INFORMATIONAL (check if time permits)
- [ ] Magic Numbers: unexplained constants, config that should be externalized
- [ ] Dead Code: unused functions, unreachable branches, commented-out code
- [ ] Test Gaps: untested error paths, missing edge cases
- [ ] Performance: N+1 queries, unbounded allocations, missing pagination
- [ ] API Contract: breaking changes to public interfaces without versioning

`)

	// False-positive suppressions
	sb.WriteString(`## Suppressions — DO NOT flag these

- TODO/FIXME comments that reference an issue tracker (e.g. TODO(#123))
- Missing tests for trivial getters/setters or simple delegation methods
- Import ordering or formatting differences (these are handled by formatters)
- Variable naming that follows the project's existing conventions, even if you'd prefer different names
- Missing documentation on unexported/private functions
- Minor style preferences that don't affect correctness or readability

`)

	// Anti-sycophancy rules
	sb.WriteString(`## Communication Style

Be direct and decisive in your findings. Do NOT hedge:
- Do NOT write "you might want to consider..." — state what IS wrong
- Do NOT write "this could potentially cause..." — state what WILL happen
- Do NOT write "it might be worth looking into..." — state the specific problem
- Take a clear position on every finding. If something is wrong, say it is wrong.
- If something is fine, do not mention it at all.

`)

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
      "actionability": "auto-fix|needs-discussion|architectural",
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

Actionability classification:
- auto-fix: A senior engineer would apply this fix without discussion (dead code, magic numbers, missing error wrapping, stale comments)
- needs-discussion: Requires team input before fixing (security decisions, API changes, race condition fixes that change behavior)
- architectural: Fundamental design issue that needs a broader conversation (wrong abstraction, missing layer, significant refactor needed)

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
		result.Findings[i].Actionability = report.NormalizeActionability(string(result.Findings[i].Actionability))
		counters[sev]++
		prefix := prefixes[sev]
		if prefix == "" {
			prefix = "X"
		}
		result.Findings[i].ID = fmt.Sprintf("%s-%03d", prefix, counters[sev])
	}
}
