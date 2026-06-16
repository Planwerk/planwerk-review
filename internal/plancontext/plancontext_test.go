package plancontext

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/implement"
)

// --- fakes -----------------------------------------------------------------

type fakeClaude struct {
	questions    []string
	questionsErr error
	qTitle       string
	qBody        string
	qPlan        string
	qCalls       int

	plan     string
	planErr  error
	planCtx  implement.Context
	planDir  string
	planCall int
}

func (f *fakeClaude) ContextQuestions(title, body, prior string) ([]string, error) {
	f.qCalls++
	f.qTitle, f.qBody, f.qPlan = title, body, prior
	return f.questions, f.questionsErr
}

func (f *fakeClaude) Plan(dir string, ctx implement.Context) (string, error) {
	f.planCall++
	f.planDir = dir
	f.planCtx = ctx
	return f.plan, f.planErr
}

type fakeGitHub struct {
	issue       *github.Issue
	issueErr    error
	comments    []github.IssueComment
	commentsErr error
	cloneDir    string
	cloneErr    error

	cloneCalls      int
	cloneLocalCalls int
	commentErr      error
	commentBodies   []string
}

func (g *fakeGitHub) GetIssue(owner, name string, number int) (*github.Issue, error) {
	if g.issueErr != nil {
		return nil, g.issueErr
	}
	iss := *g.issue
	iss.Owner, iss.Name, iss.Number = owner, name, number
	return &iss, nil
}

func (g *fakeGitHub) ListIssueComments(_, _ string, _ int) ([]github.IssueComment, error) {
	if g.commentsErr != nil {
		return nil, g.commentsErr
	}
	return g.comments, nil
}

func (g *fakeGitHub) CloneRepo(ref string) (*github.Repo, error) {
	g.cloneCalls++
	if g.cloneErr != nil {
		return nil, g.cloneErr
	}
	owner, name, err := github.ParseRepoRef(ref)
	if err != nil {
		return nil, err
	}
	return &github.Repo{Owner: owner, Name: name, Dir: g.cloneDir}, nil
}

func (g *fakeGitHub) CloneRepoLocal(ref string, _ github.LocalOptions) (*github.Repo, error) {
	g.cloneLocalCalls++
	if g.cloneErr != nil {
		return nil, g.cloneErr
	}
	owner, name, err := github.ParseRepoRef(ref)
	if err != nil {
		return nil, err
	}
	return &github.Repo{Owner: owner, Name: name, Dir: g.cloneDir, Local: true}, nil
}

func (g *fakeGitHub) AddIssueComment(owner, name string, number int, body string) (string, error) {
	if g.commentErr != nil {
		return "", g.commentErr
	}
	g.commentBodies = append(g.commentBodies, body)
	return "https://github.com/owner/repo/issues/42#issuecomment-1", nil
}

// --- helpers ---------------------------------------------------------------

func sampleIssue() *github.Issue {
	return &github.Issue{
		Title: "Add foo widget",
		Body:  "## Description\n\nFoo widget does X.\n",
		URL:   "https://github.com/owner/repo/issues/42",
		State: "open",
	}
}

// planComment wraps a plan body the way implement posts it, so
// MostRecentPlanComment recognizes it.
func planComment(status string) github.IssueComment {
	plan := "## Implementation Plan (issue #42)\n\n### Status\nSTATUS: " + status
	return github.IssueComment{Body: implement.FormatPlanComment(plan)}
}

func newTestRunner(gh *fakeGitHub, cl *fakeClaude, stdin string) *Runner {
	return &Runner{
		Claude:               cl,
		GitHub:               gh,
		BuildQuestionsPrompt: func(t, b, p string) string { return "QUESTIONS-PROMPT" },
		BuildPlanPrompt:      func(ctx implement.Context) string { return "PLAN-PROMPT prior=" + ctx.PriorPlan },
		In:                   strings.NewReader(stdin),
		Prompt:               io.Discard,
		IsTTY:                func() bool { return false },
		Capture:              nil,
	}
}

// --- tests -----------------------------------------------------------------

func TestRun_NoPriorPlan_Errors(t *testing.T) {
	gh := &fakeGitHub{issue: sampleIssue()} // no comments
	cl := &fakeClaude{}
	err := newTestRunner(gh, cl, "").Run(&bytes.Buffer{}, Options{IssueRef: "owner/repo#42"})
	if err == nil || !strings.Contains(err.Error(), "no implementation plan is posted") {
		t.Fatalf("Run err = %v, want a 'no plan posted' error", err)
	}
	if cl.planCall != 0 {
		t.Errorf("Plan was called %d times, want 0", cl.planCall)
	}
}

func TestRun_ExecutablePlan_Errors(t *testing.T) {
	gh := &fakeGitHub{issue: sampleIssue(), comments: []github.IssueComment{planComment("PLAN_READY")}}
	cl := &fakeClaude{}
	err := newTestRunner(gh, cl, "").Run(&bytes.Buffer{}, Options{IssueRef: "owner/repo#42"})
	if err == nil || !strings.Contains(err.Error(), "already executable") {
		t.Fatalf("Run err = %v, want an 'already executable' error", err)
	}
	if cl.qCalls != 0 || cl.planCall != 0 {
		t.Errorf("no Claude call expected for an executable plan; got q=%d plan=%d", cl.qCalls, cl.planCall)
	}
}

func TestRun_ReplansWithSuppliedContext(t *testing.T) {
	gh := &fakeGitHub{
		issue:    sampleIssue(),
		comments: []github.IssueComment{planComment("NEEDS_CONTEXT")},
		cloneDir: t.TempDir(),
	}
	cl := &fakeClaude{
		questions: []string{"Scope it here or split?", "Which store name?"},
		plan:      "## Implementation Plan (issue #42)\n\n### Status\nSTATUS: PLAN_READY",
	}
	var buf bytes.Buffer
	err := newTestRunner(gh, cl, "do it here\nopenbao-cluster-store\n").
		Run(&buf, Options{IssueRef: "owner/repo#42"})
	if err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	// Questions were generated from the issue + the escalated plan.
	if cl.qCalls != 1 {
		t.Fatalf("ContextQuestions called %d times, want 1", cl.qCalls)
	}
	if cl.qTitle != "Add foo widget" || !strings.Contains(cl.qPlan, "STATUS: NEEDS_CONTEXT") {
		t.Errorf("questions call got title=%q plan=%q", cl.qTitle, cl.qPlan)
	}

	// The re-plan ran in the clone with the prior plan + answers folded in.
	if cl.planCall != 1 {
		t.Fatalf("Plan called %d times, want 1", cl.planCall)
	}
	if cl.planDir != gh.cloneDir {
		t.Errorf("re-plan ran in %q, want clone dir %q", cl.planDir, gh.cloneDir)
	}
	if !strings.Contains(cl.planCtx.PriorPlan, "STATUS: NEEDS_CONTEXT") {
		t.Errorf("re-plan PriorPlan = %q, want the escalated plan", cl.planCtx.PriorPlan)
	}
	if len(cl.planCtx.Clarifications) != 2 {
		t.Fatalf("re-plan got %d clarifications, want 2", len(cl.planCtx.Clarifications))
	}
	if cl.planCtx.Clarifications[0].Question != "Scope it here or split?" ||
		cl.planCtx.Clarifications[0].Answer != "do it here" {
		t.Errorf("first clarification = %+v, want the first Q paired with the first answer", cl.planCtx.Clarifications[0])
	}
	if cl.planCtx.Clarifications[1].Answer != "openbao-cluster-store" {
		t.Errorf("second clarification answer = %q", cl.planCtx.Clarifications[1].Answer)
	}

	// The revised plan was posted in the same format implement reuses.
	if len(gh.commentBodies) != 1 {
		t.Fatalf("AddIssueComment posted %d comments, want 1", len(gh.commentBodies))
	}
	reused := implement.MostRecentPlanComment([]github.IssueComment{{Body: gh.commentBodies[0]}})
	if reused != strings.TrimSpace(cl.plan) {
		t.Errorf("posted comment is not reusable as a plan:\nreused=%q\nwant=%q", reused, cl.plan)
	}
	if !strings.Contains(buf.String(), "PLAN_READY") ||
		!strings.Contains(buf.String(), "implement owner/repo#42") {
		t.Errorf("output missing the ready/next-step guidance:\n%s", buf.String())
	}
}

func TestRun_RevisedStillNeedsContext(t *testing.T) {
	gh := &fakeGitHub{
		issue:    sampleIssue(),
		comments: []github.IssueComment{planComment("NEEDS_CONTEXT")},
		cloneDir: t.TempDir(),
	}
	cl := &fakeClaude{
		questions: []string{"Still unclear?"},
		plan:      "## Implementation Plan (issue #42)\n\n### Status\nSTATUS: NEEDS_CONTEXT",
	}
	var buf bytes.Buffer
	if err := newTestRunner(gh, cl, "dunno\n").Run(&buf, Options{IssueRef: "owner/repo#42"}); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	if len(gh.commentBodies) != 1 {
		t.Fatalf("AddIssueComment posted %d comments, want 1 (the revised plan still posts)", len(gh.commentBodies))
	}
	if !strings.Contains(buf.String(), "still reports STATUS: NEEDS_CONTEXT") {
		t.Errorf("output should flag the unresolved status:\n%s", buf.String())
	}
}

func TestRun_NoPlanComment_SkipsPosting(t *testing.T) {
	gh := &fakeGitHub{
		issue:    sampleIssue(),
		comments: []github.IssueComment{planComment("NEEDS_CONTEXT")},
		cloneDir: t.TempDir(),
	}
	cl := &fakeClaude{
		questions: []string{"Q?"},
		plan:      "## Implementation Plan (issue #42)\n\nSTATUS: PLAN_READY",
	}
	if err := newTestRunner(gh, cl, "a\n").Run(&bytes.Buffer{}, Options{IssueRef: "owner/repo#42", NoPlanComment: true}); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	if len(gh.commentBodies) != 0 {
		t.Errorf("--no-plan-comment must not post; got %d comments", len(gh.commentBodies))
	}
}

func TestRun_NoInteractive_SkipsQuestions(t *testing.T) {
	gh := &fakeGitHub{
		issue:    sampleIssue(),
		comments: []github.IssueComment{planComment("NEEDS_CONTEXT")},
		cloneDir: t.TempDir(),
	}
	cl := &fakeClaude{plan: "## Implementation Plan (issue #42)\n\nSTATUS: PLAN_READY"}
	if err := newTestRunner(gh, cl, "").Run(&bytes.Buffer{}, Options{IssueRef: "owner/repo#42", NoInteractive: true}); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	if cl.qCalls != 0 {
		t.Errorf("--no-interactive must skip the questions call; got %d", cl.qCalls)
	}
	if cl.planCall != 1 || len(cl.planCtx.Clarifications) != 0 {
		t.Errorf("re-plan should run with no clarifications; planCall=%d clarifications=%d", cl.planCall, len(cl.planCtx.Clarifications))
	}
}

func TestRun_DryRun_NoCloneNoReplan(t *testing.T) {
	gh := &fakeGitHub{
		issue:    sampleIssue(),
		comments: []github.IssueComment{planComment("NEEDS_CONTEXT")},
		cloneDir: t.TempDir(),
	}
	cl := &fakeClaude{questions: []string{"Q?"}}
	var buf bytes.Buffer
	if err := newTestRunner(gh, cl, "a\n").Run(&buf, Options{IssueRef: "owner/repo#42", DryRun: true}); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	if gh.cloneCalls != 0 || cl.planCall != 0 || cl.qCalls != 0 {
		t.Errorf("dry-run must not clone or invoke Claude; clone=%d plan=%d questions=%d", gh.cloneCalls, cl.planCall, cl.qCalls)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("dry-run output missing marker:\n%s", buf.String())
	}
}

func TestRun_PrintQuestionsPrompt(t *testing.T) {
	gh := &fakeGitHub{issue: sampleIssue(), comments: []github.IssueComment{planComment("NEEDS_CONTEXT")}}
	cl := &fakeClaude{}
	var buf bytes.Buffer
	if err := newTestRunner(gh, cl, "").Run(&buf, Options{IssueRef: "owner/repo#42", PrintQuestionsPrompt: true}); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	if !strings.Contains(buf.String(), "QUESTIONS-PROMPT") {
		t.Errorf("expected the questions prompt on stdout, got:\n%s", buf.String())
	}
	if cl.qCalls != 0 || cl.planCall != 0 {
		t.Errorf("print mode must not invoke Claude; q=%d plan=%d", cl.qCalls, cl.planCall)
	}
}

func TestRun_CommentsFetchFails(t *testing.T) {
	gh := &fakeGitHub{issue: sampleIssue(), commentsErr: errors.New("github down")}
	err := newTestRunner(gh, &fakeClaude{}, "").Run(&bytes.Buffer{}, Options{IssueRef: "owner/repo#42"})
	if err == nil || !strings.Contains(err.Error(), "reading issue comments") {
		t.Fatalf("Run err = %v, want a comment-fetch error", err)
	}
}
