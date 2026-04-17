package cli

import (
	"time"

	"github.com/planwerk/planwerk-review/internal/audit"
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
	ClearCacheScope string // restrict --clear-cache to a single command (review/propose/audit); empty = all
	CacheStats      bool
	CacheInspect    string // cache key to print contents + metadata for
	CacheMaxAge     time.Duration
	Format          string
	PostReview      bool
	InlineReview    bool
	Thorough        bool
	CoverageMap     bool
	MaxPatterns     int
	MaxFindings     int
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
		MaxPatterns:     c.MaxPatterns,
		MaxFindings:     c.MaxFindings,
		CacheMaxAge:     c.CacheMaxAge,
	}
}

// ProposeConfig holds configuration for the propose command.
type ProposeConfig struct {
	RepoRef         string
	PatternDirs     []string
	NoRepoPatterns  bool
	NoLocalPatterns bool
	NoCache         bool
	Format          string // "markdown", "json", "issues"
	MaxPatterns     int
	CreateIssues    bool
	NoIssueDedupe   bool
	CacheMaxAge     time.Duration
}

func (c ProposeConfig) ToProposeOptions(version string) propose.Options {
	return propose.Options{
		RepoRef:         c.RepoRef,
		PatternDirs:     c.PatternDirs,
		NoRepoPatterns:  c.NoRepoPatterns,
		NoLocalPatterns: c.NoLocalPatterns,
		NoCache:         c.NoCache,
		Format:          c.Format,
		Version:         version,
		MaxPatterns:     c.MaxPatterns,
		CreateIssues:    c.CreateIssues,
		NoIssueDedupe:   c.NoIssueDedupe,
		CacheMaxAge:     c.CacheMaxAge,
	}
}

// AuditConfig holds configuration for the audit command.
type AuditConfig struct {
	RepoRef          string
	PatternDirs      []string
	NoRepoPatterns   bool
	NoLocalPatterns  bool
	NoCache          bool
	MinSeverity      report.Severity
	Format           string // "markdown" or "json"
	MaxPatterns      int
	MaxFindings      int
	CreateIssues     bool
	IssueMinSeverity report.Severity
	NoIssueDedupe    bool
	CacheMaxAge      time.Duration
}

func (c AuditConfig) ToAuditOptions(version string) audit.Options {
	return audit.Options{
		RepoRef:          c.RepoRef,
		PatternDirs:      c.PatternDirs,
		NoRepoPatterns:   c.NoRepoPatterns,
		NoLocalPatterns:  c.NoLocalPatterns,
		NoCache:          c.NoCache,
		MinSeverity:      c.MinSeverity,
		Format:           c.Format,
		Version:          version,
		MaxPatterns:      c.MaxPatterns,
		MaxFindings:      c.MaxFindings,
		CreateIssues:     c.CreateIssues,
		IssueMinSeverity: c.IssueMinSeverity,
		NoIssueDedupe:    c.NoIssueDedupe,
		CacheMaxAge:      c.CacheMaxAge,
	}
}
