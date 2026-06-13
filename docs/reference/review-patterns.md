# Review patterns

Review patterns are structured rules that systematically improve the review.
They codify knowledge from past reviews and make it reusable. This page is the
reference for how patterns are sourced, prioritized, and formatted; for the task
of authoring one, see [Write your own review patterns](/how-to/write-review-patterns).

## Pattern Sources

Patterns are resolved from up to five tiers, listed here from lowest to highest
priority. Later tiers override earlier ones by pattern name, so the more
specific source wins on a name collision:

1. **Embedded catalog** — a compile-time copy of `internal/patterns/patterns/`
   baked into the binary with `//go:embed`. It is always present, so every
   install method (`go install`, raw `go build`, release archives, OS packages)
   produces a self-contained binary that loads the full catalog with no external
   files. This is the lowest-priority source; any on-disk source below overrides
   an embedded pattern of the same name.

2. **Bundled on-disk catalog** (`<binDir>/../patterns`) — an optional copy next
   to the installed binary. Lets a distribution ship the catalog as a separately
   updatable data file (so pattern fixes can land without rebuilding the binary);
   when present it overrides the embedded copy.

3. **Working-directory catalog** (`./patterns`) — picked up when running from a
   planwerk-review checkout during development of the tool itself.

4. **Repo-specific patterns** (`.planwerk/review_patterns/*.md` in the target repo)
   - Created and maintained by the development team (Planwerk) themselves
   - Contain repo-specific knowledge (e.g., "In this repo, all DB queries must go through the QueryBuilder")
   - Versioned with the repository
   - Suppressed independently by `--no-repo-patterns`

5. **Explicit / remote patterns** (passed via `--patterns <URI>` or the config file)
   - Local directories or remote URIs — lets a team maintain a single, shared pattern catalog in a separate repository instead of vendoring it into every consuming repo
   - Remote sources are cloned into a per-user cache on first use and refreshed by TTL
   - Highest priority: override every tier above on a name collision
   - See [Remote Pattern Sources](#remote-pattern-sources) below for URI forms, caching, and authentication

`--no-local-patterns` suppresses the first three tiers — the embedded catalog
and both on-disk tool copies (`<binDir>/../patterns` and `./patterns`) — leaving
only repo-specific and `--patterns` sources. `--no-repo-patterns` independently
drops tier 4.

## Remote Pattern Sources

Any value passed to `--patterns` (or the `patterns:` array in
`.planwerk/config.yaml`) may be either a local directory or a remote URI. Two
URI forms are accepted:

```text
github:owner/repo[/subpath][@ref]              # GitHub shorthand
git+https://host.example/group/repo.git[#ref[:subpath]]   # any git host
```

Examples:

```bash
# Default branch of a GitHub repo
planwerk-review --patterns github:planwerk/patterns owner/repo#123

# Pinned tag, sub-directory inside the repo
planwerk-review --patterns github:planwerk/patterns/security@v1.2.3 owner/repo#123

# Generic git URL with ref + subpath (separator: ":" inside the fragment)
planwerk-review --patterns git+https://gitlab.example.com/team/p.git#main:patterns/web owner/repo#123

# Mix local + remote, in priority order
planwerk-review --patterns ./local-overrides --patterns github:planwerk/patterns owner/repo#123
```

Anything that doesn't match `github:` or `git+http(s)://` is treated as a local
path, so existing usage is unchanged.

**Caching.** Remote sources are cloned into
`<UserCacheDir>/planwerk-review/patterns/<hash>/repo/` (typically
`~/.cache/planwerk-review/patterns/…` on Linux,
`~/Library/Caches/planwerk-review/patterns/…` on macOS). A neighbouring
`meta.json` records when the clone was last refreshed. The cache is keyed by the
URI (excluding the subpath), so two URIs that differ only in their subpath share
the same checkout.

**Refresh TTL.** Cached clones are refreshed when older than
`--remote-patterns-ttl` (default `24h`, env: `PLANWERK_REMOTE_PATTERNS_TTL`).
Setting `--remote-patterns-ttl 0` disables refresh entirely — once cached, the
clone is reused indefinitely (useful for offline / air-gapped environments). On
refresh the existing checkout is removed and re-cloned; this keeps the cache
logic simple and is cheap because pattern repos are small.

**Authentication.**

| Form | How auth works |
|------|----------------|
| `github:owner/repo` | Cloned via `gh repo clone`, which uses your `gh auth login` credentials or the `GH_TOKEN` env var. Private GitHub repos work transparently if you can already access them with `gh`. |
| `git+https://…` | Cloned via plain `git clone`. Standard git credential helpers (`~/.git-credentials`, `git config credential.helper`) apply. For env-var-based auth, embed the token directly in the URI using shell-style `${VAR}` expansion: `git+https://oauth2:${MY_TOKEN}@gitlab.example.com/team/p.git`. The expansion runs before `git clone` is invoked. |

## Prompt Budget

By default, all loaded patterns are injected into the prompt without truncation
(`--max-patterns 0`, env: `PLANWERK_MAX_PATTERNS`). To cap pattern injection —
e.g. to keep prompts within Claude's context window — set `--max-patterns` to a
positive integer. When more patterns are loaded than the budget allows, the tool
keeps the highest-priority patterns by severity (`BLOCKING` > `CRITICAL` >
`WARNING` > `INFO`) and prints a warning to stderr.

## Pattern Format

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
