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
