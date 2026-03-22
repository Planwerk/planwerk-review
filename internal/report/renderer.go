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
	_, _ = fmt.Fprintf(r.w, "> *%s*\n", pr.Title)
	_, _ = fmt.Fprintf(r.w, "> Reviewed by planwerk-review %s with Claude CLI\n\n", version)

	r.renderSection("BLOCKING", cf.Blocking)
	r.renderSection("CRITICAL", cf.Critical)
	r.renderSection("WARNING", cf.Warning)
	r.renderSection("INFO", cf.Info)

	r.renderSummary(cf)
	r.renderRecommendation(cf, result.Recommendation)
}

func (r *Renderer) renderSection(label string, findings []Finding) {
	_, _ = fmt.Fprintf(r.w, "## %s (%d)\n\n", label, len(findings))
	if len(findings) == 0 {
		_, _ = fmt.Fprint(r.w, "No findings.\n\n")
		_, _ = fmt.Fprint(r.w, "---\n\n")
		return
	}
	for _, f := range findings {
		_, _ = fmt.Fprintf(r.w, "### %s: %s\n", f.ID, f.Title)
		if f.Line > 0 {
			_, _ = fmt.Fprintf(r.w, "**File**: `%s:%d`\n", f.File, f.Line)
		} else {
			_, _ = fmt.Fprintf(r.w, "**File**: `%s`\n", f.File)
		}
		if f.Pattern != "" {
			_, _ = fmt.Fprintf(r.w, "**Pattern**: *%s*\n", f.Pattern)
		}
		if f.Actionability != "" {
			_, _ = fmt.Fprintf(r.w, "**Actionability**: %s\n", f.Actionability)
		}
		_, _ = fmt.Fprintln(r.w)
		_, _ = fmt.Fprintf(r.w, "**Problem**: %s\n\n", f.Problem)
		_, _ = fmt.Fprintf(r.w, "**Action Required**: %s\n\n", f.Action)
	}
	_, _ = fmt.Fprint(r.w, "---\n\n")
}

func (r *Renderer) renderSummary(cf CategorizedFindings) {
	_, _ = fmt.Fprint(r.w, "## Summary\n\n")
	_, _ = fmt.Fprintln(r.w, "| Category  | Count |")
	_, _ = fmt.Fprintln(r.w, "|-----------|-------|")
	_, _ = fmt.Fprintf(r.w, "| BLOCKING  | %-5d |\n", len(cf.Blocking))
	_, _ = fmt.Fprintf(r.w, "| CRITICAL  | %-5d |\n", len(cf.Critical))
	_, _ = fmt.Fprintf(r.w, "| WARNING   | %-5d |\n", len(cf.Warning))
	_, _ = fmt.Fprintf(r.w, "| INFO      | %-5d |\n\n", len(cf.Info))
}

func (r *Renderer) renderRecommendation(cf CategorizedFindings, custom string) {
	if custom != "" {
		_, _ = fmt.Fprintf(r.w, "**Recommendation**: %s\n", custom)
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
		_, _ = fmt.Fprintf(r.w, "**Recommendation**: PR should not be merged due to %s findings until they are resolved.\n",
			strings.Join(parts, " and "))
	} else if len(cf.Warning) > 0 {
		_, _ = fmt.Fprintf(r.w, "**Recommendation**: PR can be merged but has %d warnings that should be addressed.\n", len(cf.Warning))
	} else {
		_, _ = fmt.Fprintln(r.w, "**Recommendation**: PR looks good to merge.")
	}
}

type PRInfo struct {
	Owner  string
	Repo   string
	Number int
	Title  string
}
