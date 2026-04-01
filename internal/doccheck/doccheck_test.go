package doccheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDocFiles(t *testing.T) {
	dir := t.TempDir()
	// Plant some files
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CONTRIBUTING.md"), []byte("# Contrib"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	docs := findDocFiles(dir)
	if len(docs) != 2 {
		t.Fatalf("expected 2 .md files, got %d: %v", len(docs), docs)
	}

	found := false
	for _, d := range docs {
		if d == "README.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find README.md")
	}
}

func TestFindDocFiles_NonexistentDir(t *testing.T) {
	docs := findDocFiles("/nonexistent/path")
	if docs != nil {
		t.Error("expected nil for nonexistent dir")
	}
}

func TestCheck_EmptyInputs(t *testing.T) {
	hints := Check("", "main")
	if hints != nil {
		t.Error("expected nil for empty repoDir")
	}

	hints = Check("/some/dir", "")
	if hints != nil {
		t.Error("expected nil for empty baseBranch")
	}
}

func TestGetChangedDirs(t *testing.T) {
	// With a nonexistent repo dir, should return nil
	dirs := getChangedDirs("/nonexistent", "main")
	if dirs != nil {
		t.Error("expected nil for nonexistent dir")
	}
}

func TestCheckNewFeatures_EmptyInputs(t *testing.T) {
	hints := CheckNewFeatures("", "main")
	if hints != nil {
		t.Error("expected nil for empty repoDir")
	}

	hints = CheckNewFeatures("/some/dir", "")
	if hints != nil {
		t.Error("expected nil for empty baseBranch")
	}
}

func TestCheckNewFeatures_NonexistentDir(t *testing.T) {
	hints := CheckNewFeatures("/nonexistent", "main")
	if hints != nil {
		t.Error("expected nil for nonexistent dir")
	}
}

func TestIsTestFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"go test file", "internal/foo/foo_test.go", true},
		{"go source file", "internal/foo/foo.go", false},
		{"python test prefix", "tests/test_foo.py", true},
		{"python test suffix", "tests/foo_test.py", true},
		{"python source", "src/foo.py", false},
		{"jest test ts", "src/foo.test.ts", true},
		{"jest spec ts", "src/foo.spec.ts", true},
		{"jest test tsx", "src/foo.test.tsx", true},
		{"jest spec jsx", "src/foo.spec.jsx", true},
		{"ts source", "src/foo.ts", false},
		{"tests directory", "__tests__/foo.js", true},
		{"readme", "README.md", false},
		{"go main", "cmd/main.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTestFile(tt.file); got != tt.want {
				t.Errorf("isTestFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestIsInternalConfig(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"planwerk config", ".planwerk/checklist.md", true},
		{"github config", ".github/workflows/ci.yml", true},
		{"gitignore", ".gitignore", true},
		{"source file", "internal/foo/foo.go", false},
		{"readme", "README.md", false},
		{"cmd file", "cmd/main.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInternalConfig(tt.file); got != tt.want {
				t.Errorf("isInternalConfig(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}
