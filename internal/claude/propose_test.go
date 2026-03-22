package claude

import (
	"testing"

	"github.com/planwerk/planwerk-review/internal/propose"
)

func TestAssignProposalIDs(t *testing.T) {
	result := &propose.ProposalResult{
		Proposals: []propose.Proposal{
			{Priority: "high"},
			{Priority: "medium"},
			{Priority: "high"},
			{Priority: "low"},
			{Priority: "medium"},
		},
	}

	assignProposalIDs(result)

	expected := []struct {
		id       string
		priority string
	}{
		{"H-001", "HIGH"},
		{"M-001", "MEDIUM"},
		{"H-002", "HIGH"},
		{"L-001", "LOW"},
		{"M-002", "MEDIUM"},
	}

	for i, exp := range expected {
		p := result.Proposals[i]
		if p.ID != exp.id {
			t.Errorf("proposal[%d].ID = %q, want %q", i, p.ID, exp.id)
		}
		if p.Priority != exp.priority {
			t.Errorf("proposal[%d].Priority = %q, want %q", i, p.Priority, exp.priority)
		}
	}
}

func TestAssignProposalIDs_UnknownPriority(t *testing.T) {
	result := &propose.ProposalResult{
		Proposals: []propose.Proposal{
			{Priority: "CRITICAL"},
		},
	}

	assignProposalIDs(result)

	if result.Proposals[0].ID != "X-001" {
		t.Errorf("unknown priority should get X prefix, got %q", result.Proposals[0].ID)
	}
}

func TestAssignProposalIDs_Empty(t *testing.T) {
	result := &propose.ProposalResult{}
	assignProposalIDs(result)
	if len(result.Proposals) != 0 {
		t.Error("expected no proposals")
	}
}
