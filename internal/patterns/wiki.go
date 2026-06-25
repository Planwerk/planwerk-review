package patterns

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Wiki convention defaults. A target repo's GitHub Wiki holds review patterns
// under review_patterns/ (in the same format as .planwerk/review_patterns/) and
// free-form project memory pages under memory/.
const (
	defaultWikiPatternsSubpath = "review_patterns"
	defaultWikiMemorySubpath   = "memory"
)

// maxMemoryBytes caps the concatenated project-memory block read into a prompt.
// A wiki is human-editable through the web UI, so an over-long (or runaway)
// memory tree would balloon the prompt and its API cost; pages past the cap are
// skipped with a warning. Mirrors glossary.maxGlossaryBytes.
const maxMemoryBytes = 64 * 1024

// wikiRevParseTimeout bounds the `git rev-parse HEAD` that resolves the wiki to
// a concrete commit for the report.
const wikiRevParseTimeout = 30 * time.Second

// WikiOptions configures whether and how a target repo's GitHub Wiki is used as
// a knowledge source. The zero value is disabled, so a caller that does not opt
// in resolves no wiki. The CLI leaves Enabled off by default: a wiki is a
// separate permission surface (often world-editable, never gated by branch
// protection or PR review), so its content is fed to the agent only on an
// explicit per-repo opt-in (--wiki).
type WikiOptions struct {
	// Enabled turns wiki resolution on. When false ResolveWiki returns the zero
	// ResolvedWiki without touching the network.
	Enabled bool
	// Repo overrides the wiki source as "owner/repo"; empty derives it from the
	// resolved target repository. The wiki cloned is always <repo>.wiki.git.
	Repo string
	// Ref pins the wiki to a branch, tag, or commit; empty uses the wiki's
	// default branch.
	Ref string
}

// ResolvedWiki is the outcome of resolving a target repo's wiki. The zero value
// (every field empty) means "no wiki" — disabled, uninitialized, offline, or
// failed — and every consumer treats it as such, so a run without a usable wiki
// proceeds unchanged.
type ResolvedWiki struct {
	// Repo is the "owner/repo" the wiki was resolved for, for the report header.
	Repo string
	// CommitSHA is the wiki's resolved HEAD commit, recorded in the report so a
	// review is reproducible against a fixed wiki state. Empty when unresolved.
	CommitSHA string
	// Dir is the local clone root of the resolved wiki, so a consumer that needs
	// per-entry access (e.g. sync, enumerating review_patterns/ and memory/ as
	// individual pages) can walk it. Empty when no wiki was resolved. PatternsDir
	// is a subdirectory of it; Memory is read from its memory/ subdir.
	Dir string
	// PatternsDir is the local directory the wiki's review patterns live in, to
	// feed ResolveOptions.Wiki. Empty when the wiki has no patterns subdir.
	PatternsDir string
	// Memory is the concatenated project-memory block, injected into the
	// analysis and planning prompts. Empty when the wiki has no memory.
	Memory string
}

// ResolveWiki materializes the target repo's GitHub Wiki and returns its review
// patterns directory, concatenated project memory, and resolved commit. It is
// best-effort, mirroring glossary.LoadBody: when disabled, or when the wiki is
// uninitialized, offline, or otherwise unresolvable, it logs and returns the
// zero ResolvedWiki so the caller runs unchanged rather than failing.
//
// The wiki is cloned (and refreshed by TTL) through the same remote-cache
// machinery as --patterns remote sources, via the wiki: URI shorthand, so it
// reuses caching, locking, and token authentication. It is resolved at run
// start and pinned to a concrete commit so a review does not drift with a moving
// wiki between runs.
func ResolveWiki(owner, name string, wopts WikiOptions, ropts RemoteOptions) ResolvedWiki {
	if !wopts.Enabled {
		return ResolvedWiki{}
	}

	repo := wopts.Repo
	if repo == "" {
		repo = owner + "/" + name
	}

	// Build the wiki: URI for the clone root (no subpath, so the cache holds the
	// whole wiki and the subpaths below are joined locally).
	uri := prefixWiki + repo
	if wopts.Ref != "" {
		uri += "@" + wopts.Ref
	}

	dir, err := ResolveRemote(uri, ropts)
	if err != nil {
		slog.Warn("could not resolve target repo wiki, proceeding without it", "repo", repo, "err", err)
		return ResolvedWiki{}
	}

	patternsDir := filepath.Join(dir, defaultWikiPatternsSubpath)
	if info, err := os.Stat(patternsDir); err != nil || !info.IsDir() {
		patternsDir = "" // no review-patterns directory in this wiki
	}

	resolved := ResolvedWiki{
		Repo:        repo,
		CommitSHA:   wikiHeadSHA(dir),
		Dir:         dir,
		PatternsDir: patternsDir,
		Memory:      LoadMemory(filepath.Join(dir, defaultWikiMemorySubpath)),
	}
	slog.Info("resolved target repo wiki",
		"repo", repo,
		"commit", resolved.CommitSHA,
		"has_patterns", resolved.PatternsDir != "",
		"memory_bytes", len(resolved.Memory))
	return resolved
}

// wikiHeadSHA returns the wiki clone's HEAD commit, or "" when it cannot be
// resolved (e.g. an empty repo or a git failure). A missing SHA only costs the
// reproducibility note in the report, so it is non-fatal.
func wikiHeadSHA(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), wikiRevParseTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		slog.Warn("could not resolve wiki commit", "dir", dir, "err", err)
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LoadMemory reads every *.md page under dir (in sorted filename order) and
// concatenates them into one project-memory block, each page prefixed with a
// "### <name>" header derived from its filename. Every .md page under the memory
// subdir is project memory by convention; the patterns the wiki author keeps for
// review live under the separate review_patterns subdir. The total is capped at
// maxMemoryBytes; once the cap is reached the remaining pages are skipped with a
// warning. A missing, unreadable, or empty directory yields "".
func LoadMemory(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "" // absent or unreadable memory dir: no memory
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		// Skip directories, symlinks, and non-markdown entries. The symlink
		// guard is load-bearing: a wiki is world-editable, so a *.md symlink
		// pointing at e.g. ~/.aws/credentials or ~/.ssh/id_rsa would otherwise be
		// followed by the read below and its target concatenated into the prompt.
		// os.DirEntry reports the entry's own type without following it, so a
		// symlink is rejected here before anything opens its target.
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var sb strings.Builder
	for _, n := range names {
		body, err := readMemoryPage(filepath.Join(dir, n))
		if err != nil {
			// An oversized page is skipped with a warning; an unreadable one is
			// skipped silently. Either way a single bad page must not suppress the
			// legitimate pages after it (hence continue, not break).
			if errors.Is(err, errMemoryPageTooLarge) {
				slog.Warn("project memory page exceeds size cap; skipping", "dir", dir, "page", n, "cap", maxMemoryBytes)
			}
			continue
		}
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		entry := "### " + strings.TrimSuffix(n, ".md") + "\n\n" + body + "\n"
		if sb.Len()+len(entry) > maxMemoryBytes {
			slog.Warn("project memory exceeds total size cap; skipping page", "dir", dir, "page", n, "cap", maxMemoryBytes)
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(entry)
	}
	return strings.TrimSpace(sb.String())
}

// errMemoryPageTooLarge marks a memory page that exceeds maxMemoryBytes on its
// own, so LoadMemory can warn about it specifically rather than treating it like
// an unreadable file.
var errMemoryPageTooLarge = errors.New("memory page exceeds size cap")

// readMemoryPage reads a single memory page through a bounded read: it opens the
// file and pulls at most maxMemoryBytes+1 bytes via an io.LimitReader, so a
// runaway (multi-gigabyte) page never allocates more than the cap before it is
// rejected. It returns errMemoryPageTooLarge when the page is larger than
// maxMemoryBytes, so the caller skips the whole page rather than truncating it
// mid-content. Reading the cap into a []byte first (os.ReadFile) would allocate
// the entire file before any size check could run.
func readMemoryPage(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxMemoryBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxMemoryBytes {
		return "", errMemoryPageTooLarge
	}
	return string(data), nil
}
