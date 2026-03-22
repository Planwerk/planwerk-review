package cli

import (
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/review"
)

type Config struct {
	PRRef           string
	PatternDirs     []string
	MinSeverity     report.Severity
	NoRepoPatterns  bool
	NoLocalPatterns bool
	NoCache         bool
	ClearCache      bool
	Format          string
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
	}
}
