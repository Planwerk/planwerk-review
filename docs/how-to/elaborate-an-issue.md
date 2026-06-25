# Elaborate an issue

Take a high-level GitHub issue (typically the output of `propose` or `audit`)
and expand it into a deeply detailed engineering plan grounded in the actual
repository state — the kind of issue body a senior engineer can pick up and
execute without further clarification (mirrors the structure shown in
[plexsphere/plexsphere#10](https://github.com/plexsphere/plexsphere/issues/10):
Description with concrete "already exists / this story adds" boundaries,
Motivation, Affected Areas, Acceptance Criteria, Non-Goals, References).

```bash
# Render the elaborated body to stdout
planwerk-agent elaborate https://github.com/owner/repo/issues/123

# Short form
planwerk-agent elaborate owner/repo#123

# JSON for automation
planwerk-agent elaborate --format json owner/repo#123

# Replace the issue body with the elaborated body
planwerk-agent elaborate --update-issue owner/repo#123

# Or post the elaboration as a new comment instead
planwerk-agent elaborate --post-comment owner/repo#123
```

`--update-issue` and `--post-comment` are mutually exclusive — pick the one that
matches your team's workflow (overwrite the source issue vs. preserve history
and append a follow-up comment). See the
[CLI reference](/reference/cli#elaborate) for every flag.

## How it works

1. **Issue Input**: The tool receives a GitHub issue reference (URL or `owner/repo#number`).
2. **Fetch Issue**: Title, body, URL, and state are fetched via `gh issue view`.
3. **Fetch Relations**: When the issue is a **Sub Issue** of a Meta Issue, the Meta Issue and the other Sub Issues are fetched via the GitHub GraphQL API (best-effort — a repo without sub-issue links, a missing token scope, or an older GitHub Enterprise Server degrades to "no relations" without failing the run). See [Sub Issues are elaborated against their Meta Issue](#sub-issues-are-elaborated-against-their-meta-issue) below.
4. **Cache Check**: The default-branch HEAD SHA is resolved via `gh api graphql`. The cache key combines repo + HEAD + issue number + a fingerprint of the issue body — plus, when the issue is a Sub Issue, a fingerprint of the Meta Issue and sibling Sub Issues — so the cache invalidates automatically when the repo, the issue, the Meta Issue, or any sibling is edited.
5. **Clone**: On a cache miss, the repository is cloned locally.
6. **Pattern Load**: The same pattern catalog used by `review` / `audit` / `propose` is loaded, filtered by detected technologies.
7. **Claude Elaboration**: Claude is instructed to walk the repo first, identify what already exists vs. what the issue adds, and emit a detailed plan in six sections (Description with concrete "already exists / this story adds" boundaries, Motivation, Affected Areas, Acceptance Criteria, Non-Goals, References). For a Sub Issue, the Meta Issue and sibling Sub Issues from step 3 are injected so the elaboration covers only this issue's slice and defers adjacent parts to the sibling that owns them.
8. **Structuring**: A second Claude call converts the elaboration into a strict JSON schema so the final body renders consistently.
9. **Output**: The elaborated body is rendered as Markdown (default) or JSON. With `--update-issue`, the issue body is overwritten; with `--post-comment`, the elaboration is posted as a new comment.

## Sub Issues are elaborated against their Meta Issue

When the issue is a **Sub Issue** created by [`meta`](/how-to/split-a-meta-issue)
(or linked through GitHub's native sub-issue relationship), `elaborate` reads the
**Meta Issue** and the **other Sub Issues** alongside it and injects them into the
prompt as a *Meta / Sub-Issue Context* section. The elaboration is then told to:

- plan only this Sub Issue's slice of the larger effort and honor the Meta
  Issue's framing rather than re-deciding it;
- avoid duplicating work a sibling Sub Issue owns; and
- when this Sub Issue intentionally implements only part of a shared task because
  the remaining part lands in another Sub Issue, scope it to its part and
  cross-reference the sibling that carries the rest (e.g. *"the remaining X is
  handled by #K"*), recording the deferral under Non-Goals.

A closed sibling is treated as already-implemented context to build on; an open
one as work that may land in parallel. This is automatic — there is no flag — and
best-effort: an issue that is not a Sub Issue, or a repo where the relationship
cannot be read, elaborates exactly as before.

## Score the draft before output (`--review`)

`--review` adds a reviewer pass between elaboration and output. A reviewer
scores the draft from 0 to 10 for executability — a 10 is a plan a zero-context
implementer executes without asking a single question. While the score stays
below the bar, the refine loop revises the draft to close the reviewer's gaps
and iterates until the score clears the bar or `--max-review-iterations` is
exhausted (default 3).

The final score is surfaced in the output as `Executability score: N/10`, so a
near-miss is visible rather than hidden behind a binary pass/fail. When the loop
runs out of iterations below the bar, the surviving gaps and a "what a 10/10
plan would look like" target are rendered alongside the score under **Reviewer
Notes (unresolved)** — address them before implementing.
