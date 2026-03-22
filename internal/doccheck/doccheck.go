// Package doccheck detects documentation files that may need updating
// based on code changes in a PR.
package doccheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StaleDocHint describes a documentation file that may need updating.
type StaleDocHint struct {
	DocFile     string
	RelatedDirs []string
}

// Check finds documentation files in the repo root that reference code directories
// which were modified in the PR. Returns nil if no stale docs are detected.
func Check(repoDir, baseBranch string) []StaleDocHint {
	if repoDir == "" || baseBranch == "" {
		return nil
	}

	changedDirs := getChangedDirs(repoDir, baseBranch)
	if len(changedDirs) == 0 {
		return nil
	}

	docFiles := findDocFiles(repoDir)
	if len(docFiles) == 0 {
		return nil
	}

	var hints []StaleDocHint
	for _, docFile := range docFiles {
		content, err := os.ReadFile(filepath.Join(repoDir, docFile))
		if err != nil {
			continue
		}
		docContent := string(content)

		var relatedDirs []string
		for _, dir := range changedDirs {
			if strings.Contains(docContent, dir) {
				relatedDirs = append(relatedDirs, dir)
			}
		}
		if len(relatedDirs) > 0 {
			hints = append(hints, StaleDocHint{
				DocFile:     docFile,
				RelatedDirs: relatedDirs,
			})
		}
	}

	return hints
}

// getChangedDirs returns unique directory paths of files changed in the PR.
func getChangedDirs(repoDir, baseBranch string) []string {
	cmd := exec.Command("git", "diff", "--name-only", "origin/"+baseBranch+"...HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var dirs []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		dir := filepath.Dir(line)
		if dir == "." {
			continue
		}
		// Add the directory and its parents
		for dir != "." && dir != "" {
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
			dir = filepath.Dir(dir)
		}
	}
	return dirs
}

// findDocFiles returns markdown files in the repo root directory.
func findDocFiles(repoDir string) []string {
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil
	}

	var docs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			docs = append(docs, name)
		}
	}
	return docs
}
