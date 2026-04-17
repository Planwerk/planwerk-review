package propose

import (
	"github.com/planwerk/planwerk-review/internal/github"
)

// ClaudeAnalyzer performs the Claude-backed codebase analysis that produces
// feature proposals. The propose package depends on this interface rather
// than the concrete claude package so tests can inject fakes.
type ClaudeAnalyzer interface {
	Analyze(dir string) (*ProposalResult, error)
}

// GitHubClient wraps the GitHub operations the propose pipeline needs:
// cloning the repository, resolving the default-branch HEAD for cache keying,
// and listing existing issues for duplicate detection. Tests inject mock
// clients to avoid touching the real git or gh CLI.
type GitHubClient interface {
	CloneRepo(ref string) (*github.Repo, error)
	DefaultBranchHEAD(owner, name string) (string, error)
	ListExistingIssues(owner, name string) ([]github.ExistingIssue, error)
}

// AnalyzeFn is the bare-function form of ClaudeAnalyzer that existing callers
// (the CLI) pass in. It is adapted to the interface via analyzeFnAdapter.
type AnalyzeFn func(dir string) (*ProposalResult, error)

type analyzeFnAdapter struct {
	fn AnalyzeFn
}

func (a analyzeFnAdapter) Analyze(dir string) (*ProposalResult, error) {
	return a.fn(dir)
}

// defaultGitHubClient is the production GitHubClient backed by the github package.
type defaultGitHubClient struct{}

func (defaultGitHubClient) CloneRepo(ref string) (*github.Repo, error) {
	return github.CloneRepo(ref)
}

func (defaultGitHubClient) DefaultBranchHEAD(owner, name string) (string, error) {
	return github.DefaultBranchHEAD(owner, name)
}

func (defaultGitHubClient) ListExistingIssues(owner, name string) ([]github.ExistingIssue, error) {
	return github.ListAllIssues(owner, name)
}
