// Package plancontext implements the "context" subcommand: a second,
// interactive run that resolves an implementation plan which stopped at
// STATUS: NEEDS_CONTEXT (or BLOCKED).
//
// The implement command's planning session escalates to NEEDS_CONTEXT when the
// issue is underspecified, posts the plan as a comment, and aborts before any
// code is written. This package picks that plan back up: it derives the
// clarifying questions the plan's open questions imply (a no-clone Claude call),
// collects the maintainer's answers in a short interactive Q&A loop — the same
// composer the draft command uses — then runs a fresh read-only planning session
// with the answers folded in and posts the revised plan back on the issue. The
// next `implement` run reuses that revised plan verbatim.
//
// The package is named plancontext rather than context to avoid shadowing the
// standard library's context package; the user-facing subcommand is still
// "context".
package plancontext

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/planwerk/planwerk-review/internal/detect"
	"github.com/planwerk/planwerk-review/internal/draft/inputeditor"
	"github.com/planwerk/planwerk-review/internal/github"
	"github.com/planwerk/planwerk-review/internal/implement"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/workspace"
)

// maxQuestions caps how many clarifying questions the interactive loop asks, so
// a chatty model cannot turn the context step into an interrogation. The prompt
// already asks for at most six; this is the defensive backstop.
const maxQuestions = 6

// Options configures a context run.
type Options struct {
	IssueRef             string
	NoInteractive        bool // skip the Q&A loop and re-plan from the prior plan alone
	NoPlanComment        bool // do not post the revised plan as a comment
	DryRun               bool // report what would happen but do not clone or invoke Claude
	PrintQuestionsPrompt bool // render the questions prompt and exit
	PrintPlanPrompt      bool // render the re-plan prompt and exit
	Local                bool // operate on the current working directory instead of cloning
	Force                bool // with Local, skip the dirty-working-tree confirmation prompt
	Version              string

	// Pattern loading mirrors implement so the revised plan honors the same
	// review catalog the original plan did.
	PatternDirs     []string
	NoRepoPatterns  bool
	NoLocalPatterns bool
	MaxPatterns     int
}

// Runner executes the context pipeline using injected Claude, GitHub, and
// terminal dependencies. The shape mirrors draft.Runner (Q&A loop, TTY gating)
// and implement.Runner (clone, pattern loading, plan posting).
type Runner struct {
	Claude               ClaudeContext
	GitHub               GitHubClient
	BuildQuestionsPrompt QuestionsPromptFn
	BuildPlanPrompt      PlanPromptFn
	In                   io.Reader
	// Prompt receives the interactive Q&A prompts. It is stderr in production so
	// the prompts stay visible even when stdout is redirected and never pollute
	// the plan written to w.
	Prompt io.Writer
	IsTTY  func() bool
	// Capture is the multi-line composer used for each answer on an interactive
	// terminal. Nil-safe: when nil (or the run is non-interactive / not a TTY)
	// the single-line reads are used instead.
	Capture Capturer
}

// NewRunner wires a Runner with the production backends: the github client,
// stdin, the multi-line composer, and the TTY check. The composer renders to
// stderr, so IsTTY requires both stdin and stderr to be terminals before it
// engages.
func NewRunner(qFn QuestionsFn, planFn PlanFn, qBuild QuestionsPromptFn, planBuild PlanPromptFn) *Runner {
	return &Runner{
		Claude:               claudeFnAdapter{questions: qFn, plan: planFn},
		GitHub:               defaultGitHubClient{},
		BuildQuestionsPrompt: qBuild,
		BuildPlanPrompt:      planBuild,
		In:                   os.Stdin,
		Prompt:               os.Stderr,
		IsTTY:                func() bool { return workspace.IsStdinTTY() && workspace.IsStderrTTY() },
		Capture:              inputeditor.New(),
	}
}

// Run is a package-level convenience that delegates to NewRunner(...).Run.
func Run(w io.Writer, opts Options, qFn QuestionsFn, planFn PlanFn, qBuild QuestionsPromptFn, planBuild PlanPromptFn) error {
	return NewRunner(qFn, planFn, qBuild, planBuild).Run(w, opts)
}

// Run executes the context pipeline:
//  1. Resolve the issue and find the plan a prior implement run posted.
//  2. Require that plan to be escalated (NEEDS_CONTEXT or BLOCKED); a plan that
//     is already executable, or a missing plan, is a no-op with guidance.
//  3. In a --print-*-prompt mode: render the requested prompt and exit.
//  4. Derive clarifying questions from the issue + the escalated plan and
//     collect the maintainer's answers in the interactive Q&A loop.
//  5. Clone the repo and run a fresh read-only planning session with the prior
//     plan and the answers folded in.
//  6. Post the revised plan back on the issue (unless --no-plan-comment) and
//     report whether it reached PLAN_READY or still needs context.
func (r *Runner) Run(w io.Writer, opts Options) error {
	owner, name, number, err := github.ParseIssueRef(opts.IssueRef)
	if err != nil {
		return fmt.Errorf("parsing issue ref: %w", err)
	}
	fullName := fmt.Sprintf("%s/%s", owner, name)
	slog.Info("context starting", "issue", fmt.Sprintf("%s#%d", fullName, number))

	issue, err := r.GitHub.GetIssue(owner, name, number)
	if err != nil {
		return fmt.Errorf("fetching issue: %w", err)
	}

	priorPlan, status, err := r.findEscalatedPlan(owner, name, number, fullName)
	if err != nil {
		return err
	}

	if opts.PrintQuestionsPrompt {
		return writePrompt(w, r.BuildQuestionsPrompt(issue.Title, issue.Body, priorPlan))
	}
	if opts.PrintPlanPrompt {
		return writePrompt(w, r.BuildPlanPrompt(r.replanContext(fullName, number, issue, priorPlan, nil, opts)))
	}

	_, _ = fmt.Fprintf(w, "Found a plan on issue #%d that returned STATUS: %s. Supplying context to revise it.\n", number, status)

	// Short-circuit before any Claude call so --dry-run honors its "do not
	// clone or invoke Claude" contract: the questions step is a Claude call too.
	if opts.DryRun {
		_, _ = fmt.Fprintf(w, "[dry-run] would ask clarifying questions, clone %s, and re-plan #%d with the answers folded in\n",
			fullName, number)
		return nil
	}

	var answers []QA
	if !opts.NoInteractive {
		answers, err = r.clarify(opts, issue.Title, issue.Body, priorPlan)
		if err != nil {
			return err
		}
	}

	repo, err := r.openRepo(opts, fullName)
	if err != nil {
		return fmt.Errorf("cloning repo: %w", err)
	}
	defer repo.Cleanup()
	slog.Info("cloned repository", "dir", repo.Dir)

	ctx := r.replanContext(fullName, number, issue, priorPlan, answers, opts)
	ctx.Patterns = loadPatterns(opts, repo.Dir)

	slog.Info("running re-plan session", "issue", number, "answers", len(answers))
	revised, err := r.Claude.Plan(repo.Dir, ctx)
	if err != nil {
		return fmt.Errorf("re-planning with supplied context: %w", err)
	}
	revised = strings.TrimSpace(revised)
	if revised == "" {
		return fmt.Errorf("the re-plan produced an empty plan; nothing was posted")
	}
	_, _ = fmt.Fprintf(w, "\nRevised implementation plan:\n%s\n", revised)
	r.postPlanComment(w, opts, owner, name, number, revised)

	if newStatus := implement.PlanEscalation(revised); newStatus != "" {
		_, _ = fmt.Fprintf(w, "\nThe revised plan still reports STATUS: %s. Review the remaining open questions above; rerun `planwerk-review context %s#%d` once you can answer them.\n",
			newStatus, fullName, number)
		return nil
	}
	_, _ = fmt.Fprintf(w, "\nThe revised plan is PLAN_READY. Run `planwerk-review implement %s#%d` to implement it — it reuses this plan.\n",
		fullName, number)
	return nil
}

// findEscalatedPlan reads the issue comments, returns the most recent posted
// plan together with its escalation verdict, and rejects the cases where there
// is nothing to do: no plan posted yet, or a plan that is already executable.
// Reading the comments is load-bearing — the whole command operates on that
// plan — so a GitHub failure aborts rather than proceeding blind.
func (r *Runner) findEscalatedPlan(owner, name string, number int, fullName string) (plan, status string, err error) {
	comments, err := r.GitHub.ListIssueComments(owner, name, number)
	if err != nil {
		return "", "", fmt.Errorf("reading issue comments to find the posted plan: %w", err)
	}
	plan = implement.MostRecentPlanComment(comments)
	if plan == "" {
		return "", "", fmt.Errorf("no implementation plan is posted on issue #%d; run `planwerk-review implement %s#%d` first to produce one",
			number, fullName, number)
	}
	status = implement.PlanEscalation(plan)
	if status == "" {
		return "", "", fmt.Errorf("the implementation plan on issue #%d is already executable (it did not report NEEDS_CONTEXT or BLOCKED); run `planwerk-review implement %s#%d` to implement it",
			number, fullName, number)
	}
	return plan, status, nil
}

// replanContext assembles the implement.Context for the re-plan: the issue
// identity plus the escalated plan (PriorPlan) and the maintainer's answers
// (Clarifications). Patterns are loaded separately once the clone exists.
func (r *Runner) replanContext(fullName string, number int, issue *github.Issue, priorPlan string, answers []QA, opts Options) implement.Context {
	return implement.Context{
		RepoFullName:   fullName,
		IssueNumber:    number,
		IssueTitle:     issue.Title,
		IssueBody:      issue.Body,
		IssueURL:       issue.URL,
		IssueState:     issue.State,
		MaxPatterns:    opts.MaxPatterns,
		PriorPlan:      priorPlan,
		Clarifications: toClarifications(answers),
	}
}

// toClarifications converts the Q&A pairs into the implement.Clarification
// slice the planning prompt embeds.
func toClarifications(answers []QA) []implement.Clarification {
	if len(answers) == 0 {
		return nil
	}
	out := make([]implement.Clarification, 0, len(answers))
	for _, qa := range answers {
		out = append(out, implement.Clarification{Question: qa.Question, Answer: qa.Answer})
	}
	return out
}

// clarify generates the clarifying questions from the issue + the escalated
// plan and collects one answer per question. Mirrors draft.Runner.clarify:
// questions are written to r.Prompt (stderr) so the plan on w stays clean; on an
// interactive terminal each answer uses the multi-line composer, otherwise it
// reads one line per question and returns early on EOF so exhausted piped input
// does not error.
func (r *Runner) clarify(opts Options, issueTitle, issueBody, priorPlan string) ([]QA, error) {
	questions, err := r.Claude.ContextQuestions(issueTitle, issueBody, priorPlan)
	if err != nil {
		return nil, fmt.Errorf("generating clarifying questions: %w", err)
	}
	if len(questions) > maxQuestions {
		questions = questions[:maxQuestions]
	}
	if len(questions) == 0 {
		_, _ = fmt.Fprintln(r.Prompt, "No clarifying questions were generated; re-planning from the prior plan alone.")
		return nil, nil
	}

	reader := bufio.NewReader(r.In)
	answers := make([]QA, 0, len(questions))
	for i, q := range questions {
		if r.useComposer(opts) {
			answer, err := r.Capture.Capture(fmt.Sprintf("Q%d/%d: %s", i+1, len(questions), q), r.In, r.Prompt)
			if err != nil {
				return nil, fmt.Errorf("reading answer: %w", err)
			}
			answers = append(answers, QA{Question: q, Answer: answer})
			continue
		}

		_, _ = fmt.Fprintf(r.Prompt, "\nQ%d/%d: %s\n> ", i+1, len(questions), q)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("reading answer: %w", err)
		}
		answers = append(answers, QA{Question: q, Answer: strings.TrimSpace(line)})
		if err == io.EOF {
			break
		}
	}
	return answers, nil
}

// useComposer reports whether the interactive multi-line composer should be used
// for this run: only when the run is interactive, a composer is wired, and both
// stdin and stderr are a terminal. Everywhere else — piped stdin,
// --no-interactive, or a non-TTY — the single-line reads are used. Mirrors
// draft.Runner.useComposer.
func (r *Runner) useComposer(opts Options) bool {
	return !opts.NoInteractive && r.Capture != nil && r.IsTTY != nil && r.IsTTY()
}

// openRepo returns the working tree for the re-plan: the user's cwd when --local
// is set (no clone, Cleanup is a no-op), otherwise a fresh temp-dir clone.
// Mirrors implement.Runner.openRepo.
func (r *Runner) openRepo(opts Options, fullName string) (*github.Repo, error) {
	if opts.Local {
		repo, err := r.GitHub.CloneRepoLocal(fullName, github.LocalOptions{Force: opts.Force, Prompter: workspace.NewStdinPrompter()})
		if err != nil {
			return nil, err
		}
		slog.Info("operating on local checkout", "dir", repo.Dir)
		return repo, nil
	}
	slog.Info("cloning repository for re-plan", "repo", fullName)
	return r.GitHub.CloneRepo(fullName)
}

// postPlanComment posts the revised plan as a comment on the source issue, using
// the same comment format implement posts so the next implement run reuses it
// verbatim. Disabled by --no-plan-comment. Posting is best-effort: a GitHub
// failure is logged and surfaced but never aborts — the revised plan is already
// on stdout. Mirrors implement.Runner.postPlanComment.
func (r *Runner) postPlanComment(w io.Writer, opts Options, owner, name string, number int, plan string) {
	if opts.NoPlanComment {
		return
	}
	url, err := r.GitHub.AddIssueComment(owner, name, number, implement.FormatPlanComment(plan))
	if err != nil {
		slog.Warn("posting revised plan comment failed", "issue", number, "err", err)
		_, _ = fmt.Fprintf(w, "\nCould not post the revised plan as an issue comment: %v\n", err)
		return
	}
	slog.Info("posted revised plan comment", "issue", number, "url", url)
	_, _ = fmt.Fprintf(w, "\nPosted the revised plan as a comment on issue #%d", number)
	if url != "" {
		_, _ = fmt.Fprintf(w, " (%s)", url)
	}
	_, _ = fmt.Fprintln(w)
}

// writePrompt writes a rendered prompt to w with a trailing newline, mirroring
// the --print-prompt helpers in the implement and draft packages.
func writePrompt(w io.Writer, prompt string) error {
	if _, err := io.WriteString(w, prompt); err != nil {
		return fmt.Errorf("writing prompt: %w", err)
	}
	if !strings.HasSuffix(prompt, "\n") {
		_, _ = fmt.Fprintln(w)
	}
	return nil
}

// loadPatterns runs technology detection on the working tree and loads the
// review-pattern catalog filtered by those tags, so the revised plan is grounded
// in the same patterns the original plan used. Mirrors implement.loadPatterns;
// failures are non-fatal.
func loadPatterns(opts Options, repoDir string) []patterns.Pattern {
	tags := detect.Technologies(repoDir)
	if len(tags) > 0 {
		slog.Info("detected technologies", "technologies", strings.Join(tags, ", "))
	}
	dirs, err := patterns.Resolve(patterns.ResolveOptions{
		NoLocal: opts.NoLocalPatterns,
		NoRepo:  opts.NoRepoPatterns,
		RepoDir: repoDir,
		Extra:   opts.PatternDirs,
	})
	if err != nil {
		slog.Warn("resolving pattern sources failed; continuing without them", "err", err)
	}
	pats, err := patterns.LoadFilteredWithOptions(patterns.LoadOptions{Remote: patterns.RemoteOpts(), NoEmbedded: opts.NoLocalPatterns}, tags, dirs...)
	if err != nil {
		slog.Warn("loading review patterns failed; continuing without them", "err", err)
		return nil
	}
	if len(pats) > 0 {
		slog.Info("loaded review patterns", "count", len(pats))
	}
	return pats
}
