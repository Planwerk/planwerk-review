package patterns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load reads all .md files from the given directories and parses them as patterns.
// Directories are processed in order; later directories have higher priority
// (repo-specific patterns override general patterns with the same name).
func Load(dirs ...string) ([]Pattern, error) {
	seen := make(map[string]int) // pattern name -> index in result
	var result []Pattern

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading pattern directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading pattern %s: %w", entry.Name(), err)
			}

			p, err := Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("parsing pattern %s: %w", entry.Name(), err)
			}

			if idx, ok := seen[p.Name]; ok {
				result[idx] = p // override with higher-priority pattern
			} else {
				seen[p.Name] = len(result)
				result = append(result, p)
			}
		}
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
