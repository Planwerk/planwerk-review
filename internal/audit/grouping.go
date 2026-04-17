package audit

import (
	"sort"

	"github.com/planwerk/planwerk-review/internal/report"
)

// FindingGroup bundles findings that share the same pattern and file. All
// findings in a group are treated as one issue candidate to avoid issue
// flooding when the same pattern triggers many times in one location.
type FindingGroup struct {
	Key         string          // "<pattern>|<file>" or "<title>|<file>" when pattern is empty
	Pattern     string          // pattern name, or the finding title when pattern is empty
	File        string          // file path
	MaxSeverity report.Severity // highest severity among findings in the group
	Findings    []report.Finding
}

// severityRank mirrors the order used in report.severityOrder: lower is worse.
var severityRank = map[report.Severity]int{
	report.SeverityBlocking: 0,
	report.SeverityCritical: 1,
	report.SeverityWarning:  2,
	report.SeverityInfo:     3,
}

// GroupFindings groups findings by (Pattern, File). When a finding has no
// Pattern, it is grouped by (Title, File) instead. The returned slice is
// deterministically ordered: by MaxSeverity (worst first), then File, then
// Pattern. Findings within each group are sorted by Line.
func GroupFindings(findings []report.Finding) []FindingGroup {
	byKey := make(map[string]*FindingGroup)

	for _, f := range findings {
		pattern := f.Pattern
		if pattern == "" {
			pattern = f.Title
		}
		key := pattern + "|" + f.File

		g, ok := byKey[key]
		if !ok {
			g = &FindingGroup{
				Key:         key,
				Pattern:     pattern,
				File:        f.File,
				MaxSeverity: f.Severity,
			}
			byKey[key] = g
		}
		if rank(f.Severity) < rank(g.MaxSeverity) {
			g.MaxSeverity = f.Severity
		}
		g.Findings = append(g.Findings, f)
	}

	groups := make([]FindingGroup, 0, len(byKey))
	for _, g := range byKey {
		sort.Slice(g.Findings, func(i, j int) bool {
			return g.Findings[i].Line < g.Findings[j].Line
		})
		groups = append(groups, *g)
	}

	sort.Slice(groups, func(i, j int) bool {
		ri, rj := rank(groups[i].MaxSeverity), rank(groups[j].MaxSeverity)
		if ri != rj {
			return ri < rj
		}
		if groups[i].File != groups[j].File {
			return groups[i].File < groups[j].File
		}
		return groups[i].Pattern < groups[j].Pattern
	})

	return groups
}

// FilterBySeverity returns the subset of groups whose MaxSeverity meets the
// minimum severity threshold.
func FilterBySeverity(groups []FindingGroup, minSeverity report.Severity) []FindingGroup {
	if minSeverity == "" {
		return groups
	}
	out := make([]FindingGroup, 0, len(groups))
	for _, g := range groups {
		if g.MaxSeverity.MeetsMinimum(minSeverity) {
			out = append(out, g)
		}
	}
	return out
}

// rank returns the severity rank, defaulting to the lowest (info) for unknown
// values so they don't accidentally sort above real findings.
func rank(s report.Severity) int {
	if r, ok := severityRank[s]; ok {
		return r
	}
	return severityRank[report.SeverityInfo]
}
