package plancontext

import (
	"io"

	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/implement"
)

// QA is a single clarifying question and the human's answer collected during
// the interactive context loop. It is converted to implement.Clarification when
// the re-plan context is assembled.
type QA struct {
	Question string
	Answer   string
}

// ClaudeContext drives the two Claude calls the context pipeline needs: one to
// derive clarifying questions from the issue plus the escalated plan, and one
// read-only planning session that re-plans with the answers folded in. The
// package depends on this interface so tests can inject a deterministic fake,
// mirroring draft.ClaudeDrafter and implement.ClaudePlanner.
type ClaudeContext interface {
	ContextQuestions(issueTitle, issueBody, priorPlan string) ([]string, error)
	Plan(dir string, ctx implement.Context) (string, error)
}

// QuestionsFn is the bare-function form of the question generator the CLI wires
// in (claude.ContextQuestions). It is adapted to ClaudeContext via claudeFnAdapter.
type QuestionsFn func(issueTitle, issueBody, priorPlan string) ([]string, error)

// PlanFn is the bare-function form of the read-only planning session the CLI
// wires in (claude.Plan). The context re-plan reuses the same planning session
// the implement command runs — same planning model and effort — only with
// Context.PriorPlan and Context.Clarifications set.
type PlanFn func(dir string, ctx implement.Context) (string, error)

// QuestionsPromptFn renders the questions prompt without invoking Claude
// (--print-questions-prompt mode), wired to claude.BuildContextQuestionsPrompt.
type QuestionsPromptFn func(issueTitle, issueBody, priorPlan string) string

// PlanPromptFn renders the planning prompt without invoking Claude
// (--print-plan-prompt mode), wired to claude.BuildPlanPrompt.
type PlanPromptFn func(ctx implement.Context) string

// claudeFnAdapter adapts a QuestionsFn + PlanFn pair to ClaudeContext.
type claudeFnAdapter struct {
	questions QuestionsFn
	plan      PlanFn
}

func (a claudeFnAdapter) ContextQuestions(issueTitle, issueBody, priorPlan string) ([]string, error) {
	return a.questions(issueTitle, issueBody, priorPlan)
}

func (a claudeFnAdapter) Plan(dir string, ctx implement.Context) (string, error) {
	return a.plan(dir, ctx)
}

// Capturer captures one block of user text for an interactive prompt. It is the
// same shape draft.Capturer uses, so *inputeditor.Editor satisfies it
// structurally; the package depends on the local interface so tests can inject a
// deterministic fake.
type Capturer interface {
	Capture(prompt string, in io.Reader, out io.Writer) (string, error)
}

// GitHubClient is the subset of github operations the context command needs:
// fetch the source issue, list its comments (to find the plan a prior implement
// run posted), clone the repository for the read-only re-plan, and post the
// revised plan back as a comment. Mirrors implement.GitHubClient so the two
// commands read and write the same plan comments.
type GitHubClient interface {
	GetIssue(owner, name string, number int) (*github.Issue, error)
	ListIssueComments(owner, name string, number int) ([]github.IssueComment, error)
	CloneRepo(ref string) (*github.Repo, error)
	CloneRepoLocal(ref string, opts github.LocalOptions) (*github.Repo, error)
	AddIssueComment(owner, name string, number int, body string) (string, error)
}

// defaultGitHubClient is the production GitHubClient backed by the github
// package. Mirrors the implement package's adapter shape.
type defaultGitHubClient struct{}

func (defaultGitHubClient) GetIssue(owner, name string, number int) (*github.Issue, error) {
	return github.GetIssue(owner, name, number)
}

func (defaultGitHubClient) ListIssueComments(owner, name string, number int) ([]github.IssueComment, error) {
	return github.ListIssueComments(owner, name, number)
}

func (defaultGitHubClient) CloneRepo(ref string) (*github.Repo, error) {
	return github.CloneRepo(ref)
}

func (defaultGitHubClient) CloneRepoLocal(ref string, opts github.LocalOptions) (*github.Repo, error) {
	return github.UseLocalRepo(ref, opts)
}

func (defaultGitHubClient) AddIssueComment(owner, name string, number int, body string) (string, error) {
	return github.AddIssueComment(owner, name, number, body)
}
