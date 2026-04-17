package propose

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/github"
)

// Options configures the proposal pipeline.
type Options struct {
	RepoRef       string
	NoCache       bool
	Format        string // "markdown", "json", "issues"
	Version       string
	CreateIssues  bool // interactive issue creation after proposal generation
	NoIssueDedupe bool // skip filtering proposals against existing GitHub issues
}

// Runner executes the propose pipeline using injected Claude and GitHub
// clients. Constructing a Runner per invocation keeps dependencies explicit
// and allows tests to run in parallel without mutating package-level state.
type Runner struct {
	Claude ClaudeAnalyzer
	GitHub GitHubClient
}

// NewRunner returns a Runner wired with the production GitHub (git/gh CLI)
// backend and the given Claude analyze function.
func NewRunner(analyzeFn AnalyzeFn) *Runner {
	return &Runner{
		Claude: analyzeFnAdapter{fn: analyzeFn},
		GitHub: defaultGitHubClient{},
	}
}

// Run is a package-level convenience that delegates to NewRunner(analyzeFn).Run.
// Callers that need to inject alternative Claude or GitHub backends should
// construct a Runner directly.
func Run(w io.Writer, opts Options, analyzeFn AnalyzeFn) error {
	return NewRunner(analyzeFn).Run(w, opts)
}

// Run executes the full proposal pipeline: resolve HEAD SHA, check cache, and
// on a miss clone the repo, run Claude, cache, and render. Cache hits skip the
// clone entirely so CI loops against the same commit don't pay the clone cost.
func (r *Runner) Run(w io.Writer, opts Options) error {
	owner, name, err := github.ParseRepoRef(opts.RepoRef)
	if err != nil {
		return fmt.Errorf("parsing repo ref: %w", err)
	}

	// Resolve HEAD SHA via git ls-remote before cloning, so a cache hit can
	// short-circuit the clone entirely.
	headSHA, err := r.GitHub.DefaultBranchHEAD(owner, name)
	if err != nil {
		slog.Warn("could not fetch HEAD SHA, caching disabled", "err", err)
		opts.NoCache = true
		headSHA = ""
	}

	cacheKey := cache.RepoKey(owner, name, headSHA)
	if !opts.NoCache && headSHA != "" {
		if data, ok := cache.GetRaw(cacheKey); ok {
			var result ProposalResult
			if err := json.Unmarshal(data, &result); err == nil {
				slog.Info("using cached proposal result — skipping clone", "repo", opts.RepoRef)
				r.applyIssueDedupe(&result, owner, name, opts)
				return renderProposals(w, &result, &github.Repo{Owner: owner, Name: name}, opts)
			}
			slog.Warn("cache corrupted, running fresh analysis")
		}
	}

	slog.Info("cloning repository", "repo", opts.RepoRef)
	repo, err := r.GitHub.CloneRepo(opts.RepoRef)
	if err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}
	defer repo.Cleanup()

	slog.Info("cloned repository", "dir", repo.Dir)

	slog.Info("analyzing codebase with Claude")
	result, err := r.Claude.Analyze(repo.Dir)
	if err != nil {
		return fmt.Errorf("claude analysis: %w", err)
	}

	if !opts.NoCache && headSHA != "" {
		if data, err := json.Marshal(result); err == nil {
			if err := cache.PutRaw(cacheKey, data); err != nil {
				slog.Warn("could not cache result", "err", err)
			}
		}
	}

	slog.Info("analysis complete")
	r.applyIssueDedupe(result, owner, name, opts)
	return renderProposals(w, result, repo, opts)
}

// applyIssueDedupe filters out proposals whose title matches an existing
// GitHub issue (open or closed) in the target repo. Dedupe runs after the
// cache layer so cached Claude output stays faithful to what Claude returned —
// issue state changes take effect on every run without invalidating the cache.
// A lister error logs a warning and skips dedupe rather than failing the run.
func (r *Runner) applyIssueDedupe(result *ProposalResult, owner, name string, opts Options) {
	if opts.NoIssueDedupe || r.GitHub == nil {
		return
	}
	existing, err := r.GitHub.ListExistingIssues(owner, name)
	if err != nil {
		slog.Warn("could not list existing issues, skipping dedupe", "err", err)
		return
	}
	idx := github.BuildTitleIndex(existing)
	if len(idx) == 0 {
		return
	}
	before := len(result.Proposals)
	kept := make([]Proposal, 0, before)
	for _, p := range result.Proposals {
		if match, ok := idx.Lookup(p.Title); ok {
			slog.Debug("skipping proposal already tracked by an existing issue",
				"title", p.Title, "existing", match.URL)
			continue
		}
		kept = append(kept, p)
	}
	result.Proposals = kept
	if filtered := before - len(kept); filtered > 0 {
		slog.Info("filtered proposals with existing issues",
			"filtered", filtered, "kept", len(kept))
	}
}

func renderProposals(w io.Writer, result *ProposalResult, repo *github.Repo, opts Options) error {
	renderer := NewRenderer(w)

	switch opts.Format {
	case "json":
		return renderer.RenderJSON(*result)
	case "issues":
		renderer.RenderIssues(*result, repo.FullName())
		return nil
	default:
		renderer.RenderMarkdown(*result, repo.FullName(), opts.Version)
	}

	if opts.CreateIssues {
		return RunInteractiveIssueCreation(w, result, repo.Owner, repo.Name)
	}

	return nil
}
