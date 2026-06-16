package claude

import (
	"fmt"
	"strings"
)

// ContextQuestions asks Claude for the short list of clarifying questions a
// maintainer must answer to lift a NEEDS_CONTEXT implementation plan to
// PLAN_READY. It reads the issue and the escalated plan's open questions only —
// no checkout — mirroring DraftQuestions; the subsequent re-plan re-grounds in
// the repository. It is the first of the context subcommand's two Claude calls.
func ContextQuestions(issueTitle, issueBody, priorPlan string) ([]string, error) {
	text, err := runClaude("", BuildContextQuestionsPrompt(issueTitle, issueBody, priorPlan), "context-questions")
	if err != nil {
		return nil, fmt.Errorf("generating context questions: %w", err)
	}
	var payload struct {
		Questions []string `json:"questions"`
	}
	if err := decodeJSONWithRepair(text, "context questions", &payload); err != nil {
		return nil, err
	}
	return payload.Questions, nil
}

// BuildContextQuestionsPrompt assembles the clarifying-questions prompt for the
// context subcommand: it turns the prior plan's open questions and scope
// decisions into a short list of concrete questions only a human can answer.
// Exported so the subcommand can render the prompt without invoking Claude
// (--print-questions-prompt mode).
//
// Unlike the draft questions prompt — which asks in the seed's language so the
// author can answer in their own words — these questions stay in English: the
// input here is the English issue and the English plan, and the artifact the
// answers feed (the revised plan) is English too (design-decision #26).
func BuildContextQuestionsPrompt(issueTitle, issueBody, priorPlan string) string {
	var sb strings.Builder

	sb.WriteString(`A read-only planning session produced an implementation plan for the GitHub issue below and stopped at STATUS: NEEDS_CONTEXT (or BLOCKED): the issue is underspecified, so it cannot be implemented as written. Your job is to ask the maintainer the few questions whose answers would let a fresh planning session resolve the plan's open questions and reach STATUS: PLAN_READY.

Read the plan's "Risks & Open Questions" and its STATUS rationale first — that is where the missing context lives. Ask only the decisions a human must actually make: scope reconciliation (do it here vs split into its own issue), contradictions between the issue and the current code, and the genuine either/or choices the plan surfaced. Do NOT ask about implementation details the plan already settled, and do NOT re-derive the analysis — the plan already did that.

`)

	fmt.Fprintf(&sb, "## Issue\n\n- Title: %s\n\n<issue-body>\n%s\n</issue-body>\n\n",
		strings.TrimSpace(issueTitle), strings.TrimSpace(issueBody))

	sb.WriteString("## Plan that returned NEEDS_CONTEXT\n\n<plan>\n")
	sb.WriteString(strings.TrimSpace(priorPlan))
	sb.WriteString("\n</plan>\n\n")

	sb.WriteString(`Output between 1 and 6 questions, each a single short sentence the maintainer can answer in a line or two. Order them most-blocking first. Where the plan already laid out concrete options (a/b/c), fold them into the question so the answer can just pick one. Write the questions in English, matching the issue and the plan.

Output ONLY valid JSON (no markdown fences, no surrounding text):

{
  "questions": ["First question?", "Second question?"]
}`)

	return sb.String()
}
