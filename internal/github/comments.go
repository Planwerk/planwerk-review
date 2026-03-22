package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	// maxCommentLen is the GitHub API limit for issue/PR comment bodies.
	maxCommentLen = 65536
	// commentSignature is appended to comments so we can detect duplicates.
	commentSignature = "<!-- planwerk-review -->"
	truncationNotice = "\n\n---\n*Review truncated due to GitHub comment size limit.*\n"
)

// PostPRComment posts a comment on a GitHub pull request via the gh CLI.
// It detects and replaces any previous planwerk-review comment on the same PR.
// Bodies exceeding GitHub's 65 536-character limit are truncated.
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

	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh pr comment: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// truncateComment ensures body does not exceed GitHub's comment size limit.
func truncateComment(body string) string {
	if len(body) <= maxCommentLen {
		return body
	}
	// Reserve space for truncation notice + signature
	suffix := truncationNotice + commentSignature
	return body[:maxCommentLen-len(suffix)] + suffix
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

	cmd := exec.Command("gh", args...)
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
func editComment(repo, commentID, body string) (string, error) {
	// gh api to update the comment via GraphQL node ID
	mutation := fmt.Sprintf(`mutation { updateIssueComment(input: {id: "%s", body: %s}) { issueComment { url } } }`,
		commentID, jsonString(body))

	cmd := exec.Command("gh", "api", "graphql", "-f", "query="+mutation)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api graphql (update comment): %s: %w", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}

// jsonString returns s as a JSON-encoded string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
