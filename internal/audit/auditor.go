package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/detect"
	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/report"
)

// Options configures the audit pipeline.
type Options struct {
	RepoRef         string
	PatternDirs     []string
	NoRepoPatterns  bool
	NoLocalPatterns bool
	NoCache         bool
	MinSeverity     report.Severity
	Format          string // "markdown" or "json"
	Version         string
	MaxPatterns     int // max patterns to inject into prompt; <= 0 disables truncation
	MaxFindings     int // cap on findings Claude returns; <= 0 disables cap
}

// AuditFn performs the Claude-backed codebase audit for a cloned repo.
// It is injected so tests can substitute a fake implementation without
// invoking the real Claude CLI.
type AuditFn func(dir string, ctx AuditContext) (*report.ReviewResult, error)

// AuditContext holds all context needed to build the audit prompt.
type AuditContext struct {
	Patterns    []patterns.Pattern
	MaxPatterns int
	MaxFindings int
	RepoName    string // "owner/repo" for context in the prompt
}

// Run executes the full audit pipeline:
// clone repo → detect technologies → load patterns → Claude audit → render report.
func Run(w io.Writer, opts Options, auditFn AuditFn) error {
	// 1. Clone the repository
	fmt.Fprintf(os.Stderr, "Cloning repository %s...\n", opts.RepoRef)
	repo, err := github.CloneRepo(opts.RepoRef)
	if err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}
	defer repo.Cleanup()

	fmt.Fprintf(os.Stderr, "Cloned to %s\n", repo.Dir)

	// 2. Fetch HEAD SHA for cache key (so cache invalidates when repo changes)
	headSHA, err := github.DefaultBranchHEAD(repo.Owner, repo.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch HEAD SHA, caching disabled: %v\n", err)
		opts.NoCache = true
		headSHA = ""
	}

	// 3. Build cache key (includes min-severity so filtered caches don't leak)
	var cacheFlags []string
	if opts.MinSeverity != "" {
		cacheFlags = append(cacheFlags, "min="+string(opts.MinSeverity))
	}
	cacheKey := cache.AuditKey(repo.Owner, repo.Name, headSHA, cacheFlags...)

	// 4. Check cache
	if !opts.NoCache && headSHA != "" {
		if data, ok := cache.GetRaw(cacheKey); ok {
			fmt.Fprintln(os.Stderr, "Using cached audit result.")
			var result report.ReviewResult
			if err := json.Unmarshal(data, &result); err == nil {
				return renderAudit(w, &result, repo, opts)
			}
			fmt.Fprintln(os.Stderr, "Cache corrupted, running fresh audit.")
		}
	}

	// 5. Detect technologies in the cloned repo
	techTags := detect.Technologies(repo.Dir)
	if len(techTags) > 0 {
		fmt.Fprintf(os.Stderr, "Detected technologies: %s\n", strings.Join(techTags, ", "))
	}

	// 6. Load patterns (filtered by detected technologies)
	patternDirs := collectPatternDirs(opts, repo.Dir)
	pats, err := patterns.LoadFiltered(techTags, patternDirs...)
	if err != nil {
		return fmt.Errorf("loading patterns: %w", err)
	}
	if len(pats) == 0 {
		return fmt.Errorf("no review patterns loaded — nothing to audit against")
	}
	fmt.Fprintf(os.Stderr, "Loaded %d review pattern(s).\n", len(pats))

	// 7. Run Claude audit
	fmt.Fprintln(os.Stderr, "Auditing codebase with Claude...")
	result, err := auditFn(repo.Dir, AuditContext{
		Patterns:    pats,
		MaxPatterns: opts.MaxPatterns,
		MaxFindings: opts.MaxFindings,
		RepoName:    repo.FullName(),
	})
	if err != nil {
		return fmt.Errorf("claude audit: %w", err)
	}

	// 8. Cache result
	if !opts.NoCache && headSHA != "" {
		if data, err := json.Marshal(result); err == nil {
			if err := cache.PutRaw(cacheKey, data); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not cache result: %v\n", err)
			}
		}
	}

	// 9. Render output
	fmt.Fprintln(os.Stderr, "Audit complete.")
	return renderAudit(w, result, repo, opts)
}

// collectPatternDirs assembles the list of pattern directories to load from,
// honoring the --no-local-patterns and --no-repo-patterns flags and appending
// any explicit --patterns directories supplied by the caller.
func collectPatternDirs(opts Options, repoDir string) []string {
	var patternDirs []string

	if !opts.NoLocalPatterns {
		if exe, err := os.Executable(); err == nil {
			localPatterns := filepath.Join(filepath.Dir(exe), "..", "patterns")
			if info, err := os.Stat(localPatterns); err == nil && info.IsDir() {
				patternDirs = append(patternDirs, localPatterns)
			}
		}
		if info, err := os.Stat("patterns"); err == nil && info.IsDir() {
			patternDirs = append(patternDirs, "patterns")
		}
	}

	if !opts.NoRepoPatterns {
		repoPatterns := filepath.Join(repoDir, ".planwerk", "review_patterns")
		if info, err := os.Stat(repoPatterns); err == nil && info.IsDir() {
			patternDirs = append(patternDirs, repoPatterns)
		}
	}

	patternDirs = append(patternDirs, opts.PatternDirs...)
	return patternDirs
}

func renderAudit(w io.Writer, result *report.ReviewResult, repo *github.Repo, opts Options) error {
	renderer := report.NewRenderer(w)
	repoInfo := report.RepoInfo{
		Owner: repo.Owner,
		Name:  repo.Name,
	}

	switch opts.Format {
	case "json":
		return renderer.RenderJSON(*result, opts.MinSeverity)
	default:
		renderer.RenderAuditMarkdown(*result, repoInfo, opts.MinSeverity, opts.Version)
	}
	return nil
}
