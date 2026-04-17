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
	RepoRef      string
	NoCache      bool
	Format       string // "markdown", "json", "issues"
	Version      string
	CreateIssues bool // interactive issue creation after proposal generation
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

// Run executes the full proposal pipeline:
// clone repo → analyze with Claude → structure proposals → render output.
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

	// 3. Check cache
	cacheKey := cache.RepoKey(repo.Owner, repo.Name, headSHA)
	if !opts.NoCache {
		if data, ok := cache.GetRaw(cacheKey); ok {
			slog.Info("using cached proposal result")
			var result ProposalResult
			if err := json.Unmarshal(data, &result); err == nil {
				return renderProposals(w, &result, repo, opts)
			}
			// If cache is corrupted, continue with fresh analysis
			slog.Warn("cache corrupted, running fresh analysis")
		}
	}

	// 4. Run Claude analysis
	slog.Info("analyzing codebase with Claude")
	result, err := r.Claude.Analyze(repo.Dir)
	if err != nil {
		return fmt.Errorf("claude analysis: %w", err)
	}

	// 5. Cache result
	if !opts.NoCache {
		if data, err := json.Marshal(result); err == nil {
			if err := cache.PutRaw(cacheKey, data); err != nil {
				slog.Warn("could not cache result", "err", err)
			}
		}
	}

	// 6. Render output
	slog.Info("analysis complete")
	return renderProposals(w, result, repo, opts)
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
