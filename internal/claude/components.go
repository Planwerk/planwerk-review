package claude

import "strings"

// This file holds prompt building blocks that are shared across more than one
// prompt builder. Keeping them in one place stops the copies from drifting
// (the failure mode that motivated extracting them): before this, the audit
// prompt carried a shortened Suppressions list and the adversarial and
// compliance prompts had no anti-hedging discipline at all.
//
// Blocks that carry intentional, builder-specific variation (the Staff
// Engineer persona, Verification of Claims, and Finding Enrichment differ
// between a diff review and a whole-codebase audit) are deliberately NOT
// shared here — forcing them into one shape would inject diff-only wording
// into the codebase audit and vice versa.

// promptScope distinguishes a diff-scoped review (a PR or branch comparison)
// from a whole-codebase audit. It selects the scope-specific suppression
// bullets so a single source can serve both without leaking diff-only wording
// into the codebase audit.
type promptScope int

const (
	// scopeDiff is a review that only considers added/modified lines relative
	// to a base branch (review, adversarial, compliance).
	scopeDiff promptScope = iota
	// scopeCodebase is a review of the entire current repository state (audit).
	scopeCodebase
)

// suppressionsBlock returns the "## Suppressions — DO NOT flag these" section.
// The common bullets apply to every review type; the two diff-only bullets
// (already-addressed-in-the-same-diff, and only-review-changed-lines) are
// emitted only for scopeDiff, where a diff actually exists.
//
// For scopeDiff this reproduces the canonical review suppression list verbatim.
func suppressionsBlock(scope promptScope) string {
	bullets := []string{
		`TODO/FIXME comments that reference an issue tracker (e.g. TODO(#123))`,
		`Missing tests for trivial getters/setters, simple delegation methods, or configuration constants — this does NOT suppress missing tests for functions with logic or branching`,
		`Import ordering or formatting differences (these are handled by formatters)`,
		`Variable naming that follows the project's existing conventions, even if you'd prefer different names`,
		`Missing documentation on unexported/private functions or internal implementation details — this does NOT suppress missing documentation for new public APIs, CLI flags, or user-facing behavior changes`,
		`Minor style preferences that don't affect correctness or readability`,
		`"X is redundant with Y" when the redundancy is harmless and aids readability (defense in depth)`,
		`Threshold or constant comments that would rot faster than the code they describe`,
		`Assertions that already cover the behavior being tested (e.g. "this assertion could be tighter")`,
		`Consistency-only suggestions ("use X style everywhere") with no correctness impact`,
	}
	if scope == scopeDiff {
		bullets = append(bullets, `Issues that are already addressed elsewhere in the same diff — read the FULL diff before commenting`)
	}
	bullets = append(bullets,
		`Suggestions to "add logging" when the error path already returns a descriptive error`,
		`"Consider using X library" when the current approach works correctly — this does NOT suppress flagging deprecated, unmaintained, or severely outdated versions of NEWLY INTRODUCED dependencies`,
	)
	if scope == scopeDiff {
		bullets = append(bullets, `Code that was not changed in this diff — only review and comment on added or modified lines, never on unchanged surrounding context`)
	}

	var b strings.Builder
	b.WriteString("## Suppressions — DO NOT flag these\n\n")
	for _, bl := range bullets {
		b.WriteString("- ")
		b.WriteString(bl)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// proseStyleBlock returns the "## Prose Style" section applied to builders that
// generate narrative text a human reads — elaborate, propose, gap analysis,
// review-prepared. The rules raise writing quality (lead-first, concrete,
// active voice, no AI-slop vocabulary) and are adapted from the
// econ-writing-skill reference.
//
// The concreteness rule is deliberately subordinated to accuracy: a model told
// only to "be concrete" will fabricate file paths and numbers to sound
// specific. The block states that genuine unknowns are marked as assumptions,
// never invented — so it cooperates with, rather than fights, each builder's
// anti-hallucination rules.
func proseStyleBlock() string {
	return `## Prose Style

Apply these rules to all prose you write (descriptions, motivations, summaries, issue bodies):

- Lead with the most important information; never bury it. State the one core point in the first sentence.
- Be concrete: name the actual behavior, component, file, or change — not "improve the system" or "various aspects". This rule is subordinate to accuracy: NEVER invent a specific (a file path, symbol, or number) just to sound concrete. When a specific is genuinely unknown, mark it as an assumption rather than fabricating it.
- Active voice, present tense. Short, common words ("use", not "utilize"). One idea per paragraph, topic sentence first.
- Cut ruthlessly. Delete throat-clearing openers ("It should be noted that", "It is worth noting that", "In other words", "This contributes by"). If a sentence adds nothing, remove it.
- Never use: delve, landscape, multifaceted, notably, crucial, comprehensive, furthermore, leverage (as a verb), robust (outside its statistical sense), pivotal, groundbreaking, shed light on, pave the way.
- Vary sentence length. Do not dress up your own work with adjectives ("critical fix", "powerful feature"). Write "This change…", not a bare "This…".

`
}

// communicationStyleBlock returns the anti-sycophancy "## Communication Style"
// section. Directness is universal across every review type, so the same
// block is shared verbatim by review, audit, adversarial, and compliance.
func communicationStyleBlock() string {
	return `## Communication Style

Be direct and decisive in your findings. Do NOT hedge:
- Do NOT write "you might want to consider..." — state what IS wrong
- Do NOT write "this could potentially cause..." — state what WILL happen
- Do NOT write "it might be worth looking into..." — state the specific problem
- Take a clear position on every finding. If something is wrong, say it is wrong.
- If something is fine, do not mention it at all.

`
}
