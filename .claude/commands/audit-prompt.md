---
description: Audit a planwerk-agent prompt builder against the prompt-design doctrine (delegates to the prompt-auditor subagent).
argument-hint: <builder-name-or-file>
---

Audit the prompt builder `$ARGUMENTS` against the project's prompt-design
doctrine by delegating to the **prompt-auditor** subagent.

- If `$ARGUMENTS` is empty, list the candidate builders — the `build*Prompt` /
  `Build*Prompt` functions in `internal/claude/*.go` — and ask which one to audit
  before going further.
- If `$ARGUMENTS` names a function, locate it in `internal/claude/*.go`. If it
  names a file, audit every builder in that file.
- Delegate the analysis to the prompt-auditor subagent — do **not** audit it
  yourself in this context — and relay its structured report verbatim.
