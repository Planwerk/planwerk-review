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

func TestDefault_ContainsTestCoverageCheck(t *testing.T) {
	content := Default()
	if !strings.Contains(content, "Test Coverage for New Code") {
		t.Error("default checklist should contain 'Test Coverage for New Code' item")
	}
}

func TestDefault_ContainsDocumentationCheck(t *testing.T) {
	content := Default()
	if !strings.Contains(content, "Documentation Completeness") {
		t.Error("default checklist should contain 'Documentation Completeness' item")
	}
}

func TestDefault_TestCoverageInSemanticPass(t *testing.T) {
	content := Default()
	semanticIdx := strings.Index(content, "SEMANTIC")
	infoIdx := strings.Index(content, "INFORMATIONAL")
	testCoverageIdx := strings.Index(content, "Test Coverage for New Code")
	if semanticIdx == -1 || infoIdx == -1 || testCoverageIdx == -1 {
		t.Fatal("missing expected sections in checklist")
	}
	if testCoverageIdx < semanticIdx || testCoverageIdx > infoIdx {
		t.Error("'Test Coverage for New Code' should be in SEMANTIC pass (between SEMANTIC and INFORMATIONAL)")
	}
}

func TestDefault_DocCompletionInSemanticPass(t *testing.T) {
	content := Default()
	semanticIdx := strings.Index(content, "SEMANTIC")
	infoIdx := strings.Index(content, "INFORMATIONAL")
	docIdx := strings.Index(content, "Documentation Completeness")
	if semanticIdx == -1 || infoIdx == -1 || docIdx == -1 {
		t.Fatal("missing expected sections in checklist")
	}
	if docIdx < semanticIdx || docIdx > infoIdx {
		t.Error("'Documentation Completeness' should be in SEMANTIC pass (between SEMANTIC and INFORMATIONAL)")
	}
}

func TestDefault_TestQualityRenamed(t *testing.T) {
	content := Default()
	if !strings.Contains(content, "Test Quality:") {
		t.Error("default checklist should contain renamed 'Test Quality' item in Pass 3")
	}
	if strings.Contains(content, "Test Gaps:") {
		t.Error("default checklist should no longer contain 'Test Gaps' (renamed to 'Test Quality')")
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
