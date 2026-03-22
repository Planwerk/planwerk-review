package github

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SearchIssues searches for existing issues in the given repo whose title
// matches the query string. It returns matching issue titles and URLs.
func SearchIssues(owner, name, query string) ([]string, error) {
	repo := fmt.Sprintf("%s/%s", owner, name)
	// Quote the query and restrict search to title field for accurate matching
	search := fmt.Sprintf(`"%s" in:title`, query)
	args := []string{"issue", "list",
		"--repo", repo,
		"--search", search,
		"--state", "all",
		"--json", "title,url",
		"--template", `{{range .}}{{.title}}{{"\t"}}{{.url}}{{"\n"}}{{end}}`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh issue list: %s: %w", strings.TrimSpace(string(out)), err)
	}

	output := strings.TrimSpace(string(out))
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}

// CreateIssue creates a GitHub issue in the given repo via the gh CLI.
// It returns the URL of the created issue.
func CreateIssue(owner, name, title, body string) (string, error) {
	repo := fmt.Sprintf("%s/%s", owner, name)
	args := []string{"issue", "create",
		"--repo", repo,
		"--title", title,
		"--body", body,
	}

	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh issue create: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return strings.TrimSpace(string(out)), nil
}
