package todocheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	content := Load(dir)
	if content != "" {
		t.Errorf("expected empty string, got %q", content)
	}
}

func TestLoad_EmptyDir(t *testing.T) {
	content := Load("")
	if content != "" {
		t.Errorf("expected empty string, got %q", content)
	}
}

func TestLoad_TODOSmd(t *testing.T) {
	dir := t.TempDir()
	expected := "## TODO\n\n- [ ] Fix bug"
	if err := os.WriteFile(filepath.Join(dir, "TODOS.md"), []byte(expected), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(dir)
	if content != expected {
		t.Errorf("Load() = %q, want %q", content, expected)
	}
}

func TestLoad_TODOmd(t *testing.T) {
	dir := t.TempDir()
	expected := "# TODO\n\n- Item 1"
	if err := os.WriteFile(filepath.Join(dir, "TODO.md"), []byte(expected), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(dir)
	if content != expected {
		t.Errorf("Load() = %q, want %q", content, expected)
	}
}

func TestLoad_PrefersTODOSmd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "TODOS.md"), []byte("TODOS content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "TODO.md"), []byte("TODO content"), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(dir)
	if content != "TODOS content" {
		t.Errorf("Load() should prefer TODOS.md, got %q", content)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "TODOS.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(dir)
	if content != "" {
		t.Errorf("Load() with empty file should return empty string, got %q", content)
	}
}
