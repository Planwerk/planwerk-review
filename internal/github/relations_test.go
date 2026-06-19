package github

import "testing"

// relOwner and relRepo are the repo coordinates the relations tests stamp onto
// parsed issues; kept as constants so the literals are not repeated (goconst).
const (
	relOwner = "acme"
	relRepo  = "widgets"
)

// parentWithSiblings is a relations envelope where issue #3 is a Sub Issue of
// Meta Issue #1, which has three Sub Issues total (#2, #3, #4). #4 is closed.
const parentWithSiblings = `{"data":{"repository":{"issue":{
  "number":3,
  "parent":{
    "number":1,"title":"Meta","body":"Meta body","url":"https://example.com/1","state":"OPEN",
    "subIssues":{"totalCount":3,"nodes":[
      {"number":2,"title":"Sub two","body":"two body","url":"https://example.com/2","state":"OPEN"},
      {"number":3,"title":"Sub three","body":"three body","url":"https://example.com/3","state":"OPEN"},
      {"number":4,"title":"Sub four","body":"four body","url":"https://example.com/4","state":"CLOSED"}
    ]}
  },
  "subIssues":{"totalCount":0,"nodes":[]}
}}}}`

func TestParseIssueRelations_ParentAndSiblings(t *testing.T) {
	rel, err := parseIssueRelations([]byte(parentWithSiblings), relOwner, relRepo, 3)
	if err != nil {
		t.Fatalf("parseIssueRelations() error = %v", err)
	}

	if rel.Parent == nil {
		t.Fatal("Parent is nil, want Meta Issue #1")
	}
	if rel.Parent.Number != 1 || rel.Parent.Title != "Meta" || rel.Parent.Body != "Meta body" {
		t.Errorf("Parent = %+v, want #1 'Meta'/'Meta body'", rel.Parent)
	}
	if rel.Parent.Owner != relOwner || rel.Parent.Name != relRepo {
		t.Errorf("Parent repo coords = %q/%q, want acme/widgets", rel.Parent.Owner, rel.Parent.Name)
	}
	if rel.Parent.State != "open" {
		t.Errorf("Parent.State = %q, want lowercase %q", rel.Parent.State, "open")
	}

	// The target issue (#3) must be filtered out of its own sibling list.
	if len(rel.Siblings) != 2 {
		t.Fatalf("len(Siblings) = %d, want 2 (#3 filtered out)", len(rel.Siblings))
	}
	for _, s := range rel.Siblings {
		if s.Number == 3 {
			t.Errorf("Siblings still contains the target issue #3: %+v", s)
		}
		if s.Owner != relOwner || s.Name != relRepo {
			t.Errorf("sibling #%d repo coords = %q/%q, want acme/widgets", s.Number, s.Owner, s.Name)
		}
	}
	// The closed sibling is kept, with its state normalized to lowercase.
	var four *Issue
	for i := range rel.Siblings {
		if rel.Siblings[i].Number == 4 {
			four = &rel.Siblings[i]
		}
	}
	if four == nil {
		t.Fatal("closed sibling #4 missing; closed Sub Issues must be kept")
	}
	if four.State != "closed" {
		t.Errorf("sibling #4 State = %q, want lowercase %q", four.State, "closed")
	}
}

func TestParseIssueRelations_NoParent(t *testing.T) {
	// An issue that is neither a Sub Issue nor a Meta Issue: parent null, no children.
	const noRelations = `{"data":{"repository":{"issue":{
      "number":7,"parent":null,"subIssues":{"totalCount":0,"nodes":[]}
    }}}}`
	rel, err := parseIssueRelations([]byte(noRelations), relOwner, relRepo, 7)
	if err != nil {
		t.Fatalf("parseIssueRelations() error = %v", err)
	}
	if rel.Parent != nil {
		t.Errorf("Parent = %+v, want nil for an issue with no parent", rel.Parent)
	}
	if len(rel.Siblings) != 0 || len(rel.Children) != 0 {
		t.Errorf("Siblings=%d Children=%d, want 0/0", len(rel.Siblings), len(rel.Children))
	}
}

func TestParseIssueRelations_Children(t *testing.T) {
	// Issue #1 is itself a Meta Issue with two Sub Issues and no parent.
	const meta = `{"data":{"repository":{"issue":{
      "number":1,"parent":null,
      "subIssues":{"totalCount":2,"nodes":[
        {"number":2,"title":"Sub two","body":"two","url":"https://example.com/2","state":"OPEN"},
        {"number":3,"title":"Sub three","body":"three","url":"https://example.com/3","state":"CLOSED"}
      ]}
    }}}}`
	rel, err := parseIssueRelations([]byte(meta), relOwner, relRepo, 1)
	if err != nil {
		t.Fatalf("parseIssueRelations() error = %v", err)
	}
	if rel.Parent != nil {
		t.Errorf("Parent = %+v, want nil", rel.Parent)
	}
	if len(rel.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(rel.Children))
	}
	if rel.Children[1].State != "closed" {
		t.Errorf("Children[1].State = %q, want lowercase closed", rel.Children[1].State)
	}
}

func TestParseIssueRelations_InvalidJSON(t *testing.T) {
	if _, err := parseIssueRelations([]byte("not json"), relOwner, relRepo, 1); err == nil {
		t.Fatal("parseIssueRelations() error = nil, want a decode error")
	}
}
