package github

import (
	"strings"
	"testing"
)

func TestTruncateComment_Short(t *testing.T) {
	body := "short comment"
	got := truncateComment(body)
	if got != body {
		t.Errorf("truncateComment should not modify short body, got len=%d", len(got))
	}
}

func TestTruncateComment_ExactLimit(t *testing.T) {
	body := strings.Repeat("a", maxCommentLen)
	got := truncateComment(body)
	if got != body {
		t.Errorf("truncateComment should not modify body at exact limit, got len=%d", len(got))
	}
}

func TestTruncateComment_Oversize(t *testing.T) {
	body := strings.Repeat("x", maxCommentLen+1000)
	got := truncateComment(body)

	if len(got) > maxCommentLen {
		t.Errorf("truncated body len=%d exceeds max=%d", len(got), maxCommentLen)
	}
	if !strings.Contains(got, "truncated") {
		t.Error("truncated body should contain truncation notice")
	}
	if !strings.Contains(got, commentSignature) {
		t.Error("truncated body should preserve comment signature")
	}
}

func TestTruncateComment_PreservesSignature(t *testing.T) {
	body := strings.Repeat("y", maxCommentLen+500)
	got := truncateComment(body)

	if !strings.HasSuffix(got, commentSignature) {
		t.Error("truncated body should end with comment signature")
	}
}

func TestCommentSignature_IsHTMLComment(t *testing.T) {
	if !strings.HasPrefix(commentSignature, "<!--") || !strings.HasSuffix(commentSignature, "-->") {
		t.Error("comment signature should be an HTML comment (invisible in rendered markdown)")
	}
}
