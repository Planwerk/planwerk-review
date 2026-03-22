# planwerk-review

AI-powered code review tool for GitHub Pull Requests. Uses Claude CLI (`/review`) to automatically analyze PR changes and produces structured, categorized review results as Markdown output.

## Concept

### Overview

```
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
```

### Workflow

1. **PR Input**: The tool receives a GitHub PR as input (URL or `owner/repo#number`).
2. **Checkout**: The PR is checked out locally (diff between base and head).
3. **Load Review Patterns**: Patterns are loaded from two sources:
   - `patterns/` in the planwerk-review repository (general patterns)
   - `.planwerk/review_patterns/` in the target repository (repo-specific patterns)
4. **Claude CLI Review**: `claude /review` is executed for all changed files in the PR. Loaded review patterns are passed as additional context to Claude.
5. **Result Aggregation**: Review results are collected, deduplicated, and categorized by severity.
6. **Markdown Output**: A structured report is written to `stdout`.

### CLI Interface

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

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--patterns` | Additional pattern directory | - |
| `--min-severity` | Minimum severity level for output (`info`, `warning`, `critical`, `blocking`) | `info` |
| `--no-repo-patterns` | Ignore repo-specific patterns | `false` |
| `--no-local-patterns` | Ignore local patterns from the tool | `false` |
| `--format` | Output format (`markdown`, `json`) | `markdown` |

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

**Problem**: API secret is hardcoded directly in the source code.

**Action Required**: Remove secret from code and provide it via
environment variable or secret manager.

---

## CRITICAL (2)

### C-001: SQL Injection in User Query
**File**: `db/users.go:87`

**Problem**: User input is used in SQL query without sanitization.

**Action Required**: Use prepared statements.
`db.Query("SELECT * FROM users WHERE id = ?", userID)`

### C-002: Missing error handling
**File**: `handlers/login.go:23`

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
│       └── main.go             # CLI entrypoint (cobra)
├── internal/
│   ├── cli/
│   │   └── cli.go              # Flag parsing, configuration
│   ├── github/
│   │   └── pr.go               # Fetch PR data (gh CLI or API)
│   ├── review/
│   │   ├── reviewer.go         # Orchestration: PR → Claude → Report
│   │   └── reviewer_test.go
│   ├── claude/
│   │   ├── claude.go           # Invoke Claude CLI (/review)
│   │   └── claude_test.go
│   ├── patterns/
│   │   ├── loader.go           # Load patterns from directories
│   │   ├── loader_test.go
│   │   ├── pattern.go          # Pattern data structure + parsing
│   │   └── pattern_test.go
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
- **Claude CLI**: Must be installed and authenticated on the system
- **gh CLI** (optional): For fetching PR data, alternatively GitHub API directly

### Prerequisites

1. Go 1.25+ installed (or use release binary)
2. Claude CLI installed and authenticated (`claude` in PATH)
3. Access to the target repository (for checkout)

### Design Decisions

| # | Question | Decision | Rationale |
|---|----------|----------|-----------|
| 1 | **Claude CLI invocation** | Once for the entire PR | More efficient; Claude sees full context across files |
| 2 | **Pattern delivery** | Inline in the prompt before `/review` | Patterns are prepended to the `/review` command so Claude considers them during its built-in review |
| 3 | **Result parsing** | Second Claude call for structuring | `/review` returns unstructured text; a second `claude -p` call converts it to JSON matching the `ReviewResult` schema |
| 4 | **Authentication** | `gh auth` | Simplest setup; leverages existing developer workflow |
| 5 | **Caching** | Yes, based on PR HEAD SHA | Avoids repeated reviews of unchanged PR state |

### Future Extensions

- **Direct PR review posting**: Post results directly as a GitHub PR review comment (not just stdout)
- **Pattern suggestions**: Automatically generate new pattern suggestions after a review
- **Diff-based re-review**: Only check new changes since the last review
- **Multi-reviewer**: Integrate other review tools alongside Claude
- **Dashboard**: Overview of review statistics per repository
- **GitHub Action**: Publish as a GitHub Action for automated PR reviews
