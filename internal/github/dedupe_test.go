package github

import "testing"

func TestNormalizeIssueTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"Foo", "foo"},
		{"  Foo  ", "foo"},
		{"Foo.", "foo"},
		{"Foo!!", "foo"},
		{"Foo: Bar", "foo: bar"},
		{"Foo:", "foo"},
		{"Add   Logging\tNow", "add logging now"},
		{"[BLOCKING] Foo", "[blocking] foo"},
		{"MiXeD CaSe?", "mixed case"},
	}
	for _, tc := range tests {
		got := NormalizeIssueTitle(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeIssueTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildTitleIndexAndLookup(t *testing.T) {
	existing := []ExistingIssue{
		{Title: "Add logging", URL: "u1"},
		{Title: "Fix crash on startup.", URL: "u2"},
		{Title: "  extra  whitespace  ", URL: "u3"},
		{Title: "", URL: "u4"},            // should be skipped
		{Title: "Add LOGGING", URL: "u5"}, // duplicate of u1; first wins
	}
	idx := BuildTitleIndex(existing)

	if len(idx) != 3 {
		t.Fatalf("index size = %d, want 3", len(idx))
	}

	// Case- and punctuation-insensitive hits.
	hit, ok := idx.Lookup("add logging!")
	if !ok || hit.URL != "u1" {
		t.Errorf("lookup add-logging: ok=%v url=%q, want u1", ok, hit.URL)
	}
	hit, ok = idx.Lookup("Fix crash on startup")
	if !ok || hit.URL != "u2" {
		t.Errorf("lookup fix-crash: ok=%v url=%q, want u2", ok, hit.URL)
	}
	hit, ok = idx.Lookup("EXTRA WHITESPACE")
	if !ok || hit.URL != "u3" {
		t.Errorf("lookup whitespace: ok=%v url=%q, want u3", ok, hit.URL)
	}

	// Misses.
	if _, ok := idx.Lookup("not there"); ok {
		t.Error("unexpected hit for missing title")
	}
	if _, ok := idx.Lookup(""); ok {
		t.Error("empty string must not match anything")
	}
}
