package propose

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/planwerk/planwerk-review/internal/github"
)

// RunInteractiveIssueCreation displays all proposals in a summary table,
// then walks through each one, showing full content and asking whether to
// create a GitHub issue.
func RunInteractiveIssueCreation(w io.Writer, result *ProposalResult, owner, name string) error {
	reader := bufio.NewReader(os.Stdin)
	cp := CategorizeByPriority(result.Proposals)

	// 1. Show summary table
	printSummaryTable(w, cp)

	// 2. Walk through each proposal by priority
	allProposals := make([]Proposal, 0, len(result.Proposals))
	allProposals = append(allProposals, cp.High...)
	allProposals = append(allProposals, cp.Medium...)
	allProposals = append(allProposals, cp.Low...)

	created := 0
	skipped := 0

	for i, p := range allProposals {
		_, _ = fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 80))
		_, _ = fmt.Fprintf(w, "Proposal %d/%d\n", i+1, len(allProposals))
		_, _ = fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 80))

		// Show full proposal content
		printProposalFull(w, p)

		// Ask user
		_, _ = fmt.Fprintf(w, "\nCreate issue for this proposal in %s/%s? [y/N/q] ", owner, name)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "q", "quit":
			_, _ = fmt.Fprintf(w, "\nAborted. Created %d issue(s), skipped %d.\n", created, skipped)
			return nil
		case "y", "yes":
			title := p.Title
			body := buildIssueBody(p)

			// Check for existing issues with a similar title
			matches, err := github.SearchIssues(owner, name, title)
			if err != nil {
				_, _ = fmt.Fprintf(w, "Warning: could not check for duplicates: %v\n", err)
			} else if len(matches) > 0 {
				_, _ = fmt.Fprintf(w, "\nPossible duplicate issue(s) found:\n")
				for _, m := range matches {
					_, _ = fmt.Fprintf(w, "  - %s\n", m)
				}
				_, _ = fmt.Fprintf(w, "\nStill create this issue? [y/N] ")
				confirm, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("reading input: %w", err)
				}
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm != "y" && confirm != "yes" {
					_, _ = fmt.Fprintf(w, "Skipped (duplicate).\n")
					skipped++
					continue
				}
			}

			_, _ = fmt.Fprintf(w, "Creating issue...\n")
			url, err := github.CreateIssue(owner, name, title, body)
			if err != nil {
				_, _ = fmt.Fprintf(w, "Error creating issue: %v\n", err)
				skipped++
				continue
			}
			_, _ = fmt.Fprintf(w, "Created: %s\n", url)
			created++
		default:
			_, _ = fmt.Fprintf(w, "Skipped.\n")
			skipped++
		}
	}

	_, _ = fmt.Fprintf(w, "\nDone. Created %d issue(s), skipped %d.\n", created, skipped)
	return nil
}

func printSummaryTable(w io.Writer, cp CategorizedProposals) {
	total := len(cp.High) + len(cp.Medium) + len(cp.Low)
	_, _ = fmt.Fprintf(w, "\n## Proposals Overview (%d total)\n\n", total)
	_, _ = fmt.Fprintf(w, "| # | ID     | Priority | Category       | Scope  | Title                                                      |\n")
	_, _ = fmt.Fprintf(w, "|---|--------|----------|----------------|--------|------------------------------------------------------------|\n")

	n := 0
	for _, p := range cp.High {
		n++
		_, _ = fmt.Fprintf(w, "| %d | %s | %-8s | %-14s | %-6s | %-58s |\n", n, p.ID, p.Priority, p.Category, p.Scope, truncate(p.Title, 58))
	}
	for _, p := range cp.Medium {
		n++
		_, _ = fmt.Fprintf(w, "| %d | %s | %-8s | %-14s | %-6s | %-58s |\n", n, p.ID, p.Priority, p.Category, p.Scope, truncate(p.Title, 58))
	}
	for _, p := range cp.Low {
		n++
		_, _ = fmt.Fprintf(w, "| %d | %s | %-8s | %-14s | %-6s | %-58s |\n", n, p.ID, p.Priority, p.Category, p.Scope, truncate(p.Title, 58))
	}
	_, _ = fmt.Fprintln(w)
}

func printProposalFull(w io.Writer, p Proposal) {
	_, _ = fmt.Fprintf(w, "## %s Priority\n\n", p.Priority)
	_, _ = fmt.Fprintf(w, "### %s: %s\n\n", p.ID, p.Title)
	_, _ = fmt.Fprintf(w, "**Category**: %s | **Scope**: %s\n\n", p.Category, p.Scope)
	_, _ = fmt.Fprintf(w, "**Description**: %s\n\n", p.Description)
	_, _ = fmt.Fprintf(w, "**Motivation**: %s\n\n", p.Motivation)

	if len(p.AffectedAreas) > 0 {
		_, _ = fmt.Fprint(w, "**Affected Areas**:\n")
		for _, area := range p.AffectedAreas {
			_, _ = fmt.Fprintf(w, "- `%s`\n", area)
		}
		_, _ = fmt.Fprintln(w)
	}

	if len(p.AcceptanceCriteria) > 0 {
		_, _ = fmt.Fprint(w, "**Acceptance Criteria**:\n")
		for _, ac := range p.AcceptanceCriteria {
			_, _ = fmt.Fprintf(w, "- [ ] %s\n", ac)
		}
		_, _ = fmt.Fprintln(w)
	}
}

func buildIssueBody(p Proposal) string {
	var b strings.Builder

	fmt.Fprintf(&b, "**Category**: %s | **Scope**: %s | **Priority**: %s\n\n", p.Category, p.Scope, p.Priority)
	fmt.Fprintf(&b, "## Description\n\n%s\n\n", p.Description)
	fmt.Fprintf(&b, "## Motivation\n\n%s\n\n", p.Motivation)

	if len(p.AffectedAreas) > 0 {
		fmt.Fprint(&b, "## Affected Areas\n\n")
		for _, area := range p.AffectedAreas {
			fmt.Fprintf(&b, "- `%s`\n", area)
		}
		fmt.Fprintln(&b)
	}

	if len(p.AcceptanceCriteria) > 0 {
		fmt.Fprint(&b, "## Acceptance Criteria\n\n")
		for _, ac := range p.AcceptanceCriteria {
			fmt.Fprintf(&b, "- [ ] %s\n", ac)
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprint(&b, "---\n\n_Generated by [planwerk-review](https://github.com/planwerk/planwerk-review) with Claude CLI_\n")

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
