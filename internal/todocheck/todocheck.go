// Package todocheck reads TODOS.md from a repository and provides its content
// for cross-referencing with PR changes.
package todocheck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// maxFileSize is the maximum size of a TODO file (64 KB).
const maxFileSize = 64 * 1024

// Load reads TODOS.md from the repo root directory.
// Returns empty string if the file does not exist or cannot be read.
// Also checks for common variants: TODO.md, todos.md.
// Files larger than 64 KB are ignored with a warning on stderr.
func Load(repoDir string) string {
	if repoDir == "" {
		return ""
	}

	candidates := []string{"TODOS.md", "TODO.md", "todos.md"}
	for _, name := range candidates {
		path := filepath.Join(repoDir, name)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.Size() > maxFileSize {
			fmt.Fprintf(os.Stderr, "Warning: %s exceeds 64 KB limit (%d bytes), skipping\n", path, info.Size())
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}
