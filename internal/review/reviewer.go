package review

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/checklist"
	"github.com/planwerk/planwerk-review/internal/claude"
	"github.com/planwerk/planwerk-review/internal/doccheck"
	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/todocheck"
)

// postCommentFunc is the function used to post PR comments.
// It is a variable so tests can replace it.
var postCommentFunc = github.PostPRComment

type Options struct {
	PRRef           string
	PatternDirs     []string
	NoRepoPatterns  bool
	NoLocalPatterns bool
	NoCache         bool
	MinSeverity     report.Severity
	Format          string
	Version         string
	PostReview      bool
	Thorough        bool
	CoverageMap     bool
}

// Run executes the full review pipeline:
// fetch & checkout PR → load patterns → claude /review → structure → render report.
func Run(w io.Writer, opts Options) error {
	// 1. Fetch and checkout PR
	fmt.Fprintf(os.Stderr, "Fetching and checking out PR %s...\n", opts.PRRef)
	pr, err := github.FetchAndCheckout(opts.PRRef)
	if err != nil {
		return fmt.Errorf("fetching PR: %w", err)
	}
	defer pr.Cleanup()

	fmt.Fprintf(os.Stderr, "Checked out to %s\n", pr.Dir)

	// 2. Check cache (include flags that affect output in the cache key)
	var cacheFlags []string
	if opts.Thorough {
		cacheFlags = append(cacheFlags, "thorough")
	}
	if opts.CoverageMap {
		cacheFlags = append(cacheFlags, "coverage-map")
	}
	cacheKey := cache.Key(pr.Owner, pr.Repo, pr.Number, pr.HeadSHA, cacheFlags...)
	if !opts.NoCache {
		if result, ok := cache.Get(cacheKey); ok {
			fmt.Fprintln(os.Stderr, "Using cached review result.")
			return renderResult(w, result, pr, opts, nil)
		}
	}

	// 3. Load patterns
	var patternDirs []string

	if !opts.NoLocalPatterns {
		// General patterns shipped with planwerk-review
		exe, err := os.Executable()
		if err == nil {
			localPatterns := filepath.Join(filepath.Dir(exe), "..", "patterns")
			if info, err := os.Stat(localPatterns); err == nil && info.IsDir() {
				patternDirs = append(patternDirs, localPatterns)
			}
		}
		// Also check relative to working directory (for development)
		if info, err := os.Stat("patterns"); err == nil && info.IsDir() {
			patternDirs = append(patternDirs, "patterns")
		}
	}

	if !opts.NoRepoPatterns {
		// Repo-specific patterns from the checked-out PR repo
		repoPatterns := filepath.Join(pr.Dir, ".planwerk", "review_patterns")
		if info, err := os.Stat(repoPatterns); err == nil && info.IsDir() {
			patternDirs = append(patternDirs, repoPatterns)
		}
	}

	patternDirs = append(patternDirs, opts.PatternDirs...)

	pats, err := patterns.Load(patternDirs...)
	if err != nil {
		return fmt.Errorf("loading patterns: %w", err)
	}

	if len(pats) > 0 {
		fmt.Fprintf(os.Stderr, "Loaded %d review pattern(s).\n", len(pats))
	}

	// 4. Load checklist
	checklistContent := checklist.Load(pr.Dir)

	// 5. Fetch commit log for scope drift detection
	commitLog := getCommitLog(pr.Dir, pr.BaseBranch)

	// 6. Check for stale documentation
	staleDocs := doccheck.Check(pr.Dir, pr.BaseBranch)

	// 7. Load TODOS.md for cross-reference
	todoContent := todocheck.Load(pr.Dir)

	// 8. Run Claude /review + structuring
	fmt.Fprintln(os.Stderr, "Running Claude /review...")
	ctx := claude.ReviewContext{
		Patterns:    pats,
		PRTitle:     pr.Title,
		PRBody:      pr.Body,
		Checklist:   checklistContent,
		CommitLog:   commitLog,
		StaleDocs:   staleDocs,
		TodoContent: todoContent,
	}
	result, err := claude.Review(pr.Dir, ctx)
	if err != nil {
		return fmt.Errorf("claude review: %w", err)
	}

	// 9. Adversarial review pass (if --thorough)
	if opts.Thorough {
		fmt.Fprintln(os.Stderr, "Running adversarial review pass...")
		advResult, err := claude.AdversarialReview(pr.Dir, pr.BaseBranch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: adversarial review failed: %v\n", err)
		} else {
			result = mergeResults(result, advResult)
		}
	}

	// 10. Coverage map (if --coverage-map)
	var coverageResult *report.CoverageResult
	if opts.CoverageMap {
		fmt.Fprintln(os.Stderr, "Generating test coverage map...")
		covResult, err := claude.CoverageMap(pr.Dir, pr.BaseBranch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: coverage map failed: %v\n", err)
		} else {
			coverageResult = covResult
		}
	}

	// 11. Cache result
	if !opts.NoCache {
		if err := cache.Put(cacheKey, result); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not cache result: %v\n", err)
		}
	}

	// 12. Render
	fmt.Fprintln(os.Stderr, "Review complete.")
	return renderResult(w, result, pr, opts, coverageResult)
}

func renderResult(w io.Writer, result *report.ReviewResult, pr *github.PR, opts Options, coverage *report.CoverageResult) error {
	prInfo := report.PRInfo{
		Owner:  pr.Owner,
		Repo:   pr.Repo,
		Number: pr.Number,
		Title:  pr.Title,
	}

	// If posting review, capture output in a buffer as well
	var buf bytes.Buffer
	output := io.Writer(w)
	if opts.PostReview {
		output = io.MultiWriter(w, &buf)
	}

	renderer := report.NewRenderer(output)

	switch opts.Format {
	case "json":
		if err := renderer.RenderJSON(*result, opts.MinSeverity); err != nil {
			return err
		}
	default:
		renderer.RenderMarkdown(*result, prInfo, opts.MinSeverity, opts.Version)
	}

	// Append coverage map if available
	if coverage != nil {
		report.RenderCoverageMap(output, *coverage)
	}

	if opts.PostReview {
		fmt.Fprintln(os.Stderr, "Posting review as PR comment (will update existing if found)...")
		result, err := postCommentFunc(pr.Owner, pr.Repo, pr.Number, buf.String())
		if err != nil {
			return fmt.Errorf("posting PR comment: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Review posted: %s\n", result)
	}

	return nil
}

// gitLogTimeout is the maximum time allowed for local git log operations.
const gitLogTimeout = 30 * time.Second

// getCommitLog returns the one-line commit log between the base branch and HEAD.
// Returns empty string on error (non-fatal).
func getCommitLog(dir, baseBranch string) string {
	if baseBranch == "" {
		baseBranch = claude.DefaultBaseBranch
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitLogTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "log", "origin/"+baseBranch+"..HEAD", "--oneline")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
