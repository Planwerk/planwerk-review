package propose

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderProposals_Markdown(t *testing.T) {
	result := &ProposalResult{
		RepositoryOverview: "Test repo overview.",
		Proposals: []Proposal{
			{ID: "H-001", Priority: "HIGH", Category: "feature", Title: "Test feature", Description: "Desc", Motivation: "Mot", Scope: "Small"},
		},
	}

	var buf bytes.Buffer
	renderer := NewRenderer(&buf)
	renderer.RenderMarkdown(*result, "owner/repo", "v1.0.0")

	out := buf.String()
	if !strings.Contains(out, "# Feature Proposals: owner/repo") {
		t.Error("missing header")
	}
	if !strings.Contains(out, "Test repo overview.") {
		t.Error("missing overview")
	}
	if !strings.Contains(out, "H-001: Test feature") {
		t.Error("missing proposal")
	}
	if !strings.Contains(out, "v1.0.0") {
		t.Error("missing version")
	}
}

func TestRenderProposals_JSON(t *testing.T) {
	result := &ProposalResult{
		RepositoryOverview: "Test repo.",
		Proposals: []Proposal{
			{ID: "H-001", Priority: "HIGH", Title: "Test"},
		},
	}

	var buf bytes.Buffer
	renderer := NewRenderer(&buf)
	if err := renderer.RenderJSON(*result); err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var decoded ProposalResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.RepositoryOverview != "Test repo." {
		t.Errorf("overview = %q, want %q", decoded.RepositoryOverview, "Test repo.")
	}
	if len(decoded.Proposals) != 1 {
		t.Errorf("proposals count = %d, want 1", len(decoded.Proposals))
	}
}

func TestRenderProposals_Issues(t *testing.T) {
	result := &ProposalResult{
		Proposals: []Proposal{
			{
				ID:                 "H-001",
				Priority:           "HIGH",
				Category:           "security",
				Title:              "Fix auth",
				Description:        "Fix the auth flow.",
				Motivation:         "Security risk.",
				Scope:              "Medium",
				AffectedAreas:      []string{"internal/auth"},
				AcceptanceCriteria: []string{"Auth flow is secure"},
			},
		},
	}

	var buf bytes.Buffer
	renderer := NewRenderer(&buf)
	renderer.RenderIssues(*result, "owner/repo")

	out := buf.String()
	if !strings.Contains(out, "`[HIGH]` Fix auth") {
		t.Error("missing issue header")
	}
	if !strings.Contains(out, "security, high, scope:medium") {
		t.Error("missing labels")
	}
	if !strings.Contains(out, "`internal/auth`") {
		t.Error("missing affected areas")
	}
	if !strings.Contains(out, "- [ ] Auth flow is secure") {
		t.Error("missing acceptance criteria")
	}
}
