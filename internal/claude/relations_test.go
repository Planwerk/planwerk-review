package claude

import (
	"strings"
	"testing"

	"github.com/planwerk/planwerk-review/internal/github"
)

func TestRenderIssueRelations_StandaloneRendersNothing(t *testing.T) {
	var sb strings.Builder
	renderIssueRelations(&sb, nil, nil, nil)
	if sb.Len() != 0 {
		t.Fatalf("renderIssueRelations rendered %q for a standalone issue, want nothing", sb.String())
	}
}

func TestRenderIssueRelations_MetaAndSiblings(t *testing.T) {
	var sb strings.Builder
	meta := &github.Issue{Number: 1, Title: "Meta", Body: "Meta body", State: "open"}
	siblings := []github.Issue{
		{Number: 2, Title: "Open sibling", Body: "open body", State: "open"},
		{Number: 3, Title: "Closed sibling", Body: "closed body", State: "closed"},
	}
	renderIssueRelations(&sb, meta, siblings, nil)
	got := sb.String()

	for _, want := range []string{
		"## Meta / Sub-Issue Context",
		"Sub Issue** of a Meta Issue",
		"<meta-issue number=1 state=open>",
		"**Meta Issue #1**: Meta",
		"Meta body",
		"<sibling-sub-issues>",
		"<sibling number=2 state=open>",
		"<sibling number=3 state=closed>",
		"the remaining X is handled by #K",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered section missing %q\n---\n%s", want, got)
		}
	}
}

func TestRenderIssueRelations_NoSiblingsNote(t *testing.T) {
	var sb strings.Builder
	meta := &github.Issue{Number: 1, Title: "Meta", Body: "Meta body", State: "open"}
	renderIssueRelations(&sb, meta, nil, nil)
	if !strings.Contains(sb.String(), "no siblings yet") {
		t.Errorf("expected the no-siblings note for a lone Sub Issue\n%s", sb.String())
	}
}

func TestRenderIssueRelations_ChildrenWhenIssueIsItselfMeta(t *testing.T) {
	var sb strings.Builder
	children := []github.Issue{
		{Number: 2, Title: "Child one", Body: "child body", State: "open"},
	}
	renderIssueRelations(&sb, nil, nil, children)
	got := sb.String()
	for _, want := range []string{
		"## Meta / Sub-Issue Context",
		"itself a **Meta Issue**",
		"<child-sub-issues>",
		"<sub-issue number=2 state=open>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered children section missing %q\n---\n%s", want, got)
		}
	}
}

func TestRenderIssueRelations_EmptyStateFallsBackToUnknown(t *testing.T) {
	var sb strings.Builder
	meta := &github.Issue{Number: 1, Title: "Meta", Body: "", State: ""}
	renderIssueRelations(&sb, meta, nil, nil)
	if !strings.Contains(sb.String(), "<meta-issue number=1 state=unknown>") {
		t.Errorf("expected state=unknown fallback\n%s", sb.String())
	}
}
