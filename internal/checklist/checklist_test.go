package checklist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	content := Default()
	if content == "" {
		t.Fatal("embedded default checklist is empty")
	}
	if !strings.Contains(content, "CRITICAL") {
		t.Error("default checklist should contain CRITICAL section")
	}
	if !strings.Contains(content, "INFORMATIONAL") {
		t.Error("default checklist should contain INFORMATIONAL section")
	}
}

func TestLoad_NoRepoOverride(t *testing.T) {
	// With a non-existent dir, should return default
	content := Load("/nonexistent/path")
	if content != Default() {
		t.Error("Load with nonexistent dir should return default checklist")
	}
}

func TestLoad_EmptyDir(t *testing.T) {
	content := Load("")
	if content != Default() {
		t.Error("Load with empty dir should return default checklist")
	}
}

func TestLoad_RepoOverride(t *testing.T) {
	tmpDir := t.TempDir()
	planwerkDir := filepath.Join(tmpDir, ".planwerk")
	if err := os.MkdirAll(planwerkDir, 0o755); err != nil {
		t.Fatal(err)
	}

	customChecklist := "## Custom Checklist\n\n- [ ] Custom check item\n"
	if err := os.WriteFile(filepath.Join(planwerkDir, "checklist.md"), []byte(customChecklist), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(tmpDir)
	if content != customChecklist {
		t.Errorf("Load should return repo override, got: %s", content)
	}
}

func TestLoad_EmptyRepoOverrideFallsBack(t *testing.T) {
	tmpDir := t.TempDir()
	planwerkDir := filepath.Join(tmpDir, ".planwerk")
	if err := os.MkdirAll(planwerkDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write an empty file — should fall back to default
	if err := os.WriteFile(filepath.Join(planwerkDir, "checklist.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	content := Load(tmpDir)
	if content != Default() {
		t.Error("Load with empty override file should return default checklist")
	}
}
