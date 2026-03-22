package propose

import "testing"

func TestCategorizeByPriority(t *testing.T) {
	proposals := []Proposal{
		{ID: "H-001", Priority: "HIGH", Title: "High 1"},
		{ID: "M-001", Priority: "MEDIUM", Title: "Medium 1"},
		{ID: "L-001", Priority: "LOW", Title: "Low 1"},
		{ID: "H-002", Priority: "HIGH", Title: "High 2"},
		{ID: "L-002", Priority: "LOW", Title: "Low 2"},
	}

	cp := CategorizeByPriority(proposals)

	if len(cp.High) != 2 {
		t.Errorf("expected 2 HIGH proposals, got %d", len(cp.High))
	}
	if len(cp.Medium) != 1 {
		t.Errorf("expected 1 MEDIUM proposal, got %d", len(cp.Medium))
	}
	if len(cp.Low) != 2 {
		t.Errorf("expected 2 LOW proposals, got %d", len(cp.Low))
	}
}

func TestCategorizeByPriority_Empty(t *testing.T) {
	cp := CategorizeByPriority(nil)
	if len(cp.High) != 0 || len(cp.Medium) != 0 || len(cp.Low) != 0 {
		t.Error("expected all empty categories for nil input")
	}
}

func TestCategorizeByPriority_UnknownPriority(t *testing.T) {
	proposals := []Proposal{
		{ID: "X-001", Priority: "UNKNOWN", Title: "Unknown priority"},
	}

	cp := CategorizeByPriority(proposals)
	if len(cp.Low) != 1 {
		t.Errorf("expected unknown priority to fall into LOW, got %d", len(cp.Low))
	}
}
