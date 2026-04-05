package github

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// IssueCandidate represents a single item that may become a GitHub issue.
// Callers translate their domain objects (proposals, finding groups, ...)
// into this shape before handing off to RunInteractiveIssueCreation.
type IssueCandidate struct {
	Title   string // issue title and basis for the duplicate search
	Preview string // pre-rendered terminal block shown before the y/N prompt
	Body    string // issue body posted to GitHub
}

// IssueCreator creates a GitHub issue and returns the issue URL.
// Injected so callers can substitute a fake in tests.
type IssueCreator func(owner, name, title, body string) (string, error)

// DuplicateSearcher searches for existing issues whose title matches the query.
// Injected so callers can substitute a fake in tests.
type DuplicateSearcher func(owner, name, query string) ([]string, error)

// RunInteractiveIssueCreation walks through candidates, shows each preview,
// performs a duplicate check and asks the user whether to create an issue.
// It returns when the user quits, when all candidates have been processed,
// or when reading from in fails.
//
// kindLabel is used in user-facing prompts ("Create issue for this <kindLabel>").
func RunInteractiveIssueCreation(
	w io.Writer,
	in io.Reader,
	candidates []IssueCandidate,
	owner, name, kindLabel string,
	create IssueCreator,
	search DuplicateSearcher,
) error {
	reader := bufio.NewReader(in)

	created := 0
	skipped := 0

	for i, c := range candidates {
		_, _ = fmt.Fprintf(w, "\n%s\n", strings.Repeat("=", 80))
		_, _ = fmt.Fprintf(w, "%s %d/%d\n", titleCase(kindLabel), i+1, len(candidates))
		_, _ = fmt.Fprintf(w, "%s\n\n", strings.Repeat("=", 80))

		_, _ = fmt.Fprint(w, c.Preview)

		_, _ = fmt.Fprintf(w, "\nCreate issue for this %s in %s/%s? [y/N/q] ", kindLabel, owner, name)
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
			if search != nil {
				matches, err := search(owner, name, c.Title)
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
			}

			_, _ = fmt.Fprintf(w, "Creating issue...\n")
			url, err := create(owner, name, c.Title, c.Body)
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

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
