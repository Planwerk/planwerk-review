package patterns

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MaxPatternsInPrompt is the hard cap on patterns injected into the prompt.
const MaxPatternsInPrompt = 50

// severityOrder maps severity strings to priority (lower = higher priority for truncation).
var severityOrder = map[string]int{
	"BLOCKING": 0,
	"CRITICAL": 1,
	"WARNING":  2,
	"INFO":     3,
}

// Load reads all .md files recursively from the given directories and parses them as patterns.
// Directories are processed in order; later directories have higher priority
// (repo-specific patterns override general patterns with the same name).
// All patterns are returned regardless of technology tags.
func Load(dirs ...string) ([]Pattern, error) {
	return LoadFiltered(nil, dirs...)
}

// LoadFiltered reads patterns from the given directories recursively and
// returns only those that apply to the detected technology tags.
// If tags is nil or empty, all patterns are returned (backward compatible).
func LoadFiltered(tags []string, dirs ...string) ([]Pattern, error) {
	seen := make(map[string]int) // pattern name -> index in result
	var result []Pattern

	for _, dir := range dirs {
		patterns, err := loadDir(dir)
		if err != nil {
			return nil, err
		}

		for _, p := range patterns {
			if idx, ok := seen[p.Name]; ok {
				result[idx] = p // override with higher-priority pattern
			} else {
				seen[p.Name] = len(result)
				result = append(result, p)
			}
		}
	}

	// Filter by technology tags
	if len(tags) > 0 {
		filtered := result[:0]
		for _, p := range result {
			if p.AppliesTo(tags) {
				filtered = append(filtered, p)
			}
		}
		result = filtered
	}

	return result, nil
}

// loadDir recursively reads all .md files from a directory and parses them as patterns.
func loadDir(dir string) ([]Pattern, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading pattern directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	var result []Pattern

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		// Skip SOURCES.md (documentation catalog, not a pattern)
		if strings.EqualFold(d.Name(), "SOURCES.md") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading pattern %s: %w", path, err)
		}

		p, err := Parse(string(content))
		if err != nil {
			// Skip files that don't parse as patterns (e.g. README.md)
			return nil
		}

		result = append(result, p)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// FormatAllForPrompt concatenates all patterns into a single string for prompt inclusion.
func FormatAllForPrompt(patterns []Pattern) string {
	if len(patterns) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, p := range patterns {
		sb.WriteString(p.FormatForPrompt())
		sb.WriteString("\n")
	}
	return sb.String()
}

// FormatGroupedForPrompt groups patterns by category and formats them with XML tags
// for structured prompt injection. Applies the MaxPatternsInPrompt budget.
func FormatGroupedForPrompt(pats []Pattern) string {
	if len(pats) == 0 {
		return ""
	}

	// Apply prompt budget via truncation
	pats = truncatePatterns(pats)

	// Group by category
	var technology, design, general []Pattern
	for _, p := range pats {
		switch p.Category {
		case "technology":
			technology = append(technology, p)
		case "design-principle":
			design = append(design, p)
		default:
			general = append(general, p)
		}
	}

	var sb strings.Builder

	if len(technology) > 0 {
		sb.WriteString("<technology-patterns>\n")
		for _, p := range technology {
			sb.WriteString(p.FormatForPrompt())
			sb.WriteString("\n")
		}
		sb.WriteString("</technology-patterns>\n\n")
	}

	if len(design) > 0 {
		sb.WriteString("<design-patterns>\n")
		for _, p := range design {
			sb.WriteString(p.FormatForPrompt())
			sb.WriteString("\n")
		}
		sb.WriteString("</design-patterns>\n\n")
	}

	if len(general) > 0 {
		sb.WriteString("<project-patterns>\n")
		for _, p := range general {
			sb.WriteString(p.FormatForPrompt())
			sb.WriteString("\n")
		}
		sb.WriteString("</project-patterns>\n\n")
	}

	return sb.String()
}

// truncatePatterns limits the number of patterns to MaxPatternsInPrompt,
// prioritizing by severity (BLOCKING > CRITICAL > WARNING > INFO).
func truncatePatterns(pats []Pattern) []Pattern {
	if len(pats) <= MaxPatternsInPrompt {
		return pats
	}

	// Stable sort by severity priority
	type indexed struct {
		pattern Pattern
		order   int
	}
	items := make([]indexed, len(pats))
	for i, p := range pats {
		ord, ok := severityOrder[p.Severity]
		if !ok {
			ord = 99 // unknown severity goes last
		}
		items[i] = indexed{pattern: p, order: ord}
	}

	// Simple stable selection: pick the first MaxPatternsInPrompt by severity
	// We do a simple multi-pass to preserve original order within same severity
	var result []Pattern
	for targetOrd := 0; targetOrd <= 99 && len(result) < MaxPatternsInPrompt; targetOrd++ {
		for _, item := range items {
			if item.order == targetOrd {
				result = append(result, item.pattern)
				if len(result) >= MaxPatternsInPrompt {
					break
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Warning: %d patterns loaded, truncated to %d for prompt budget\n", len(pats), MaxPatternsInPrompt)
	return result
}
