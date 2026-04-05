package detect

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxWalkDepth limits recursive directory scanning to avoid vendor/node_modules trees.
const maxWalkDepth = 4

// maxYAMLScan is the maximum number of YAML files to inspect for Kubernetes markers.
const maxYAMLScan = 20

// maxYAMLLines is the maximum number of lines to read per YAML file.
const maxYAMLLines = 50

// Technologies scans the given directory and returns a sorted list of
// technology tags describing the project (e.g. "go", "kubernetes", "python").
func Technologies(dir string) []string {
	tags := make(map[string]bool)

	// Simple file-existence checks
	fileChecks := []struct {
		tag   string
		paths []string
	}{
		{"go", []string{"go.mod"}},
		{"python", []string{"pyproject.toml", "setup.py", "requirements.txt", "Pipfile"}},
		{"rust", []string{"Cargo.toml"}},
		{"docker", []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml"}},
	}

	for _, fc := range fileChecks {
		for _, p := range fc.paths {
			if fileExists(filepath.Join(dir, p)) {
				tags[fc.tag] = true
				break
			}
		}
	}

	// TypeScript vs JavaScript
	hasTSConfig := fileExists(filepath.Join(dir, "tsconfig.json"))
	hasPackageJSON := fileExists(filepath.Join(dir, "package.json"))
	if hasTSConfig {
		tags["typescript"] = true
	} else if hasPackageJSON {
		tags["javascript"] = true
	}

	// Java/JVM
	for _, p := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if fileExists(filepath.Join(dir, p)) {
			tags["java"] = true
			break
		}
	}

	// GitHub Actions
	if dirExists(filepath.Join(dir, ".github", "workflows")) {
		entries, err := os.ReadDir(filepath.Join(dir, ".github", "workflows"))
		if err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".yml") || strings.HasSuffix(e.Name(), ".yaml") {
					tags["github-actions"] = true
					break
				}
			}
		}
	}

	// Walk-based detection: Helm, Terraform, Kubernetes
	yamlScanned := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(dir, path)
		depth := strings.Count(rel, string(os.PathSeparator))
		if depth > maxWalkDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// Skip vendor and node_modules
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "node_modules" || base == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		name := d.Name()

		// Helm: Chart.yaml
		if name == "Chart.yaml" {
			tags["helm"] = true
		}

		// Kubernetes: YAML with apiVersion + kind
		if !tags["kubernetes"] && yamlScanned < maxYAMLScan {
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				if looksLikeKubernetes(path) {
					tags["kubernetes"] = true
				}
				yamlScanned++
			}
		}

		return nil
	})

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

// looksLikeKubernetes reads the first maxYAMLLines lines of a YAML file
// and checks for both "apiVersion:" and "kind:" markers.
func looksLikeKubernetes(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	hasAPIVersion := false
	hasKind := false
	lines := 0

	for scanner.Scan() && lines < maxYAMLLines {
		line := scanner.Text()
		if strings.HasPrefix(line, "apiVersion:") {
			hasAPIVersion = true
		}
		if strings.HasPrefix(line, "kind:") {
			hasKind = true
		}
		if hasAPIVersion && hasKind {
			return true
		}
		lines++
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
