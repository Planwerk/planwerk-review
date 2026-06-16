# Supply context to a NEEDS_CONTEXT plan

When `implement` runs its read-only planning session and finds the issue
underspecified, the plan ends in `STATUS: NEEDS_CONTEXT` (or `BLOCKED`), is
posted on the issue, and the run aborts before any code is written. Use
`context` to resolve that plan in a second, interactive pass — it asks you the
open questions, then re-plans with your answers and posts a revised plan the
next `implement` run reuses.

```bash
# Resolve the plan implement posted on the issue
planwerk-review context owner/repo#123

# Re-plan from the prior plan alone, without the Q&A (or pipe answers on stdin)
planwerk-review context --no-interactive owner/repo#123
printf 'do it here\nopenbao-cluster-store\n' | planwerk-review context owner/repo#123

# Inspect the prompts without invoking Claude
planwerk-review context --print-questions-prompt owner/repo#123
planwerk-review context --print-plan-prompt owner/repo#123
```

See the [CLI reference](/reference/cli#context) for every flag, including the
`--plan-model` / `--plan-effort` re-plan overrides and `--local`.

## How it works

1. **Find the plan**: The issue's comments are read and the most recent plan
   `implement` posted is located — by its `## Implementation Plan` heading and
   attribution footer, exactly as `implement`'s own plan-reuse does. A missing
   plan, or one that is already `PLAN_READY`, is a no-op with guidance.
2. **Require escalation**: The plan must report `NEEDS_CONTEXT` or `BLOCKED`;
   otherwise there is nothing to clarify and the command points you back at
   `implement`.
3. **Generate questions**: A no-clone Claude call turns the plan's "Risks &
   Open Questions" into a short list of concrete questions — the scope,
   contradiction, and either/or decisions only a human can make. The plan
   already did the repository analysis, so this step does not re-clone.
4. **Q&A loop**: You answer each question in the same multi-line composer the
   `draft` command uses (Enter for a newline, Ctrl-D to submit, Ctrl-E for
   `$EDITOR`). Piped stdin reads one answer per line; `--no-interactive` skips
   the loop entirely.
5. **Re-plan**: The repository is cloned (or the current checkout is used with
   `--local`) and the read-only planning session runs again — on the dedicated
   `--plan-model` / `--plan-effort` — with the prior plan and your answers
   folded in as authoritative context, so it can resolve the open questions and
   aim for `STATUS: PLAN_READY`.
6. **Post the revised plan**: The revised plan is posted back onto the issue
   (unless `--no-plan-comment`), in the same comment format `implement` reuses.
   Run `implement` next: it picks up the revised plan verbatim. If the revised
   plan still reports `NEEDS_CONTEXT`, the remaining open questions are printed
   — rerun `context` once you can answer them.

The command is interactive by design, so it is kept separate from `implement`,
which runs unattended in auto mode (including under the GitHub Action).
