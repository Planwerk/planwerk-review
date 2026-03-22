package review

import (
	"bytes"
	"strings"
	"testing"

	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/report"
)

func TestRenderResult_PostReview(t *testing.T) {
	var postedBody string
	origFunc := postCommentFunc
	postCommentFunc = func(owner, repo string, number int, body string) (string, error) {
		postedBody = body
		return "https://github.com/test/repo/pull/1#issuecomment-123", nil
	}
	defer func() { postCommentFunc = origFunc }()

	result := &report.ReviewResult{
		Summary:        "Looks good",
		Recommendation: "Merge it",
		Findings: []report.Finding{
			{ID: "F1", Severity: "WARNING", Title: "Test finding", File: "main.go", Problem: "Issue", Action: "Fix it"},
		},
	}
	pr := &github.PR{Owner: "test", Repo: "repo", Number: 1, Title: "Test PR"}
	opts := Options{
		Format:      "markdown",
		MinSeverity: report.SeverityInfo,
		Version:     "test",
		PostReview:  true,
	}

	var out bytes.Buffer
	err := renderResult(&out, result, pr, opts, nil)
	if err != nil {
		t.Fatalf("renderResult returned error: %v", err)
	}

	// Verify output was written to the writer
	if out.Len() == 0 {
		t.Error("expected output to be written to writer")
	}

	// Verify the review was posted
	if postedBody == "" {
		t.Error("expected review to be posted as PR comment")
	}

	// Verify posted body matches stdout output
	if postedBody != out.String() {
		t.Error("posted body should match stdout output")
	}
}

func TestRenderResult_NoPost(t *testing.T) {
	origFunc := postCommentFunc
	postCommentFunc = func(owner, repo string, number int, body string) (string, error) {
		t.Fatal("PostPRComment should not be called when PostReview is false")
		return "", nil
	}
	defer func() { postCommentFunc = origFunc }()

	result := &report.ReviewResult{
		Summary: "Looks good",
	}
	pr := &github.PR{Owner: "test", Repo: "repo", Number: 1, Title: "Test PR"}
	opts := Options{
		Format:      "markdown",
		MinSeverity: report.SeverityInfo,
		Version:     "test",
		PostReview:  false,
	}

	var out bytes.Buffer
	err := renderResult(&out, result, pr, opts, nil)
	if err != nil {
		t.Fatalf("renderResult returned error: %v", err)
	}

	if out.Len() == 0 {
		t.Error("expected output to be written to writer")
	}
}

func TestRenderResult_PostReviewError(t *testing.T) {
	origFunc := postCommentFunc
	postCommentFunc = func(owner, repo string, number int, body string) (string, error) {
		return "", &mockError{"post failed"}
	}
	defer func() { postCommentFunc = origFunc }()

	result := &report.ReviewResult{Summary: "Test"}
	pr := &github.PR{Owner: "test", Repo: "repo", Number: 1, Title: "Test PR"}
	opts := Options{
		Format:      "markdown",
		MinSeverity: report.SeverityInfo,
		Version:     "test",
		PostReview:  true,
	}

	var out bytes.Buffer
	err := renderResult(&out, result, pr, opts, nil)
	if err == nil {
		t.Fatal("expected error when PostPRComment fails")
	}
	if !strings.Contains(err.Error(), "posting PR comment") {
		t.Errorf("error should mention posting PR comment, got: %v", err)
	}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string { return e.msg }
