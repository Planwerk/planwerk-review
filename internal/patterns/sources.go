package patterns

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveOptions configures Resolve. The flags mirror the --no-local-patterns
// and --no-repo-patterns toggles the eight pattern-loading subcommands expose.
type ResolveOptions struct {
	// NoLocal suppresses the planwerk-agent-bundled on-disk catalog.
	NoLocal bool
	// NoRepo suppresses the target repo's .planwerk/review_patterns directory.
	NoRepo bool
	// RepoDir is the target repository checkout root. It is only consulted
	// when NoRepo is false.
	RepoDir string
	// Wiki is a resolved local directory holding the target repo's GitHub Wiki
	// review patterns (see ResolveWiki). Empty means no wiki patterns. It sits
	// below the repo's .planwerk/review_patterns and below the explicit Extra
	// dirs, so the repo's committed (reviewed, branch-protected) patterns override
	// the world-editable wiki on a name collision, while an operator's --patterns
	// still overrides both.
	Wiki string
	// Extra are explicit --patterns directories supplied by the caller. They
	// have the highest priority and are always appended.
	Extra []string
}

// Resolve assembles the ordered list of on-disk pattern directories to load,
// applying the precedence the eight subcommands share: the planwerk-agent
// bundled local catalog (lowest priority), then the target repo's GitHub Wiki
// review patterns, then the target repo's .planwerk/review_patterns directory,
// then any explicit --patterns directories (highest priority). The wiki sits
// below the committed repo patterns on purpose: the wiki is world-editable and
// unreviewed, so a repo's committed (and branch-protected) patterns must win
// over it on a name collision. The NoLocal and NoRepo toggles drop the bundled
// local and repo groups respectively; the wiki slot is dropped by passing an
// empty Wiki.
//
// Resolve is the single source of truth for this precedence order; callers
// must not re-derive it. The binary's embedded catalog is layered in
// separately by LoadFilteredWithOptions (see LoadOptions.NoEmbedded), not
// here. The error return leaves room for future fallible sources (e.g.
// XDG_DATA_DIRS); today Resolve never returns a non-nil error.
func Resolve(opts ResolveOptions) ([]string, error) {
	var dirs []string
	if dir := LocalPatternDir(opts.NoLocal); dir != "" {
		dirs = append(dirs, dir)
	}
	if opts.Wiki != "" {
		dirs = append(dirs, opts.Wiki)
	}
	if dir := RepoPatternDir(opts.NoRepo, opts.RepoDir); dir != "" {
		dirs = append(dirs, dir)
	}
	dirs = append(dirs, opts.Extra...)
	return dirs, nil
}

// LocalPatternDir returns the planwerk-agent-bundled on-disk pattern
// directory, or "" when noLocal is set or no candidate exists. It prefers the
// directory next to the executable (../patterns, the layout shipped before the
// catalog was embedded) and falls back to ./patterns relative to the working
// directory for development checkouts. The bare-prompt catalog builder uses
// this same root to map a pattern's FilePath back to its canonical URL.
func LocalPatternDir(noLocal bool) string {
	if noLocal {
		return ""
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "..", "patterns")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if info, err := os.Stat("patterns"); err == nil && info.IsDir() {
		return "patterns"
	}
	return ""
}

// RepoPatternDir returns the target repo's .planwerk/review_patterns
// directory, or "" when noRepo is set or the directory does not exist. The
// bare-prompt catalog builder uses this root to emit "read this from your
// checkout" entries instead of remote URLs.
func RepoPatternDir(noRepo bool, repoDir string) string {
	if noRepo {
		return ""
	}
	candidate := filepath.Join(repoDir, ".planwerk", "review_patterns")
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

// resolveSources turns each entry into a local directory path: remote URIs
// are materialized into the cache via ResolveRemote, local paths pass
// through unchanged.
func resolveSources(opts RemoteOptions, sources []string) ([]string, error) {
	dirs := make([]string, 0, len(sources))
	for _, src := range sources {
		if IsRemote(src) {
			d, err := ResolveRemote(src, opts)
			if err != nil {
				return nil, fmt.Errorf("resolving remote pattern source %q: %w", src, err)
			}
			dirs = append(dirs, d)
			continue
		}
		dirs = append(dirs, src)
	}
	return dirs, nil
}

// loadOrderedSources resolves opts and sources into parsed-pattern groups in
// ascending priority order (lowest first): the embedded catalog (unless
// opts.NoEmbedded) is the lowest-priority group, followed by one group per
// explicit on-disk/remote source in slice order. The caller dedups across the
// groups by Pattern.Name — later groups win — and applies tag filtering.
func loadOrderedSources(opts LoadOptions, sources []string) ([][]Pattern, error) {
	var groups [][]Pattern

	if !opts.NoEmbedded {
		embedded, err := loadEmbedded()
		if err != nil {
			return nil, fmt.Errorf("loading embedded patterns: %w", err)
		}
		groups = append(groups, embedded)
	}

	dirs, err := resolveSources(opts.Remote, sources)
	if err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		pats, err := loadDir(dir)
		if err != nil {
			return nil, err
		}
		groups = append(groups, pats)
	}

	return groups, nil
}
