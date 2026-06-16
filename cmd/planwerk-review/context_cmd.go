package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planwerk/planwerk-review/internal/claude"
	"github.com/planwerk/planwerk-review/internal/cli"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/plancontext"
)

// newContextCmd builds the "context" subcommand: a second, interactive run that
// resolves an implementation plan which stopped at STATUS: NEEDS_CONTEXT (or
// BLOCKED). It finds the plan a prior implement run posted on the issue, asks
// the maintainer the few questions its open questions imply (like draft), then
// re-runs the read-only planning session with those answers folded in and posts
// the revised plan back — ready for the next implement run to reuse.
func newContextCmd(deps *runtimeDeps) *cobra.Command {
	var contextCfg cli.ContextConfig
	var planModel string
	var planEffort string

	contextCmd := &cobra.Command{
		Use:   "context <issue-ref>",
		Short: "Supply missing context to a NEEDS_CONTEXT plan and revise it to PLAN_READY",
		Long: `Pick up an implementation plan that an earlier "implement" run posted on
the issue and stopped on because it reported STATUS: NEEDS_CONTEXT (or
BLOCKED) — the issue was underspecified. This command resolves that plan in a
second, interactive pass:

  1. It reads the posted plan and asks Claude for the few clarifying questions
     its "Risks & Open Questions" imply — the scope, contradiction, and
     either/or decisions only a human can make. This step needs no checkout.
  2. It collects your answers in a short interactive Q&A loop, the same
     multi-line composer the draft command uses.
  3. It clones the repository and re-runs the read-only planning session — on
     the dedicated planning model (--plan-model) and effort (--plan-effort) —
     with the prior plan and your answers folded in, so it can resolve the open
     questions and aim for STATUS: PLAN_READY.
  4. It posts the revised plan back onto the issue as a comment (unless
     --no-plan-comment). The next "implement" run reuses that revised plan
     verbatim, so no code is written until the plan is ready.

If the revised plan still reports NEEDS_CONTEXT, the remaining open questions
are printed; rerun "context" once you can answer them.

Use --no-interactive to skip the Q&A and re-plan from the prior plan alone, or
pipe one answer per line on stdin for scripted runs. Use
--print-questions-prompt / --print-plan-prompt to render the prompts without
invoking Claude.

Issue reference can be a URL (https://github.com/owner/repo/issues/123)
or short form (owner/repo#123).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextCfg.IssueRef = args[0]
			maxPatterns, err := resolveMaxPatterns(contextCfg.MaxPatterns, cmd.Flags().Changed("max-patterns"), nil)
			if err != nil {
				return err
			}
			contextCfg.MaxPatterns = maxPatterns
			modes := 0
			if contextCfg.DryRun {
				modes++
			}
			if contextCfg.PrintQuestionsPrompt {
				modes++
			}
			if contextCfg.PrintPlanPrompt {
				modes++
			}
			if modes > 1 {
				return fmt.Errorf("--dry-run, --print-questions-prompt, and --print-plan-prompt are mutually exclusive")
			}
			claude.SetPlanModel(resolvePlanModel(planModel, cmd.Flags().Changed("plan-model")))
			claude.SetPlanEffort(resolvePlanEffort(planEffort, cmd.Flags().Changed("plan-effort")))
			opts := contextCfg.ToContextOptions(deps.version)
			return plancontext.Run(cmd.OutOrStdout(), opts,
				claude.ContextQuestions, claude.Plan,
				claude.BuildContextQuestionsPrompt, claude.BuildPlanPrompt)
		},
	}

	contextFlags := contextCmd.Flags()
	contextFlags.BoolVarP(&contextCfg.NoInteractive, "no-interactive", "y", false, "Skip the clarifying Q&A loop and re-plan from the prior plan alone")
	contextFlags.BoolVar(&contextCfg.NoPlanComment, "no-plan-comment", false, "Do not post the revised plan as a comment on the source issue")
	contextFlags.BoolVar(&contextCfg.DryRun, "dry-run", false, "Report what would happen but do not clone or invoke Claude")
	contextFlags.BoolVar(&contextCfg.PrintQuestionsPrompt, "print-questions-prompt", false, "Render the clarifying-questions prompt (with the issue and posted plan embedded) to stdout and exit")
	contextFlags.BoolVar(&contextCfg.PrintPlanPrompt, "print-plan-prompt", false, "Render the re-plan prompt (with the prior plan embedded) to stdout and exit; do not clone or invoke Claude")
	contextFlags.StringVar(&planModel, "plan-model", claude.DefaultPlanModel, "Model for the re-plan session passed to Claude Code via --model (e.g. fable, opus; env: "+envPlanModel+")")
	contextFlags.StringVar(&planEffort, "plan-effort", claude.DefaultPlanEffort, "Reasoning effort for the re-plan session passed to Claude Code via --effort (low, medium, high, xhigh, max; env: "+envPlanEffort+")")
	contextFlags.StringSliceVar(&contextCfg.PatternDirs, "patterns", nil, "Additional pattern sources: local dirs, github:owner/repo[/sub][@ref], or git+https://...[#ref[:sub]]")
	contextFlags.BoolVar(&contextCfg.NoRepoPatterns, "no-repo-patterns", false, "Ignore repo-specific patterns under .planwerk/review_patterns/ in the target repo")
	contextFlags.BoolVar(&contextCfg.NoLocalPatterns, "no-local-patterns", false, "Ignore local patterns from the tool")
	contextFlags.IntVar(&contextCfg.MaxPatterns, "max-patterns", patterns.DefaultMaxPatternsInPrompt, "Max review patterns injected into the prompt (<=0 disables truncation, env: "+envMaxPatterns+")")
	contextFlags.BoolVar(&contextCfg.Local, "local", false, "Operate on the current working directory instead of cloning into a temp dir")
	contextFlags.BoolVar(&contextCfg.Force, "force", false, "With --local, skip the confirmation prompt when the working tree is dirty")

	return contextCmd
}
