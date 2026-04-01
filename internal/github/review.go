package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ReviewComment represents a single inline comment in a PR review.
type ReviewComment struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`                 // absolute line number on RIGHT side
	Side      string `json:"side"`                 // "RIGHT" for new-file lines
	StartLine int    `json:"start_line,omitempty"`  // for multi-line comments
	StartSide string `json:"start_side,omitempty"`  // "RIGHT" when StartLine is set
	Body      string `json:"body"`
}

// ReviewRequest represents the payload for creating a PR review.
type ReviewRequest struct {
	Body     string          `json:"body"`
	Event    string          `json:"event"`     // "COMMENT"
	CommitID string          `json:"commit_id"`
	Comments []ReviewComment `json:"comments"`
}

const reviewSignature = "<!-- planwerk-review-inline -->"

// SubmitPRReview creates a Pull Request Review with inline comments
// using the GitHub REST API via gh.
func SubmitPRReview(owner, repo string, number int, commitSHA string, body string, comments []ReviewComment) (string, error) {
	fullBody := body + "\n" + reviewSignature

	req := ReviewRequest{
		Body:     truncateComment(fullBody),
		Event:    "COMMENT",
		CommitID: commitSHA,
		Comments: comments,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling review request: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "api", endpoint,
		"--method", "POST",
		"--input", "-",
		"-H", "Accept: application/vnd.github.v3+json",
	)
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api create review: %s: %w", strings.TrimSpace(string(out)), err)
	}

	var resp struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return strings.TrimSpace(string(out)), nil
	}
	return resp.HTMLURL, nil
}
