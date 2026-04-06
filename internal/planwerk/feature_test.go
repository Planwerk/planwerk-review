package planwerk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFeature_MatchesByTitle(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "features", "CC-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field",
		"requirements": [{"id": "REQ-001", "description": "test", "priority": "SHALL"}]
	}`)

	f, err := DetectFeature(dir, "feat(CC-0042): Add resources field", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected feature to be detected")
	}
	if f.FeatureID != "CC-0042" {
		t.Errorf("feature_id = %q, want %q", f.FeatureID, "CC-0042")
	}
}

func TestDetectFeature_MatchesByBranch(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "features", "CC-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field"
	}`)

	f, err := DetectFeature(dir, "Some PR title", "", "feature/CC-0042")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected feature to be detected by branch name")
	}
}

func TestDetectFeature_MatchesByBody(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "features", "CC-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field"
	}`)

	f, err := DetectFeature(dir, "Some title", "Implements CC-0042 feature", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected feature to be detected by PR body")
	}
}

func TestDetectFeature_CompletedDir(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "completed", "CC-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field"
	}`)

	f, err := DetectFeature(dir, "feat(CC-0042): done", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected feature to be detected in completed dir")
	}
}

func TestDetectFeature_NoMatch(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "features", "CC-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field"
	}`)

	f, err := DetectFeature(dir, "Unrelated PR", "No feature ref", "bugfix/something")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Error("expected no feature to be detected")
	}
}

func TestDetectFeature_NoPlanwerkDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	f, err := DetectFeature(dir, "feat(CC-0042): test", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Error("expected nil when .planwerk directory does not exist")
	}
}

func TestDetectFeature_CaseInsensitive(t *testing.T) {
	t.Parallel()

	dir := setupFeatureDir(t, "features", "cc-0042-test.json", `{
		"feature_id": "CC-0042",
		"title": "Add resources field"
	}`)

	f, err := DetectFeature(dir, "feat(cc-0042): lowercase ref", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected case-insensitive match")
	}
}

func TestFormatForPrompt_ContainsAllSections(t *testing.T) {
	t.Parallel()

	f := &Feature{
		FeatureID: "CC-0042",
		Title:     "Add resources field",
		Stories: []Story{
			{Title: "Story 1", Role: "operator", Want: "resources", SoThat: "HPA works", Criteria: []string{"crit1", "crit2"}},
		},
		Requirements: []Requirement{
			{
				ID:          "REQ-001",
				Description: "Inject defaults",
				Priority:    "SHALL",
				Scenarios:   []Scenario{{Name: "nil defaults", When: "nil", Then: "defaults set"}},
			},
		},
		TestSpecifications: []TestSpecification{
			{TestFile: "test.go", TestFunction: "TestFoo", Expected: "bar", RequirementID: "REQ-001"},
		},
		Tasks: []Task{
			{ID: "1.1", Title: "Add field", Status: "done", Requirements: []string{"REQ-001"}},
		},
	}

	output := f.FormatForPrompt()

	for _, want := range []string{
		"CC-0042",
		"Add resources field",
		"Story 1",
		"operator",
		"crit1",
		"REQ-001",
		"SHALL",
		"Inject defaults",
		"nil defaults",
		"TestFoo",
		"test.go",
		"1.1",
		"Add field",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("FormatForPrompt output missing %q", want)
		}
	}
}

func TestFormatForPrompt_SkipsDiscoveredTests(t *testing.T) {
	t.Parallel()

	f := &Feature{
		FeatureID: "CC-0042",
		Title:     "Test",
		TestSpecifications: []TestSpecification{
			{TestFile: "test.go", TestFunction: "TestPlanned", Expected: "works", RequirementID: "REQ-001"},
			{TestFile: "test.go", TestFunction: "TestDiscovered", Expected: "Discovered during implementation"},
		},
	}

	output := f.FormatForPrompt()

	if !strings.Contains(output, "TestPlanned") {
		t.Error("should include planned test")
	}
	if strings.Contains(output, "TestDiscovered") {
		t.Error("should skip discovered-during-implementation test")
	}
}

func setupFeatureDir(t *testing.T, subdir, filename, content string) string {
	t.Helper()
	dir := t.TempDir()
	featureDir := filepath.Join(dir, ".planwerk", subdir)
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
