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

	f, err := DetectFeature(dir, "feat(CC-0042): Add resources field", "", "", nil)
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

	f, err := DetectFeature(dir, "Some PR title", "", "feature/CC-0042", nil)
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

	f, err := DetectFeature(dir, "Some title", "Implements CC-0042 feature", "", nil)
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

	f, err := DetectFeature(dir, "feat(CC-0042): done", "", "", nil)
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

	f, err := DetectFeature(dir, "Unrelated PR", "No feature ref", "bugfix/something", nil)
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

	f, err := DetectFeature(dir, "feat(CC-0042): test", "", "", nil)
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

	f, err := DetectFeature(dir, "feat(cc-0042): lowercase ref", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected case-insensitive match")
	}
}

// TestDetectFeature_BranchWinsOverBodyCrossRef reproduces the PR #167 scenario:
// two feature files on disk, branch points to CC-0055, but the PR body
// references CC-0050 ("following the pattern established by CC-0050"). The
// branch must win — returning CC-0050 would be wrong.
func TestDetectFeature_BranchWinsOverBodyCrossRef(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFeatureFile(t, dir, "features", "CC-0050-refactor-ci.json", `{
		"feature_id": "CC-0050",
		"title": "Refactor CI workflow into reusable scripts and actions"
	}`)
	writeFeatureFile(t, dir, "features", "CC-0055-refactor-build-images.json", `{
		"feature_id": "CC-0055",
		"title": "Refactor build-images workflow into reusable components"
	}`)

	body := "Refactor build-images.yaml, following the pattern established by CC-0050 for ci.yaml."
	f, err := DetectFeature(dir,
		"feat(CC-0055): Refactor build-images workflow into reusable components",
		body,
		"feature/CC-0055",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil || f.FeatureID != "CC-0055" {
		t.Fatalf("feature_id = %v, want CC-0055", f)
	}
}

// TestDetectFeature_MatchesByChangedFiles covers the case where neither the
// branch nor the title carry a feature ID, but the PR touches files under
// .planwerk/progress/ or .planwerk/reviews/ that carry the ID.
func TestDetectFeature_MatchesByChangedFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFeatureFile(t, dir, "features", "CC-0050-refactor-ci.json", `{
		"feature_id": "CC-0050",
		"title": "Refactor CI"
	}`)
	writeFeatureFile(t, dir, "features", "CC-0055-refactor-build-images.json", `{
		"feature_id": "CC-0055",
		"title": "Refactor build-images"
	}`)

	changed := []string{
		".github/workflows/build-images.yaml",
		".planwerk/progress/CC-0055-refactor-build-images.json",
		".planwerk/reviews/CC-0055-refactor-build-images-review-1.json",
	}
	f, err := DetectFeature(dir, "Some PR", "", "bugfix/thing", changed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil || f.FeatureID != "CC-0055" {
		t.Fatalf("feature_id = %v, want CC-0055", f)
	}
}

// TestDetectFeature_AmbiguousBodyReturnsNil ensures that when only the body
// carries signals and it references multiple feature IDs, detection refuses
// to guess.
func TestDetectFeature_AmbiguousBodyReturnsNil(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFeatureFile(t, dir, "features", "CC-0050-a.json", `{"feature_id": "CC-0050", "title": "A"}`)
	writeFeatureFile(t, dir, "features", "CC-0055-b.json", `{"feature_id": "CC-0055", "title": "B"}`)

	body := "See CC-0050 and CC-0055 for context."
	f, err := DetectFeature(dir, "Some PR", body, "bugfix/thing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Errorf("expected nil on ambiguous body, got %q", f.FeatureID)
	}
}

// TestDetectFeature_BranchIDWithoutFeatureFileFallsThrough ensures that a
// branch carrying a feature ID with no matching feature file on disk does
// not short-circuit detection — later stages still get a chance.
func TestDetectFeature_BranchIDWithoutFeatureFileFallsThrough(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFeatureFile(t, dir, "features", "CC-0055-b.json", `{"feature_id": "CC-0055", "title": "B"}`)

	// Branch references CC-0099 which has no feature file; changed files
	// point at CC-0055 which does.
	changed := []string{".planwerk/progress/CC-0055-b.json"}
	f, err := DetectFeature(dir, "Some PR", "", "feature/CC-0099", changed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil || f.FeatureID != "CC-0055" {
		t.Fatalf("feature_id = %v, want CC-0055", f)
	}
}

func writeFeatureFile(t *testing.T, repoDir, subdir, filename, content string) {
	t.Helper()
	featureDir := filepath.Join(repoDir, ".planwerk", subdir)
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(featureDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
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
