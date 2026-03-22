# planwerk-review

AI-powered code review and codebase analysis tool for GitHub repositories. Uses Claude CLI to automatically analyze PR changes and produce structured review results, or to analyze entire repositories and generate actionable feature proposals.

## Concept

### Overview

```
Review:
┌──────────────┐     ┌──────────────────┐     ┌───────────────┐     ┌──────────────┐
│  GitHub PR   │────▶│  planwerk-review │────▶│  Claude CLI   │────▶│  Markdown    │
│  (URL/Ref)   │     │                  │     │  /review      │     │  Report      │
└──────────────┘     └──────────────────┘     └───────────────┘     └──────────────┘
                            │                                              │
                            ▼                                              ▼
                     ┌──────────────────┐                          ┌──────────────┐
                     │ Review Patterns  │                          │  stdout      │
                     │ (local + repo)   │                          │  (Copy/Paste)│
                     └──────────────────┘                          └──────────────┘

Propose:
┌──────────────┐     ┌──────────────────┐     ┌───────────────┐     ┌──────────────┐
│  GitHub Repo │────▶│  planwerk-review │────▶│  Claude CLI   │────▶│  Proposals   │
│  (URL/Ref)   │     │  propose         │     │  (analysis)   │     │  (MD/JSON)   │
└──────────────┘     └──────────────────┘     └───────────────┘     └──────────────┘
                            │                        │
                            ▼                        ▼
                     ┌──────────────────┐     ┌───────────────┐
                     │ Cache (SHA-based)│     │  Structure    │
                     │                  │     │  into JSON    │
                     └──────────────────┘     └───────────────┘
```

### Review Workflow

1. **PR Input**: The tool receives a GitHub PR as input (URL or `owner/repo#number`).
2. **Checkout**: The PR is checked out locally (diff between base and head). PR title and description are fetched for scope analysis.
3. **Load Review Patterns**: Patterns are loaded from two sources:
   - `patterns/` in the planwerk-review repository (general patterns)
   - `.planwerk/review_patterns/` in the target repository (repo-specific patterns)
4. **Claude CLI Review**: `claude /review` is executed with a structured prompt that includes persona framing, scope analysis, a two-pass checklist, suppression rules, and review patterns.
5. **Result Aggregation**: Review results are collected, deduplicated, categorized by severity, and classified by actionability.
6. **Markdown Output**: A structured report is written to `stdout`.

### Review Methodology

The review prompt uses techniques inspired by [gstack](https://github.com/garrytan/gstack) to maximize review quality:

#### Staff Engineer Persona

Claude is instructed to review as a Staff Engineer, applying specific cognitive patterns:
- *"What happens at 10x scale?"* — Load, data volume, concurrent users
- *"What's the blast radius?"* — If this code fails, what else breaks?
- *"What happens at 3am?"* — Error paths, oncall clarity, log quality
- *"Would a new team member understand this?"* — Code clarity and intent

#### Scope Drift Detection

Before reviewing code quality, the tool checks for:
- **Scope Creep**: Files changed that are unrelated to the PR title/description
- **Missing Requirements**: Requirements from the PR description not addressed in the diff

#### Two-Pass Review Checklist

Claude works through a structured checklist in two passes:

| Pass | Focus | Categories |
|------|-------|------------|
| **Pass 1 — Critical** | Always checked | SQL & Data Safety, Race Conditions, Error Handling, Security, Input Validation |
| **Pass 2 — Informational** | Checked if time permits | Magic Numbers, Dead Code, Test Gaps, Performance, API Contract |

#### Suppressions

To reduce false positives, the following are explicitly suppressed:
- TODO/FIXME comments with issue tracker references
- Missing tests for trivial getters/setters
- Import ordering or formatting differences
- Variable naming matching existing project conventions
- Missing documentation on private functions
- Minor style preferences

#### Anti-Sycophancy Rules

Claude is instructed to be direct and decisive — no hedging with phrases like "you might want to consider" or "this could potentially cause". Every finding takes a clear position.

#### Actionability Classification

Each finding is classified by actionability:

| Classification | Meaning | Examples |
|----------------|---------|----------|
| **auto-fix** | A senior engineer would apply without discussion | Dead code, magic numbers, missing error wrapping |
| **needs-discussion** | Requires team input before fixing | Security decisions, API changes, behavioral changes |
| **architectural** | Needs a broader design conversation | Wrong abstraction, missing layer, significant refactor |

### Propose Workflow

1. **Repo Input**: The tool receives a GitHub repository reference (URL or `owner/repo`).
2. **Clone**: The repository is cloned locally with a partial clone filter.
3. **Cache Check**: The default branch HEAD SHA is fetched via `git ls-remote`. If a cached result exists for this SHA, it is reused.
4. **Claude Analysis**: Claude performs a deep codebase analysis covering architecture, code quality, feature gaps, DX, performance, security, testing, and CI/CD.
5. **Structuring**: A second Claude call converts the raw analysis into structured JSON proposals with priority, category, scope, and acceptance criteria.
6. **Output**: Proposals are rendered as Markdown (default), JSON, or GitHub issue templates.

### CLI Interface

#### Review (default command)

```bash
# Simple invocation with PR URL
planwerk-review https://github.com/owner/repo/pull/123

# Short form with owner/repo#number
planwerk-review owner/repo#123

# With explicit pattern directory
planwerk-review --patterns ./custom-patterns owner/repo#123

# Only output specific severity levels
planwerk-review --min-severity warning owner/repo#123

# Write output to file
planwerk-review owner/repo#123 > review.md
```

##### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--patterns` | Additional pattern directory | - |
| `--min-severity` | Minimum severity level for output (`info`, `warning`, `critical`, `blocking`) | `info` |
| `--no-repo-patterns` | Ignore repo-specific patterns | `false` |
| `--no-local-patterns` | Ignore local patterns from the tool | `false` |
| `--no-cache` | Ignore cache, force a fresh review | `false` |
| `--clear-cache` | Clear all cached reviews and exit | `false` |
| `--format` | Output format (`markdown`, `json`) | `markdown` |

#### Propose (subcommand)

```bash
# Analyze a repository and generate feature proposals
planwerk-review propose https://github.com/owner/repo

# Short form
planwerk-review propose owner/repo

# Output as JSON
planwerk-review propose --format json owner/repo

# Output as GitHub issue templates
planwerk-review propose --format issues owner/repo

# Force fresh analysis (ignore cache)
planwerk-review propose --no-cache owner/repo

# Write proposals to file
planwerk-review propose owner/repo > proposals.md
```

##### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--no-cache` | Ignore cache, force a fresh analysis | `false` |
| `--format` | Output format (`markdown`, `json`, `issues`) | `markdown` |

### Output Format

The generated Markdown report follows a fixed structure:

```markdown
# Review: owner/repo#123

> *Feature: Add user authentication*
> Reviewed by planwerk-review vX.Y.Z with Claude CLI

## BLOCKING (1)

### B-001: Hardcoded secrets in configuration
**File**: `config/auth.go:42`
**Pattern**: *Hardcoded values* (if detected by pattern)
**Actionability**: needs-discussion

**Problem**: API secret is hardcoded directly in the source code.

**Action Required**: Remove secret from code and provide it via
environment variable or secret manager.

---

## CRITICAL (2)

### C-001: SQL Injection in User Query
**File**: `db/users.go:87`
**Actionability**: needs-discussion

**Problem**: User input is used in SQL query without sanitization.

**Action Required**: Use prepared statements.
`db.Query("SELECT * FROM users WHERE id = ?", userID)`

### C-002: Missing error handling
**File**: `handlers/login.go:23`
**Actionability**: auto-fix

**Problem**: Error from `ValidateToken()` is ignored.

**Action Required**: Check error and return HTTP 401 on failure.

---

## WARNING (3)

### W-001: ...

---

## INFO (1)

### I-001: ...

---

## Summary

| Category  | Count |
|-----------|-------|
| BLOCKING  | 1     |
| CRITICAL  | 2     |
| WARNING   | 3     |
| INFO      | 1     |

**Recommendation**: PR should not be merged due to 1 BLOCKING and
2 CRITICAL findings until they are resolved.
```

#### Severity Levels

| Level | Meaning | Action |
|-------|---------|--------|
| **BLOCKING** | Fundamental architecture/security issues | PR must not be merged |
| **CRITICAL** | Bugs, security vulnerabilities, severe problems | Must be fixed before merge |
| **WARNING** | Code quality, potential issues | Should be fixed |
| **INFO** | Style questions, improvement suggestions | Optional, for information |

#### Actionability Levels

| Level | Meaning | Action |
|-------|---------|--------|
| **auto-fix** | A senior engineer would fix without discussion | Apply the suggested fix directly |
| **needs-discussion** | Requires team input before fixing | Discuss in PR comments or team sync |
| **architectural** | Fundamental design issue | Needs broader design conversation |

### Review Patterns

Review Patterns are structured rules that systematically improve the review. They codify knowledge from past reviews and make it reusable.

#### Pattern Sources (descending priority)

1. **Repo-specific Patterns** (`.planwerk/review_patterns/*.md` in the target repo)
   - Created and maintained by the development team (Planwerk) themselves
   - Contain repo-specific knowledge (e.g., "In this repo, all DB queries must go through the QueryBuilder")
   - Versioned with the repository

2. **General Patterns** (`patterns/` in the planwerk-review repository)
   - Created by planwerk-review and recommended for adoption
   - Contain universally applicable review knowledge (e.g., "Hardcoded values in matrix workflows")
   - Grow over time through insights from conducted reviews

#### Pattern Format

```markdown
# Review Pattern: <Pattern Name>

**Review-Area**: <architecture|security|quality|testing|workflow|...>
**Detection-Hint**: <Description of when/how this pattern should be detected>
**Severity**: <BLOCKING|CRITICAL|WARNING|INFO>
**Occurrences**: <Number of previous findings>

## What to check

<Detailed description of what to check>

## Why it matters

<Explanation of why this pattern is important>

## Examples from external reviews

### <ID> — <Source>
- **Feedback**: <Concrete feedback from an actual review>
- **What was missed**: <What was overlooked>
- **Fix**: <How it was fixed>
```

#### Knowledge Building

The tool systematically builds knowledge over time:

```
First Review           Subsequent Reviews       Mature System
────────────          ────────────────────      ─────────────
Claude /review   ──▶  Claude /review       ──▶  Claude /review
(no patterns)         + general patterns        + general patterns
                      + repo-specific           + repo-specific
      │               patterns                  patterns (many)
      ▼                     │                         │
Suggest new                 ▼                         ▼
patterns             Refine patterns            High-precision
                     + suggest new ones         reviews
```

**Knowledge building process:**

1. **After the first review**: The tool analyzes review results and suggests new general patterns that should be added to `patterns/`.
2. **For recurring findings**: When the same issue occurs across multiple repos, the `Occurrences` field is incremented and the pattern is refined.
3. **Repo-specific patterns**: The development team creates these themselves in `.planwerk/review_patterns/` based on their domain knowledge. planwerk-review picks them up automatically.

### Project Structure

```
planwerk-review/
├── .github/
│   └── workflows/
│       ├── ci.yml              # Test, Build, Vet on push/PR
│       ├── lint.yml            # golangci-lint
│       └── release.yml         # GoReleaser on tag push
├── cmd/
│   └── planwerk-review/
│       └── main.go             # CLI entrypoint (cobra): review + propose
├── internal/
│   ├── cache/
│   │   ├── cache.go            # SHA-based caching (review + propose)
│   │   └── cache_test.go
│   ├── cli/
│   │   └── cli.go              # Flag parsing, configuration
│   ├── github/
│   │   ├── pr.go               # Fetch PR data (gh CLI)
│   │   ├── pr_test.go
│   │   ├── repo.go             # Clone repo, fetch HEAD SHA
│   │   └── repo_test.go
│   ├── review/
│   │   └── reviewer.go         # Orchestration: PR → Claude → Report
│   ├── claude/
│   │   ├── claude.go           # Invoke Claude CLI for PR review
│   │   ├── claude_test.go
│   │   ├── propose.go          # Invoke Claude CLI for codebase analysis
│   │   └── propose_test.go
│   ├── patterns/
│   │   ├── loader.go           # Load patterns from directories
│   │   ├── loader_test.go
│   │   ├── pattern.go          # Pattern data structure + parsing
│   │   └── pattern_test.go
│   ├── propose/
│   │   ├── proposal.go         # Proposal data structure + categorization
│   │   ├── proposal_test.go
│   │   ├── proposer.go         # Orchestration: Repo → Claude → Proposals
│   │   ├── proposer_test.go
│   │   └── renderer.go         # Markdown/JSON/Issues output
│   └── report/
│       ├── finding.go          # Finding data structure
│       ├── renderer.go         # Markdown/JSON output
│       ├── renderer_test.go
│       ├── categorizer.go      # Severity categorization
│       └── categorizer_test.go
├── patterns/                   # General review patterns
│   └── hardcoded-matrix-values.md
├── go.mod
├── go.sum
├── .golangci.yml
├── .goreleaser.yml
└── README.md
```

### GitHub Workflows

#### CI (`ci.yml`)

- **Trigger**: Push to `main`, Pull Requests
- **Jobs**:
  - `test`: `go test ./...` on matrix (Ubuntu, macOS)
  - `build`: `go build ./cmd/planwerk-review/`
  - `vet`: `go vet ./...`

#### Lint (`lint.yml`)

- **Trigger**: Push to `main`, Pull Requests
- **Jobs**:
  - `lint`: `golangci-lint run`

#### Release (`release.yml`)

- **Trigger**: Tag push (`v*`)
- **Jobs**:
  - GoReleaser: Binaries for Linux/macOS/Windows (amd64, arm64)
  - GitHub Release with changelog

### Dependencies

- **Go 1.25.x** (as specified — Go 1.25 is not yet released, 1.24 as fallback if needed)
- **Claude CLI**: Must be installed and authenticated on the system (`claude` in PATH)
- **gh CLI**: Required for fetching PR metadata and checkout (`gh` in PATH)
- **git**: Required for cloning repositories

### Prerequisites

1. Go 1.25+ installed (or use release binary)
2. Claude CLI installed and authenticated (`claude` in PATH)
3. `gh` CLI installed and authenticated (`gh auth login`)
4. Access to the target repository (for checkout/clone)

### Design Decisions

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | **Claude CLI invocation** | Once for the entire PR | More efficient; Claude sees full context across files |
| 2 | **Pattern delivery** | Inline in the prompt before `/review` | Patterns are prepended to the `/review` command so Claude considers them during its built-in review |
| 3 | **Result parsing** | Second Claude call for structuring | `/review` returns unstructured text; a second `claude -p` call converts it to JSON matching the `ReviewResult` schema |
| 4 | **Authentication** | `gh auth` | Simplest setup; leverages existing developer workflow |
| 5 | **Review caching** | Based on PR HEAD SHA | Avoids repeated reviews of unchanged PR state |
| 6 | **Propose: two-step Claude** | Analysis → Structure | First call explores codebase freely; second call converts to strict JSON schema |
| 7 | **Propose: cache invalidation** | Based on default branch HEAD SHA | Cache key includes `git ls-remote` HEAD, so proposals refresh when the repo changes |
| 8 | **Propose: output formats** | Markdown, JSON, Issues | Markdown for reading, JSON for automation, Issues for direct GitHub issue creation |
| 9 | **Review prompt structure** | Multi-section structured prompt | Persona framing, scope analysis, two-pass checklist, suppressions, and anti-sycophancy rules produce higher-quality, more consistent reviews (inspired by [gstack](https://github.com/garrytan/gstack)) |
| 10 | **Actionability classification** | auto-fix / needs-discussion / architectural | Helps teams prioritize which findings to address immediately vs. discuss first |
| 11 | **Scope drift detection** | PR title + body analyzed before code review | Catches scope creep and missing requirements — often the most valuable review feedback |

### Future Extensions

- **Direct PR review posting**: Post results directly as a GitHub PR review comment (not just stdout)
- **Pattern suggestions**: Automatically generate new pattern suggestions after a review
- **Diff-based re-review**: Only check new changes since the last review
- **Multi-reviewer**: Integrate other review tools alongside Claude
- **Dashboard**: Overview of review statistics per repository
- **GitHub Action**: Publish as a GitHub Action for automated PR reviews
- **Propose: auto-create issues**: Directly create GitHub issues from proposals via `gh issue create`
- **Propose: incremental analysis**: Track which proposals have been addressed across runs
