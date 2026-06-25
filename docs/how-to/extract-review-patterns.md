# Extract review patterns from the wiki

A target repository's [GitHub Wiki](/reference/review-patterns#github-wiki) is a
fast-moving, world-editable place to draft and iterate on review patterns. Once
a wiki pattern proves itself, `extract` anchors it into a committed, reproducible
location — either the repo's own `.planwerk/review_patterns/` (PR-gated and
code-coupled) or `planwerk-agent`'s bundled catalog (so every project benefits).

The command is mechanical: it never calls Claude. It reads the wiki's
`review_patterns/` directory, lets you choose which entries to anchor, and writes
the selected files.

## Promote a wiki pattern into the repo (PR)

The default mode writes the selected patterns into the target repo's
`.planwerk/review_patterns/` and opens a pull request:

```bash
planwerk-agent extract owner/repo
```

You are prompted for each wiki pattern (`y/N/q`). To skip the prompt, take every
pattern with `--all`, or name specific patterns by their filename stem:

```bash
planwerk-agent extract owner/repo --all
planwerk-agent extract owner/repo --pattern no-raw-sql --pattern bounded-retries
```

Review the resulting PR like any other change — the patterns become
committed, branch-protected, and reviewable, in contrast to the wiki where
anyone can edit them.

## Write directly into the working tree

With `--local`, the patterns are written straight into the current working
tree's `.planwerk/review_patterns/` and no PR is opened. The repository
reference is inferred from the `origin` remote when omitted:

```bash
planwerk-agent extract --local --all
```

A dirty working tree prompts for confirmation first; pass `--force` to skip it.
See [Use local mode](/how-to/use-local-mode) for the shared `--local` behavior.

## Contribute a pattern to the bundled catalog

With `--to-catalog`, the patterns are anchored into this `planwerk-agent`
checkout's bundled review catalog (`internal/patterns/patterns/review/`), with
each pattern's frontmatter `**Category**:` normalized to `review` so it loads as
a first-class [review pattern](/reference/review-patterns#pattern-categories).
This is the maintainer/contribution path and must be run from a
`planwerk-agent` checkout:

```bash
planwerk-agent extract owner/repo --all --to-catalog
```

The selected files land under `internal/patterns/patterns/review/`; commit them
and open a PR against `planwerk-agent` so the patterns ship to every project.

## Selecting which patterns to anchor

| Flag | Effect |
|------|--------|
| _(none)_ | Prompt for each pattern interactively (`y/N/q`) |
| `--all` | Take every wiki review pattern |
| `--pattern <stem>` | Take only the named pattern(s), by filename stem (repeatable) |

A non-interactive run (no TTY) requires `--all` or `--pattern`. The wiki is an
untrusted, world-editable source, so an automated run (CI, cron) refuses to
extract — and, in the default mode, push into a PR — every pattern without an
explicit choice, rather than failing open on whatever the wiki currently holds.

## Overwriting an existing pattern

The destination filename is the wiki pattern's stem, so `--local` and
`--to-catalog` refuse to write when a file of that name already exists — a wiki
author cannot silently clobber a trusted repo or catalog pattern by naming a
pattern to collide with it. Pass `--overwrite` to replace it on purpose; the
report marks each file that overwrote an existing one.

## See also

- [Write your own review patterns](/how-to/write-review-patterns) — author
  patterns directly in `.planwerk/review_patterns/`.
- [Use the GitHub Wiki](/how-to/use-the-github-wiki) — enable and populate the
  wiki knowledge source `extract` reads from.
