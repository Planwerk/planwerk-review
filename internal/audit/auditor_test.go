package audit

import (
	"testing"
)

func TestCollectPatternDirs_IncludesExplicitDirs(t *testing.T) {
	opts := Options{
		PatternDirs:     []string{"/explicit/patterns"},
		NoLocalPatterns: true,
		NoRepoPatterns:  true,
	}
	dirs := collectPatternDirs(opts, "/tmp/repo")

	if len(dirs) != 1 || dirs[0] != "/explicit/patterns" {
		t.Errorf("dirs = %v, want only /explicit/patterns", dirs)
	}
}

func TestCollectPatternDirs_HonorsNoRepoPatterns(t *testing.T) {
	opts := Options{
		NoLocalPatterns: true,
		NoRepoPatterns:  true,
	}
	dirs := collectPatternDirs(opts, "/tmp/repo-that-does-not-exist")

	for _, d := range dirs {
		if d == "/tmp/repo-that-does-not-exist/.planwerk/review_patterns" {
			t.Error("repo patterns should be skipped when NoRepoPatterns is true")
		}
	}
}

func TestCollectPatternDirs_EmptyWhenEverythingDisabled(t *testing.T) {
	// This test runs from the audit package directory, which has no patterns/
	// subdirectory, so local-patterns fallback should not trigger. If it ever
	// does (future layout change), this assertion still holds: with both flags
	// set we expect no pattern directories to be returned.
	opts := Options{
		NoLocalPatterns: true,
		NoRepoPatterns:  true,
	}
	dirs := collectPatternDirs(opts, "/tmp/does-not-exist")

	if len(dirs) != 0 {
		t.Errorf("expected no pattern dirs with both flags disabled and no explicit dirs, got %v", dirs)
	}
}
