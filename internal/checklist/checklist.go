// Package checklist provides the review checklist used in the Claude prompt.
// The default checklist is embedded at compile time and can be overridden
// per-repo via .planwerk/checklist.md.
package checklist

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

// maxFileSize is the maximum size of a checklist override file (64 KB).
const maxFileSize = 64 * 1024

//go:embed checklist.md
var defaultChecklist string

// Load resolves the review checklist with the following priority:
//  1. .planwerk/checklist.md in the reviewed repo (repoDir) — per-repo override
//  2. Embedded default checklist shipped with the binary
//
// Override files larger than 64 KB are ignored with a warning on stderr.
func Load(repoDir string) string {
	if repoDir != "" {
		repoChecklist := filepath.Join(repoDir, ".planwerk", "checklist.md")
		if info, err := os.Stat(repoChecklist); err == nil {
			if info.Size() > maxFileSize {
				fmt.Fprintf(os.Stderr, "Warning: %s exceeds 64 KB limit (%d bytes), using default checklist\n", repoChecklist, info.Size())
			} else if data, err := os.ReadFile(repoChecklist); err == nil && len(data) > 0 {
				return string(data)
			}
		}
	}
	return defaultChecklist
}

// Default returns the embedded default checklist content.
func Default() string {
	return defaultChecklist
}
