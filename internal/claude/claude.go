package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/planwerk/planwerk-review/internal/doccheck"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/report"
)

const (
	// claudeTimeout is the maximum time allowed for a Claude CLI invocation.
	claudeTimeout = 15 * time.Minute
)

// DefaultBaseBranch is the fallback base branch name when none is specified.
const DefaultBaseBranch = "main"

// runClaude invokes `claude -p <prompt> --output-format json` in the given
// directory and returns the extracted text response.
func runClaude(dir, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), claudeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "json")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return "", fmt.Errorf("claude: %w", err)
	}
	return extractText(out)
}

// ReviewContext holds all context needed to build the review prompt.
type ReviewContext struct {
	Patterns    []patterns.Pattern
	PRTitle     string
	PRBody      string
	Checklist   string                // external checklist content (empty = use built-in)
	CommitLog   string                // git log output for scope drift detection
	StaleDocs   []doccheck.StaleDocHint // documentation files that may need updating
	TodoContent string                // content of TODOS.md if present
}

// Review invokes `claude /review` in the given directory and returns structured findings.
// It runs two Claude calls:
//  1. `claude /review` to get the unstructured review output
//  2. `claude -p` to structure the output into JSON
func Review(dir string, ctx ReviewContext) (*report.ReviewResult, error) {
	// Step 1: Run /review
	rawReview, err := runReview(dir, ctx)
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
func runReview(dir string, rctx ReviewContext) (string, error) {
	return runClaude(dir, buildReviewPrompt(rctx))
}

// buildReviewPrompt constructs a prompt that includes patterns and triggers /review.
func buildReviewPrompt(ctx ReviewContext) string {
	var sb strings.Builder

	// Staff Engineer persona
	sb.WriteString(`You are a Staff Engineer performing a thorough code review. Apply these thinking patterns:
- "What happens at 10x scale?" — Consider load, data volume, and concurrent users
- "What's the blast radius?" — If this code fails, what else breaks?
- "What happens at 3am?" — Is the error path clear? Will oncall understand the logs?
- "Would a new team member understand this?" — Is the intent clear from the code?

`)

	// Review patterns
	if len(ctx.Patterns) > 0 {
		sb.WriteString("Before running the review, consider these additional review patterns. Flag violations of these patterns in your review:\n\n")
		sb.WriteString("<review-patterns>\n")
		sb.WriteString(patterns.FormatAllForPrompt(ctx.Patterns))
		sb.WriteString("</review-patterns>\n\n")
	}

	// Scope Drift Detection
	if ctx.PRTitle != "" || ctx.PRBody != "" || ctx.CommitLog != "" {
		sb.WriteString("## Scope Analysis (run FIRST, before code quality review)\n\n")
		if ctx.PRTitle != "" {
			fmt.Fprintf(&sb, "PR Title: %s\n", ctx.PRTitle)
		}
		if ctx.PRBody != "" {
			sb.WriteString("<pr-body>\n")
			sb.WriteString(ctx.PRBody)
			sb.WriteString("\n</pr-body>\n")
		}
		if ctx.CommitLog != "" {
			sb.WriteString("<commit-log>\n")
			sb.WriteString(ctx.CommitLog)
			sb.WriteString("\n</commit-log>\n")
		}
		sb.WriteString(`
Before reviewing code quality, check:
1. SCOPE CREEP: Are there files changed that seem unrelated to the PR title/description? Also cross-reference with commit messages — do any commits address unrelated concerns? Flag each as WARNING with title "Scope Creep: <file or area>"
2. MISSING REQUIREMENTS: Are there requirements mentioned in the PR description that are NOT addressed in the diff? Flag each as WARNING with title "Missing Requirement: <requirement>"
3. COMMIT COHERENCE: Do the commit messages tell a coherent story? Are there commits that seem to belong to a different PR? Flag as INFO with title "Commit Coherence: <observation>"

`)
	}

	// Review checklist (always provided — checklist.Load() returns embedded default as fallback)
	if ctx.Checklist != "" {
		sb.WriteString(ctx.Checklist)
		sb.WriteString("\n\n")
	}

	// TODO cross-reference
	if ctx.TodoContent != "" {
		sb.WriteString("## TODO Cross-Reference\n\n")
		sb.WriteString("The project has a TODOS.md file. Review the PR against these open items:\n\n")
		sb.WriteString("<todos-content>\n")
		sb.WriteString(ctx.TodoContent)
		sb.WriteString("\n</todos-content>\n\n")
		sb.WriteString("Check:\n")
		sb.WriteString("1. Does this PR complete any TODO items? If so, flag as INFO with title \"TODO Completed: <item>\"\n")
		sb.WriteString("2. Does this PR introduce work that should be tracked as a TODO? If so, flag as INFO with title \"New TODO Needed: <description>\"\n\n")
	}

	// Documentation staleness hints
	if len(ctx.StaleDocs) > 0 {
		sb.WriteString("## Documentation Staleness Hints\n\n")
		sb.WriteString("The following documentation files may need updating based on code changes in this PR:\n\n")
		for _, doc := range ctx.StaleDocs {
			fmt.Fprintf(&sb, "- %s references code in %s which was modified\n", doc.DocFile, strings.Join(doc.RelatedDirs, ", "))
		}
		sb.WriteString("\nConsider flagging as INFO with title \"Stale Documentation: <file>\" if the docs are actually outdated.\n\n")
	}

	// False-positive suppressions
	sb.WriteString(`## Suppressions — DO NOT flag these

- TODO/FIXME comments that reference an issue tracker (e.g. TODO(#123))
- Missing tests for trivial getters/setters or simple delegation methods
- Import ordering or formatting differences (these are handled by formatters)
- Variable naming that follows the project's existing conventions, even if you'd prefer different names
- Missing documentation on unexported/private functions
- Minor style preferences that don't affect correctness or readability
- "X is redundant with Y" when the redundancy is harmless and aids readability (defense in depth)
- Threshold or constant comments that would rot faster than the code they describe
- Assertions that already cover the behavior being tested (e.g. "this assertion could be tighter")
- Consistency-only suggestions ("use X style everywhere") with no correctness impact
- Issues that are already addressed elsewhere in the same diff — read the FULL diff before commenting
- Suggestions to "add logging" when the error path already returns a descriptive error
- "Consider using X library" when the current approach works correctly

`)

	// Verification of claims — anti-hallucination rules
	sb.WriteString(`## Verification of Claims

These rules are MANDATORY. Violating them produces a misleading review.

- NEVER say "this is probably tested" — name the specific test file and test function, or flag as "test coverage unknown"
- NEVER say "this is handled elsewhere" — cite the exact file and line that handles it, or say "not verified"
- NEVER say "the caller validates this" — name the caller and the validation, or say "unverified assumption"
- NEVER assume error handling exists unless you can see it in the diff or trace it in the codebase
- If you are uncertain whether something is a real issue, say "UNVERIFIED: [claim]" rather than presenting it as fact
- When referencing code outside the diff, always prefix with the file path (e.g. "In cmd/main.go:42, ...")

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
	text, err := runClaude("", buildStructurePrompt(rawReview))
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

Actionability classification (determines fix approach):
- auto-fix: A senior engineer would apply this fix without discussion (dead code removal, N+1 query fixes, stale comment cleanup, magic number extraction, missing error wrapping, simple nil checks). These will be marked as AUTO-FIX — an agent should apply them directly.
- needs-discussion: Requires team input before fixing (security fixes, race condition resolutions, API/design changes, anything changing observable behavior). These will be marked as ASK — requires human confirmation.
- architectural: Fundamental design issue that needs a broader conversation (wrong abstraction, missing layer, significant refactor needed). These will be marked as ASK.

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
	counters := map[report.Severity]int{
		report.SeverityBlocking: 0,
		report.SeverityCritical: 0,
		report.SeverityWarning:  0,
		report.SeverityInfo:     0,
	}
	prefixes := map[report.Severity]string{
		report.SeverityBlocking: "B",
		report.SeverityCritical: "C",
		report.SeverityWarning:  "W",
		report.SeverityInfo:     "I",
	}

	for i := range result.Findings {
		sev := report.Severity(strings.ToUpper(string(result.Findings[i].Severity)))
		result.Findings[i].Severity = sev
		result.Findings[i].Actionability = report.NormalizeActionability(string(result.Findings[i].Actionability))
		result.Findings[i].FixClass = report.DeriveFixClass(result.Findings[i].Actionability)
		counters[sev]++
		prefix := prefixes[sev]
		if prefix == "" {
			prefix = "X"
		}
		result.Findings[i].ID = fmt.Sprintf("%s-%03d", prefix, counters[sev])
	}
}
