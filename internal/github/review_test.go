package github

import (
	"encoding/json"
	"testing"
)

func TestReviewComment_JSON(t *testing.T) {
	c := ReviewComment{
		Path: "main.go",
		Line: 42,
		Side: "RIGHT",
		Body: "This is a problem",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded ReviewComment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decoded.Path != "main.go" {
		t.Errorf("Path = %q, want %q", decoded.Path, "main.go")
	}
	if decoded.Line != 42 {
		t.Errorf("Line = %d, want %d", decoded.Line, 42)
	}
	if decoded.Side != "RIGHT" {
		t.Errorf("Side = %q, want %q", decoded.Side, "RIGHT")
	}

	// StartLine/StartSide should be omitted
	if decoded.StartLine != 0 {
		t.Errorf("StartLine should be 0 when omitted, got %d", decoded.StartLine)
	}
}

func TestReviewComment_MultiLine_JSON(t *testing.T) {
	c := ReviewComment{
		Path:      "handler.go",
		Line:      50,
		Side:      "RIGHT",
		StartLine: 45,
		StartSide: "RIGHT",
		Body:      "Multi-line issue",
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := decoded["start_line"]; !ok {
		t.Error("multi-line comment should include start_line")
	}
	if _, ok := decoded["start_side"]; !ok {
		t.Error("multi-line comment should include start_side")
	}
}

func TestReviewRequest_JSON(t *testing.T) {
	req := ReviewRequest{
		Body:     "Review summary",
		Event:    "COMMENT",
		CommitID: "abc123",
		Comments: []ReviewComment{
			{Path: "a.go", Line: 10, Side: "RIGHT", Body: "Fix this"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded ReviewRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if decoded.Event != "COMMENT" {
		t.Errorf("Event = %q, want %q", decoded.Event, "COMMENT")
	}
	if decoded.CommitID != "abc123" {
		t.Errorf("CommitID = %q, want %q", decoded.CommitID, "abc123")
	}
	if len(decoded.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(decoded.Comments))
	}
	if decoded.Comments[0].Path != "a.go" {
		t.Errorf("Comment.Path = %q, want %q", decoded.Comments[0].Path, "a.go")
	}
}
