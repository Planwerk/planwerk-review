package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	RepoRef          string
	PatternDirs      []string
	NoRepoPatterns   bool
	NoLocalPatterns  bool
	NoCache          bool
	MinSeverity      report.Severity
	Format           string // "markdown" or "json"
	Version          string
	MaxPatterns      int             // max patterns to inject into prompt; <= 0 disables truncation
	MaxFindings      int             // cap on findings Claude returns; <= 0 disables cap
	CreateIssues     bool            // interactively create GitHub issues after audit
	IssueMinSeverity report.Severity // minimum severity for a finding group to become an issue candidate
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

// Runner executes the audit pipeline using injected Claude and GitHub
// clients. Constructing a Runner per invocation keeps dependencies explicit
// and allows tests to run in parallel without mutating package-level state.
type Runner struct {
	Claude ClaudeAuditor
	GitHub GitHubClient
}

// NewRunner returns a Runner wired with the production GitHub (git/gh CLI)
// backend and the given Claude audit function.
func NewRunner(auditFn AuditFn) *Runner {
	return &Runner{
		Claude: auditFnAdapter{fn: auditFn},
		GitHub: defaultGitHubClient{},
	}
}

// Run is a package-level convenience that delegates to NewRunner(auditFn).Run.
// Callers that need to inject alternative Claude or GitHub backends should
// construct a Runner directly.
func Run(w io.Writer, opts Options, auditFn AuditFn) error {
	return NewRunner(auditFn).Run(w, opts)
}

// Run executes the full audit pipeline:
// clone repo → detect technologies → load patterns → Claude audit → render report.
func (r *Runner) Run(w io.Writer, opts Options) error {
	// 1. Clone the repository
	slog.Info("cloning repository", "repo", opts.RepoRef)
	repo, err := r.GitHub.CloneRepo(opts.RepoRef)
	if err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}
	defer repo.Cleanup()

	slog.Info("cloned repository", "dir", repo.Dir)

	// 2. Fetch HEAD SHA for cache key (so cache invalidates when repo changes)
	headSHA, err := r.GitHub.DefaultBranchHEAD(repo.Owner, repo.Name)
	if err != nil {
		slog.Warn("could not fetch HEAD SHA, caching disabled", "err", err)
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
			slog.Info("using cached audit result")
			var result report.ReviewResult
			if err := json.Unmarshal(data, &result); err == nil {
				return renderAudit(w, &result, repo, opts)
			}
			slog.Warn("cache corrupted, running fresh audit")
		}
	}

	// 5. Detect technologies in the cloned repo
	techTags := detect.Technologies(repo.Dir)
	if len(techTags) > 0 {
		slog.Info("detected technologies", "technologies", strings.Join(techTags, ", "))
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
	slog.Info("loaded review patterns", "count", len(pats))

	// 7. Run Claude audit
	slog.Info("auditing codebase with Claude")
	result, err := r.Claude.Audit(repo.Dir, AuditContext{
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
				slog.Warn("could not cache result", "err", err)
			}
		}
	}

	// 9. Render output
	slog.Info("audit complete")
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

	if opts.CreateIssues {
		return RunInteractiveIssueCreation(w, os.Stdin, result, repo.Owner, repo.Name, opts.IssueMinSeverity)
	}
	return nil
}
