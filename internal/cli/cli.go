package cli

import (
	"github.com/planwerk/planwerk-review/internal/propose"
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/review"
)

// Config holds configuration for the review command.
type Config struct {
	PRRef           string
	PatternDirs     []string
	MinSeverity     report.Severity
	NoRepoPatterns  bool
	NoLocalPatterns bool
	NoCache         bool
	ClearCache      bool
	Format          string
	PostReview      bool
	InlineReview    bool
	Thorough        bool
	CoverageMap     bool
}

func (c Config) ToReviewOptions(version string) review.Options {
	return review.Options{
		PRRef:           c.PRRef,
		PatternDirs:     c.PatternDirs,
		NoRepoPatterns:  c.NoRepoPatterns,
		NoLocalPatterns: c.NoLocalPatterns,
		NoCache:         c.NoCache,
		MinSeverity:     c.MinSeverity,
		Format:          c.Format,
		Version:         version,
		PostReview:      c.PostReview,
		InlineReview:    c.InlineReview,
		Thorough:        c.Thorough,
		CoverageMap:     c.CoverageMap,
	}
}

// ProposeConfig holds configuration for the propose command.
type ProposeConfig struct {
	RepoRef      string
	NoCache      bool
	Format       string // "markdown", "json", "issues"
	CreateIssues bool
}

func (c ProposeConfig) ToProposeOptions(version string) propose.Options {
	return propose.Options{
		RepoRef:      c.RepoRef,
		NoCache:      c.NoCache,
		Format:       c.Format,
		Version:      version,
		CreateIssues: c.CreateIssues,
	}
}
