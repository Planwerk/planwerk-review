package github

import "testing"

func TestParseRef(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		owner   string
		repo    string
		number  int
		wantErr bool
	}{
		{
			name:   "URL form",
			ref:    "https://github.com/planwerk/planwerk-review/pull/42",
			owner:  "planwerk",
			repo:   "planwerk-review",
			number: 42,
		},
		{
			name:   "short form",
			ref:    "planwerk/planwerk-review#42",
			owner:  "planwerk",
			repo:   "planwerk-review",
			number: 42,
		},
		{
			name:   "short form with whitespace",
			ref:    "  planwerk/planwerk-review#42  ",
			owner:  "planwerk",
			repo:   "planwerk-review",
			number: 42,
		},
		{
			name:   "dots and underscores in names",
			ref:    "my.org/my_repo#1",
			owner:  "my.org",
			repo:   "my_repo",
			number: 1,
		},
		{
			name:    "empty string",
			ref:     "",
			wantErr: true,
		},
		{
			name:    "missing number",
			ref:     "owner/repo",
			wantErr: true,
		},
		{
			name:    "invalid owner characters",
			ref:     "ow ner/repo#1",
			wantErr: true,
		},
		{
			name:    "invalid repo characters",
			ref:     "owner/re po#1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, number, err := ParseRef(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got owner=%q repo=%q number=%d", owner, repo, number)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.owner {
				t.Errorf("owner = %q, want %q", owner, tt.owner)
			}
			if repo != tt.repo {
				t.Errorf("repo = %q, want %q", repo, tt.repo)
			}
			if number != tt.number {
				t.Errorf("number = %d, want %d", number, tt.number)
			}
		})
	}
}
