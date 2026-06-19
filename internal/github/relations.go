package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
)

// maxRelatedSubIssues bounds how many sub-issues the relations query pulls for
// the parent (siblings) and the issue itself (children). A Meta Issue with more
// than this many Sub Issues is unrealistic; the cap keeps a single GraphQL page
// sufficient. Truncation past it is logged, never silent.
const maxRelatedSubIssues = 100

// IssueRelations is the Meta/Sub-Issue neighborhood of an issue, used by
// elaborate and plan so they ground a Sub Issue in its larger effort instead of
// in isolation:
//   - Parent is the Meta Issue when the issue is a Sub Issue (nil otherwise).
//   - Siblings are the Meta Issue's other Sub Issues (the issue itself filtered out).
//   - Children are the issue's own Sub Issues, present when the issue is itself a
//     Meta Issue.
type IssueRelations struct {
	Parent   *Issue
	Siblings []Issue
	Children []Issue
}

// relationsQuery is the GraphQL query that fetches an issue's parent (with the
// parent's own sub-issues, i.e. the siblings) and the issue's own sub-issues
// (the children) in a single round trip. Bodies are included so elaborate/plan
// can read the Meta Issue and sibling content, not just their titles. The page
// size is interpolated from maxRelatedSubIssues so the cap has one source.
var relationsQuery = fmt.Sprintf(`query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    issue(number: $number) {
      number
      parent {
        number title body url state
        subIssues(first: %[1]d) { totalCount nodes { number title body url state } }
      }
      subIssues(first: %[1]d) { totalCount nodes { number title body url state } }
    }
  }
}`, maxRelatedSubIssues)

// GetIssueRelations resolves the Meta/Sub-Issue neighborhood of an issue via a
// single gh GraphQL call. Callers treat a returned error as best-effort: a repo
// without sub-issue relationships, a token lacking the scope, or an older GHES
// that does not expose the fields all surface here and should degrade to "no
// relations" rather than abort the elaborate/plan run.
func GetIssueRelations(owner, name string, number int) (*IssueRelations, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "api", "graphql",
		"-F", "owner="+owner,
		"-F", "name="+name,
		"-F", "number="+strconv.Itoa(number),
		"-f", "query="+relationsQuery)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh api graphql sub-issue relations: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return parseIssueRelations(out, owner, name, number)
}

// graphqlIssueNode is the minimal issue projection the relations query returns
// for the parent and for each sub-issue node.
type graphqlIssueNode struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

// graphqlSubIssues is the connection wrapper around a list of sub-issue nodes,
// carrying totalCount so truncation past maxRelatedSubIssues is detectable.
type graphqlSubIssues struct {
	TotalCount int                `json:"totalCount"`
	Nodes      []graphqlIssueNode `json:"nodes"`
}

// graphqlRelationsResponse mirrors the gh api graphql envelope for relationsQuery.
// Parent is a pointer so a missing parent (the issue is not a Sub Issue) decodes
// to nil rather than a zero-valued issue.
type graphqlRelationsResponse struct {
	Data struct {
		Repository struct {
			Issue struct {
				Number int `json:"number"`
				Parent *struct {
					graphqlIssueNode
					SubIssues graphqlSubIssues `json:"subIssues"`
				} `json:"parent"`
				SubIssues graphqlSubIssues `json:"subIssues"`
			} `json:"issue"`
		} `json:"repository"`
	} `json:"data"`
}

// parseIssueRelations decodes the relations GraphQL envelope into IssueRelations.
// It filters the target issue out of the sibling list, stamps Owner/Name onto
// every returned issue, and normalizes the GraphQL state enum (OPEN/CLOSED) to
// the lowercase form the rest of the github package uses (open/closed). Sibling
// or child lists truncated past maxRelatedSubIssues are logged, never dropped
// silently. Kept separate from GetIssueRelations so the decode is unit-testable
// without invoking gh.
func parseIssueRelations(out []byte, owner, name string, number int) (*IssueRelations, error) {
	var resp graphqlRelationsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing gh api graphql sub-issue relations: %w", err)
	}

	issue := resp.Data.Repository.Issue
	rel := &IssueRelations{}

	if p := issue.Parent; p != nil {
		parent := toIssue(p.graphqlIssueNode, owner, name)
		rel.Parent = &parent
		rel.Siblings = nodesToIssues(p.SubIssues.Nodes, owner, name, number)
		warnIfTruncated(p.SubIssues, "sibling", number)
	}

	rel.Children = nodesToIssues(issue.SubIssues.Nodes, owner, name, 0)
	warnIfTruncated(issue.SubIssues, "child", number)

	return rel, nil
}

// nodesToIssues converts sub-issue nodes to github.Issue values, dropping the
// node whose number equals exclude (used to filter the target issue out of its
// own sibling list; pass 0 to keep every node, since issue numbers start at 1).
func nodesToIssues(nodes []graphqlIssueNode, owner, name string, exclude int) []Issue {
	var issues []Issue
	for _, n := range nodes {
		if n.Number == exclude {
			continue
		}
		issues = append(issues, toIssue(n, owner, name))
	}
	return issues
}

// toIssue maps a GraphQL node onto the package's Issue type, stamping the repo
// coordinates and lowercasing the state enum to match GetIssue's convention.
func toIssue(n graphqlIssueNode, owner, name string) Issue {
	return Issue{
		Owner:  owner,
		Name:   name,
		Number: n.Number,
		Title:  n.Title,
		Body:   n.Body,
		URL:    n.URL,
		State:  strings.ToLower(n.State),
	}
}

// warnIfTruncated logs when a sub-issue connection returned fewer nodes than its
// totalCount, so a Meta Issue with more than maxRelatedSubIssues Sub Issues does
// not silently drop the overflow from the planning context.
func warnIfTruncated(conn graphqlSubIssues, kind string, number int) {
	if conn.TotalCount > len(conn.Nodes) {
		slog.Warn("sub-issue relations truncated; some are omitted from the planning context",
			"issue", number, "kind", kind, "total", conn.TotalCount, "fetched", len(conn.Nodes), "cap", maxRelatedSubIssues)
	}
}
