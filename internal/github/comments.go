package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	// maxCommentLen is the GitHub API limit for issue/PR comment bodies.
	maxCommentLen = 65536
	// commentSignature is appended to comments so we can detect duplicates.
	commentSignature = "<!-- planwerk-review -->"
)

// truncationNotice is appended when a comment body is cut to fit within
// maxCommentLen. The %d is filled in with maxCommentLen at format time.
var truncationNotice = fmt.Sprintf(
	"\n\n---\n*Review truncated due to GitHub comment size limit (%d characters).*\n",
	maxCommentLen,
)

// PostPRComment posts a comment on a GitHub pull request via the gh CLI.
// It detects and replaces any previous planwerk-review comment on the same PR.
// Bodies exceeding maxCommentLen are truncated.
func PostPRComment(owner, repo string, number int, body string) (string, error) {
	fullName := fmt.Sprintf("%s/%s", owner, repo)
	body = truncateComment(body + "\n" + commentSignature)

	// Check for an existing planwerk-review comment to update.
	existingID, err := findExistingComment(fullName, number)
	if err != nil {
		return "", fmt.Errorf("checking existing comments: %w", err)
	}

	if existingID != "" {
		return editComment(fullName, existingID, body)
	}

	args := []string{"pr", "comment", strconv.Itoa(number),
		"--repo", fullName,
		"--body", body,
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr comment: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// truncateComment ensures body does not exceed GitHub's comment size limit.
// It avoids splitting multi-byte UTF-8 characters at the cut point.
func truncateComment(body string) string {
	if len(body) <= maxCommentLen {
		return body
	}
	// Reserve space for truncation notice + signature
	suffix := truncationNotice + commentSignature
	cutPoint := maxCommentLen - len(suffix)
	// Avoid splitting a multi-byte UTF-8 character: walk back to a valid boundary.
	for cutPoint > 0 && !utf8.RuneStart(body[cutPoint]) {
		cutPoint--
	}
	return body[:cutPoint] + suffix
}

type ghComment struct {
	ID   string `json:"id"`
	Body string `json:"body"`
}

// findExistingComment searches for a prior planwerk-review comment on the PR.
func findExistingComment(repo string, number int) (string, error) {
	args := []string{"pr", "view", strconv.Itoa(number),
		"--repo", repo,
		"--json", "comments",
		"--jq", `.comments[] | select(.body | contains("` + commentSignature + `")) | {id, body}`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr view comments: %s: %w", strings.TrimSpace(string(out)), err)
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return "", nil
	}

	// Take the last matching comment (most recent).
	lines := strings.Split(output, "\n")
	var last ghComment
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c ghComment
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		last = c
	}

	return last.ID, nil
}

// editComment updates an existing comment by its node ID.
// Uses GraphQL variables to avoid injection via commentID or body.
func editComment(repo, commentID, body string) (string, error) {
	query := `mutation($id: ID!, $body: String!) { updateIssueComment(input: {id: $id, body: $body}) { issueComment { url } } }`

	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "api", "graphql",
		"-f", "query="+query,
		"-f", "id="+commentID,
		"-f", "body="+body,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api graphql (update comment): %s: %w", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}
