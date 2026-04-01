package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Renderer struct {
	w io.Writer
}

func NewRenderer(w io.Writer) *Renderer {
	return &Renderer{w: w}
}

func (r *Renderer) RenderJSON(result ReviewResult, minSeverity Severity) error {
	cf := Categorize(result.Findings, minSeverity)
	filtered := ReviewResult{
		Summary:        result.Summary,
		Recommendation: result.Recommendation,
	}
	filtered.Findings = append(filtered.Findings, cf.Blocking...)
	filtered.Findings = append(filtered.Findings, cf.Critical...)
	filtered.Findings = append(filtered.Findings, cf.Warning...)
	filtered.Findings = append(filtered.Findings, cf.Info...)

	enc := json.NewEncoder(r.w)
	enc.SetIndent("", "  ")
	return enc.Encode(filtered)
}

func (r *Renderer) RenderMarkdown(result ReviewResult, pr PRInfo, minSeverity Severity, version string) {
	cf := Categorize(result.Findings, minSeverity)

	_, _ = fmt.Fprintf(r.w, "# Review: %s/%s#%d\n\n", pr.Owner, pr.Repo, pr.Number)
	_, _ = fmt.Fprintf(r.w, "> *%s*  \n", pr.Title)
	_, _ = fmt.Fprintf(r.w, "> Reviewed by planwerk-review %s with Claude CLI\n\n", version)

	// Machine-readable summary for tooling (Claude Code, CI scripts, etc.)
	_, _ = fmt.Fprintf(r.w, "<!-- planwerk-review: blocking=%d critical=%d warning=%d info=%d recommendation=%s -->\n\n",
		len(cf.Blocking), len(cf.Critical), len(cf.Warning), len(cf.Info),
		r.recommendationKey(cf, result.Recommendation))

	r.renderSection("BLOCKING", cf.Blocking)
	r.renderSection("CRITICAL", cf.Critical)
	r.renderSection("WARNING", cf.Warning)
	r.renderSection("INFO", cf.Info)

	r.renderSummary(cf)
	r.renderRecommendation(cf, result.Recommendation)
}

// recommendationKey returns a short machine-readable verdict for the HTML comment.
func (r *Renderer) recommendationKey(cf CategorizedFindings, custom string) string {
	if custom != "" {
		return "CUSTOM"
	}
	if cf.HasBlockersOrCritical() {
		return "HOLD"
	}
	if len(cf.Warning) > 0 {
		return "REVIEW"
	}
	return "MERGE"
}

func (r *Renderer) renderSection(label string, findings []Finding) {
	if len(findings) == 0 {
		return // skip empty sections — no noise for tooling or readers
	}
	_, _ = fmt.Fprintf(r.w, "## %s (%d)\n\n", label, len(findings))
	for i, f := range findings {
		_, _ = fmt.Fprintf(r.w, "### %s: %s\n", f.ID, f.Title)

		// Compact single-line metadata: File — Fix — Confidence — Pattern
		meta := fmt.Sprintf("**File**: `%s`", fileRef(f))
		if f.FixClass != "" {
			meta += fmt.Sprintf(" — **Fix**: %s", f.FixClass)
		}
		if f.Confidence != "" {
			meta += fmt.Sprintf(" — **Confidence**: %s", f.Confidence)
		}
		if f.Pattern != "" {
			meta += fmt.Sprintf(" — **Pattern**: %s", f.Pattern)
		}
		_, _ = fmt.Fprintln(r.w, meta)
		_, _ = fmt.Fprintln(r.w)
		_, _ = fmt.Fprintf(r.w, "**Problem**: %s\n\n", f.Problem)
		if f.CodeSnippet != "" {
			_, _ = fmt.Fprintf(r.w, "**Code**:\n```\n%s\n```\n\n", f.CodeSnippet)
		}
		_, _ = fmt.Fprintf(r.w, "**Action Required**: %s\n\n", f.Action)
		if f.SuggestedFix != "" {
			_, _ = fmt.Fprintf(r.w, "**Suggested Fix**:\n```\n%s\n```\n\n", f.SuggestedFix)
		}
		if len(f.RelatedTo) > 0 {
			_, _ = fmt.Fprintf(r.w, "**Related**: %s\n\n", strings.Join(f.RelatedTo, ", "))
		}

		if i < len(findings)-1 {
			_, _ = fmt.Fprint(r.w, "---\n\n")
		}
	}
	_, _ = fmt.Fprint(r.w, "---\n\n")
}

// fileRef returns "file:line" when a line number is known, otherwise just "file".
func fileRef(f Finding) string {
	if f.Line > 0 && f.LineEnd > 0 && f.LineEnd != f.Line {
		return fmt.Sprintf("%s:%d-%d", f.File, f.Line, f.LineEnd)
	}
	if f.Line > 0 {
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	return f.File
}

func (r *Renderer) renderSummary(cf CategorizedFindings) {
	_, _ = fmt.Fprint(r.w, "## Summary\n\n")
	_, _ = fmt.Fprintln(r.w, "| Severity | Count |")
	_, _ = fmt.Fprintln(r.w, "|----------|-------|")
	_, _ = fmt.Fprintf(r.w, "| BLOCKING | %d |\n", len(cf.Blocking))
	_, _ = fmt.Fprintf(r.w, "| CRITICAL | %d |\n", len(cf.Critical))
	_, _ = fmt.Fprintf(r.w, "| WARNING  | %d |\n", len(cf.Warning))
	_, _ = fmt.Fprintf(r.w, "| INFO     | %d |\n\n", len(cf.Info))
}

func (r *Renderer) renderRecommendation(cf CategorizedFindings, custom string) {
	if custom != "" {
		_, _ = fmt.Fprintf(r.w, "> [!IMPORTANT]\n> **Recommendation**: %s\n", custom)
		return
	}
	var parts []string
	if len(cf.Blocking) > 0 {
		parts = append(parts, fmt.Sprintf("%d BLOCKING", len(cf.Blocking)))
	}
	if len(cf.Critical) > 0 {
		parts = append(parts, fmt.Sprintf("%d CRITICAL", len(cf.Critical)))
	}
	if cf.HasBlockersOrCritical() {
		_, _ = fmt.Fprintf(r.w, "> [!CAUTION]\n> **Do not merge** — %s findings must be resolved first.\n",
			strings.Join(parts, " and "))
	} else if len(cf.Warning) > 0 {
		_, _ = fmt.Fprintf(r.w, "> [!WARNING]\n> **Review before merging** — %d warning(s) should be addressed.\n", len(cf.Warning))
	} else {
		_, _ = fmt.Fprint(r.w, "> [!TIP]\n> **Ready to merge** — no blocking or critical findings.\n")
	}
}

type PRInfo struct {
	Owner  string
	Repo   string
	Number int
	Title  string
}

// dataBlockPayload is the JSON structure embedded in the HTML comment for machine consumption.
type dataBlockPayload struct {
	CommitSHA string    `json:"commit_sha"`
	Findings  []Finding `json:"findings"`
}

// RenderDataBlock returns an HTML comment containing the JSON-encoded findings
// and metadata for machine consumption by tools like Claude Code.
func RenderDataBlock(result ReviewResult, commitSHA string) string {
	payload := dataBlockPayload{
		CommitSHA: commitSHA,
		Findings:  result.Findings,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\n<!-- planwerk-review-data\n%s\n-->\n", string(data))
}
