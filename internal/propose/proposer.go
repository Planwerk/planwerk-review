package propose

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/github"
)

// Options configures the proposal pipeline.
type Options struct {
	RepoRef string
	NoCache bool
	Format  string // "markdown", "json", "issues"
	Version string
}

// Run executes the full proposal pipeline:
// clone repo → analyze with Claude → structure proposals → render output.
func Run(w io.Writer, opts Options, analyzeFn func(dir string) (*ProposalResult, error)) error {
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

	// 3. Check cache
	cacheKey := cache.RepoKey(repo.Owner, repo.Name, headSHA)
	if !opts.NoCache {
		if data, ok := cache.GetRaw(cacheKey); ok {
			fmt.Fprintln(os.Stderr, "Using cached proposal result.")
			var result ProposalResult
			if err := json.Unmarshal(data, &result); err == nil {
				return renderProposals(w, &result, repo, opts)
			}
			// If cache is corrupted, continue with fresh analysis
			fmt.Fprintln(os.Stderr, "Cache corrupted, running fresh analysis.")
		}
	}

	// 4. Run Claude analysis
	fmt.Fprintln(os.Stderr, "Analyzing codebase with Claude...")
	result, err := analyzeFn(repo.Dir)
	if err != nil {
		return fmt.Errorf("claude analysis: %w", err)
	}

	// 5. Cache result
	if !opts.NoCache {
		if data, err := json.Marshal(result); err == nil {
			if err := cache.PutRaw(cacheKey, data); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not cache result: %v\n", err)
			}
		}
	}

	// 6. Render output
	fmt.Fprintln(os.Stderr, "Analysis complete.")
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
		return nil
	}
}
