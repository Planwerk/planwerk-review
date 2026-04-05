package github

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func newTestCandidate(title string) IssueCandidate {
	return IssueCandidate{
		Title:   title,
		Preview: "preview for " + title + "\n",
		Body:    "body for " + title,
	}
}

// recordingCreator records all invocations and returns a deterministic URL.
type recordingCreator struct {
	calls []struct{ owner, name, title, body string }
	err   error
}

func (r *recordingCreator) fn() IssueCreator {
	return func(owner, name, title, body string) (string, error) {
		r.calls = append(r.calls, struct{ owner, name, title, body string }{owner, name, title, body})
		if r.err != nil {
			return "", r.err
		}
		return "https://example.invalid/" + owner + "/" + name + "/issues/1", nil
	}
}

// recordingSearcher returns fixed matches for every call.
type recordingSearcher struct {
	calls   int
	matches []string
	err     error
}

func (r *recordingSearcher) fn() DuplicateSearcher {
	return func(owner, name, query string) ([]string, error) {
		r.calls++
		if r.err != nil {
			return nil, r.err
		}
		return r.matches, nil
	}
}

func TestRunInteractiveIssueCreation_HappyPath(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A")}
	creator := &recordingCreator{}
	searcher := &recordingSearcher{}

	var buf bytes.Buffer
	in := strings.NewReader("y\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "acme", "widget", "finding", creator.fn(), searcher.fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creator.calls) != 1 {
		t.Fatalf("creator called %d times, want 1", len(creator.calls))
	}
	if creator.calls[0].title != "A" {
		t.Errorf("title = %q, want %q", creator.calls[0].title, "A")
	}
	out := buf.String()
	if !strings.Contains(out, "Created: https://example.invalid/acme/widget/issues/1") {
		t.Errorf("missing created URL in output: %s", out)
	}
	if !strings.Contains(out, "Done. Created 1 issue(s), skipped 0.") {
		t.Errorf("missing final summary: %s", out)
	}
}

func TestRunInteractiveIssueCreation_SkipOnNo(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A")}
	creator := &recordingCreator{}

	var buf bytes.Buffer
	in := strings.NewReader("n\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "o", "r", "finding", creator.fn(), (&recordingSearcher{}).fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creator.calls) != 0 {
		t.Errorf("creator should not be called, got %d calls", len(creator.calls))
	}
	if !strings.Contains(buf.String(), "Skipped.") {
		t.Errorf("missing 'Skipped.' in output: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "Done. Created 0 issue(s), skipped 1.") {
		t.Errorf("missing final summary: %s", buf.String())
	}
}

func TestRunInteractiveIssueCreation_QuitMidway(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A"), newTestCandidate("B"), newTestCandidate("C")}
	creator := &recordingCreator{}

	var buf bytes.Buffer
	in := strings.NewReader("y\nq\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "o", "r", "finding", creator.fn(), (&recordingSearcher{}).fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creator.calls) != 1 {
		t.Errorf("creator called %d times, want 1", len(creator.calls))
	}
	if !strings.Contains(buf.String(), "Aborted. Created 1 issue(s), skipped 0.") {
		t.Errorf("missing abort summary: %s", buf.String())
	}
	// Third candidate should never have been shown
	if strings.Contains(buf.String(), "3/3") {
		t.Errorf("third candidate should not have been shown: %s", buf.String())
	}
}

func TestRunInteractiveIssueCreation_DuplicateConfirmed(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A")}
	creator := &recordingCreator{}
	searcher := &recordingSearcher{matches: []string{"existing\thttps://example.invalid/issues/99"}}

	var buf bytes.Buffer
	in := strings.NewReader("y\ny\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "o", "r", "finding", creator.fn(), searcher.fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creator.calls) != 1 {
		t.Errorf("creator should have been called after confirm, got %d", len(creator.calls))
	}
	if !strings.Contains(buf.String(), "Possible duplicate issue(s) found") {
		t.Errorf("missing duplicate warning: %s", buf.String())
	}
}

func TestRunInteractiveIssueCreation_DuplicateDeclined(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A")}
	creator := &recordingCreator{}
	searcher := &recordingSearcher{matches: []string{"existing\thttps://example.invalid/issues/99"}}

	var buf bytes.Buffer
	in := strings.NewReader("y\nn\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "o", "r", "finding", creator.fn(), searcher.fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creator.calls) != 0 {
		t.Errorf("creator must not be called when duplicate declined, got %d", len(creator.calls))
	}
	if !strings.Contains(buf.String(), "Skipped (duplicate).") {
		t.Errorf("missing duplicate-skip message: %s", buf.String())
	}
}

func TestRunInteractiveIssueCreation_CreatorError(t *testing.T) {
	candidates := []IssueCandidate{newTestCandidate("A"), newTestCandidate("B")}
	creator := &recordingCreator{err: errors.New("boom")}

	var buf bytes.Buffer
	in := strings.NewReader("y\nn\n")

	err := RunInteractiveIssueCreation(&buf, in, candidates, "o", "r", "finding", creator.fn(), (&recordingSearcher{}).fn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Error creating issue: boom") {
		t.Errorf("missing error message: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "Done. Created 0 issue(s), skipped 2.") {
		t.Errorf("error-path should count as skip: %s", buf.String())
	}
}
