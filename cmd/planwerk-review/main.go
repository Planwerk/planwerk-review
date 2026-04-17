package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/planwerk/planwerk-review/internal/audit"
	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/claude"
	"github.com/planwerk/planwerk-review/internal/cli"
	"github.com/planwerk/planwerk-review/internal/logging"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/propose"
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/review"
)

// envMaxPatterns is the environment variable used to override the default
// maximum number of review patterns injected into the prompt.
const envMaxPatterns = "PLANWERK_MAX_PATTERNS"

// Output format identifiers accepted by the --format flag.
const (
	formatMarkdown = "markdown"
	formatJSON     = "json"
	formatIssues   = "issues"
)

// resolveMaxPatterns returns the effective max-patterns limit. Precedence:
// explicit CLI flag, then .planwerk/config.yaml, then PLANWERK_MAX_PATTERNS,
// then the compiled-in default. A value of 0 or negative disables truncation.
func resolveMaxPatterns(flagValue int, flagSet bool, fileValue *int) (int, error) {
	if flagSet {
		return flagValue, nil
	}
	if fileValue != nil {
		return *fileValue, nil
	}
	if raw, ok := os.LookupEnv(envMaxPatterns); ok && raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return 0, fmt.Errorf("invalid %s=%q: %w", envMaxPatterns, raw, err)
		}
		return v, nil
	}
	return patterns.DefaultMaxPatternsInPrompt, nil
}

// devVersion is the placeholder version string used when no release version
// has been injected via ldflags. Triggers the unreleased-build warning.
const devVersion = "dev"

var version = devVersion

// buildInfo holds resolved version and build metadata, populated either from
// ldflags (main.version) or from runtime/debug.ReadBuildInfo.
type buildInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	IsDev     bool
}

// resolveBuildInfo returns build metadata, preferring the ldflags-injected
// version and falling back to debug.ReadBuildInfo when it is unset.
func resolveBuildInfo(ldflagsVersion string) buildInfo {
	bi := buildInfo{Version: ldflagsVersion}

	if info, ok := debug.ReadBuildInfo(); ok {
		bi.GoVersion = info.GoVersion
		if bi.Version == "" || bi.Version == devVersion {
			if v := info.Main.Version; v != "" && v != "(devel)" {
				bi.Version = v
			}
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				bi.Commit = s.Value
			case "vcs.time":
				bi.BuildDate = s.Value
			}
		}
	}

	if bi.Version == "" {
		bi.Version = devVersion
	}
	bi.IsDev = bi.Version == devVersion
	return bi
}

// writeVersion renders the version line, optional verbose build details, and
// a warning when this is an unreleased development build.
func writeVersion(w io.Writer, bi buildInfo, verbose bool) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "planwerk-review version %s\n", bi.Version)
	if verbose {
		if bi.Commit != "" {
			fmt.Fprintf(&sb, "commit: %s\n", bi.Commit)
		}
		if bi.BuildDate != "" {
			fmt.Fprintf(&sb, "built: %s\n", bi.BuildDate)
		}
		if bi.GoVersion != "" {
			fmt.Fprintf(&sb, "go: %s\n", bi.GoVersion)
		}
	}
	if bi.IsDev {
		sb.WriteString("warning: unreleased development build — version metadata unavailable\n")
	}
	_, _ = io.WriteString(w, sb.String())
}

func main() {
	var cfg cli.Config
	var minSeverity string
	var showVersion, verbose bool
	var logFormat string
	var fileCfg cli.FileConfig

	rootCmd := &cobra.Command{
		Use:   "planwerk-review <pr-ref>",
		Short: "AI-powered code review for GitHub Pull Requests",
		Long: `planwerk-review uses Claude Code to analyze GitHub PR changes and produces
structured, categorized review results as Markdown or JSON output.

PR reference can be a URL (https://github.com/owner/repo/pull/123)
or short form (owner/repo#123).`,
		Args: cobra.RangeArgs(0, 1),
		// Errors are reported via slog in main() so they honor --log-format;
		// silencing cobra's own error/usage print avoids duplicate output.
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			format, err := logging.ParseFormat(logFormat)
			if err != nil {
				return err
			}
			if err := logging.Init(logging.Options{
				Writer:  os.Stderr,
				Format:  format,
				Verbose: verbose,
			}); err != nil {
				return err
			}
			loaded, _, err := cli.LoadFileConfig(cli.DefaultConfigPath)
			if err != nil {
				return err
			}
			fileCfg = loaded
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				writeVersion(cmd.OutOrStdout(), resolveBuildInfo(version), verbose)
				return nil
			}
			if cfg.ClearCache {
				return cache.Clear()
			}

			if len(args) == 0 {
				return fmt.Errorf("requires a PR reference argument")
			}
			cfg.PRRef = args[0]

			fileCfg.ApplyReview(&cfg, &minSeverity, cmd.Flags().Changed)

			maxPatterns, err := resolveMaxPatterns(cfg.MaxPatterns, cmd.Flags().Changed("max-patterns"), fileCfg.Review.MaxPatterns)
			if err != nil {
				return err
			}
			cfg.MaxPatterns = maxPatterns

			if minSeverity != "" {
				sev, err := report.ParseSeverity(minSeverity)
				if err != nil {
					return err
				}
				cfg.MinSeverity = sev
			} else {
				cfg.MinSeverity = report.SeverityInfo
			}

			switch cfg.Format {
			case formatMarkdown, formatJSON:
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json", cfg.Format)
			}

			if cfg.PostReview && cfg.Format == formatJSON {
				return fmt.Errorf("--post-review cannot be used with --format json")
			}

			// --inline implies --post-review
			if cfg.InlineReview {
				cfg.PostReview = true
			}
			if cfg.InlineReview && cfg.Format == formatJSON {
				return fmt.Errorf("--inline cannot be used with --format json")
			}

			opts := cfg.ToReviewOptions(version)
			return review.Run(os.Stdout, opts)
		},
	}

	persistent := rootCmd.PersistentFlags()
	persistent.BoolVarP(&verbose, "verbose", "v", false, "Enable debug-level logging (and verbose build info with --version)")
	persistent.StringVar(&logFormat, "log-format", "text", "Log output format (text, json)")

	flags := rootCmd.Flags()
	flags.StringSliceVar(&cfg.PatternDirs, "patterns", nil, "Additional pattern directories")
	flags.StringVar(&minSeverity, "min-severity", "", "Minimum severity level (info, warning, critical, blocking)")
	flags.BoolVar(&cfg.NoRepoPatterns, "no-repo-patterns", false, "Ignore repo-specific patterns")
	flags.BoolVar(&cfg.NoLocalPatterns, "no-local-patterns", false, "Ignore local patterns from the tool")
	flags.BoolVar(&cfg.NoCache, "no-cache", false, "Ignore cache, force a fresh review")
	flags.BoolVar(&cfg.ClearCache, "clear-cache", false, "Clear all cached reviews and exit")
	flags.StringVar(&cfg.Format, "format", "markdown", "Output format (markdown, json)")
	flags.BoolVar(&cfg.PostReview, "post-review", false, "Post the review as a comment on the PR")
	flags.BoolVar(&cfg.InlineReview, "inline", false, "Post review with inline comments using GitHub Review API (implies --post-review)")
	flags.BoolVar(&cfg.Thorough, "thorough", false, "Run additional adversarial review pass")
	flags.BoolVar(&cfg.CoverageMap, "coverage-map", false, "Generate test coverage map for changed functions")
	flags.IntVar(&cfg.MaxPatterns, "max-patterns", patterns.DefaultMaxPatternsInPrompt, "Max review patterns injected into the prompt (<=0 disables truncation, env: "+envMaxPatterns+")")
	flags.IntVar(&cfg.MaxFindings, "max-findings", 0, "Cap on findings returned (<=0 disables cap)")
	flags.BoolVar(&showVersion, "version", false, "Show version information and exit")

	// propose subcommand
	var proposeCfg cli.ProposeConfig

	proposeCmd := &cobra.Command{
		Use:   "propose <repo-ref>",
		Short: "Analyze a codebase and generate feature proposals",
		Long: `Analyze a GitHub repository in depth and generate concrete, actionable
feature proposals as structured Markdown suitable for GitHub issues.

Repository reference can be a URL (https://github.com/owner/repo)
or short form (owner/repo).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proposeCfg.RepoRef = args[0]

			fileCfg.ApplyPropose(&proposeCfg, cmd.Flags().Changed)

			maxPatterns, err := resolveMaxPatterns(proposeCfg.MaxPatterns, cmd.Flags().Changed("max-patterns"), fileCfg.Propose.MaxPatterns)
			if err != nil {
				return err
			}
			proposeCfg.MaxPatterns = maxPatterns

			switch proposeCfg.Format {
			case formatMarkdown, formatJSON, formatIssues:
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json, issues", proposeCfg.Format)
			}

			opts := proposeCfg.ToProposeOptions(version)
			return propose.Run(os.Stdout, opts, claude.Propose)
		},
	}

	proposeFlags := proposeCmd.Flags()
	proposeFlags.StringSliceVar(&proposeCfg.PatternDirs, "patterns", nil, "Additional pattern directories")
	proposeFlags.BoolVar(&proposeCfg.NoRepoPatterns, "no-repo-patterns", false, "Ignore repo-specific patterns")
	proposeFlags.BoolVar(&proposeCfg.NoLocalPatterns, "no-local-patterns", false, "Ignore local patterns from the tool")
	proposeFlags.BoolVar(&proposeCfg.NoCache, "no-cache", false, "Ignore cache, force a fresh analysis")
	proposeFlags.StringVar(&proposeCfg.Format, "format", "markdown", "Output format (markdown, json, issues)")
	proposeFlags.IntVar(&proposeCfg.MaxPatterns, "max-patterns", patterns.DefaultMaxPatternsInPrompt, "Max review patterns injected into the prompt (<=0 disables truncation, env: "+envMaxPatterns+")")
	proposeFlags.BoolVar(&proposeCfg.CreateIssues, "create-issues", false, "Interactively create GitHub issues from proposals")
	proposeFlags.BoolVar(&proposeCfg.NoIssueDedupe, "no-issue-dedupe", false, "Do not filter proposals whose title matches an existing GitHub issue")

	rootCmd.AddCommand(proposeCmd)

	// audit subcommand
	var auditCfg cli.AuditConfig
	var auditMinSeverity string
	var auditIssueMinSeverity string

	auditCmd := &cobra.Command{
		Use:   "audit <repo-ref>",
		Short: "Apply all known review patterns to an entire codebase",
		Long: `Clone a GitHub repository and apply every loaded review pattern to the
entire current state of the codebase. Produces concrete, prioritized
improvement findings (blocking/critical/warning/info) with file paths,
line numbers, and suggested fixes — identical finding format to the
review command, but analyzing the whole repo instead of a PR diff.

Repository reference can be a URL (https://github.com/owner/repo)
or short form (owner/repo).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			auditCfg.RepoRef = args[0]

			fileCfg.ApplyAudit(&auditCfg, &auditMinSeverity, &auditIssueMinSeverity, cmd.Flags().Changed)

			maxPatterns, err := resolveMaxPatterns(auditCfg.MaxPatterns, cmd.Flags().Changed("max-patterns"), fileCfg.Audit.MaxPatterns)
			if err != nil {
				return err
			}
			auditCfg.MaxPatterns = maxPatterns

			if auditMinSeverity != "" {
				sev, err := report.ParseSeverity(auditMinSeverity)
				if err != nil {
					return err
				}
				auditCfg.MinSeverity = sev
			} else {
				auditCfg.MinSeverity = report.SeverityInfo
			}

			switch auditCfg.Format {
			case formatMarkdown, formatJSON:
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json", auditCfg.Format)
			}

			if auditIssueMinSeverity != "" {
				sev, err := report.ParseSeverity(auditIssueMinSeverity)
				if err != nil {
					return err
				}
				auditCfg.IssueMinSeverity = sev
			} else {
				auditCfg.IssueMinSeverity = report.SeverityWarning
			}

			if auditCfg.CreateIssues && auditCfg.Format == formatJSON {
				return fmt.Errorf("--create-issues cannot be used with --format json")
			}

			opts := auditCfg.ToAuditOptions(version)
			return audit.Run(os.Stdout, opts, claude.Audit)
		},
	}

	auditFlags := auditCmd.Flags()
	auditFlags.StringSliceVar(&auditCfg.PatternDirs, "patterns", nil, "Additional pattern directories")
	auditFlags.StringVar(&auditMinSeverity, "min-severity", "", "Minimum severity level (info, warning, critical, blocking)")
	auditFlags.BoolVar(&auditCfg.NoRepoPatterns, "no-repo-patterns", false, "Ignore repo-specific patterns")
	auditFlags.BoolVar(&auditCfg.NoLocalPatterns, "no-local-patterns", false, "Ignore local patterns from the tool")
	auditFlags.BoolVar(&auditCfg.NoCache, "no-cache", false, "Ignore cache, force a fresh audit")
	auditFlags.StringVar(&auditCfg.Format, "format", "markdown", "Output format (markdown, json)")
	auditFlags.IntVar(&auditCfg.MaxPatterns, "max-patterns", patterns.DefaultMaxPatternsInPrompt, "Max review patterns injected into the prompt (<=0 disables truncation, env: "+envMaxPatterns+")")
	auditFlags.IntVar(&auditCfg.MaxFindings, "max-findings", 0, "Cap on findings returned (<=0 disables cap)")
	auditFlags.BoolVar(&auditCfg.CreateIssues, "create-issues", false, "Interactively create GitHub issues from audit findings")
	auditFlags.StringVar(&auditIssueMinSeverity, "issue-min-severity", "", "Minimum severity for issue creation (default WARNING)")
	auditFlags.BoolVar(&auditCfg.NoIssueDedupe, "no-issue-dedupe", false, "Do not filter findings whose title matches an existing GitHub issue")

	rootCmd.AddCommand(auditCmd)

	// gen-man-pages: hidden helper used by release tooling to emit man pages.
	genManCmd := &cobra.Command{
		Use:    "gen-man-pages <dir>",
		Short:  "Generate man pages into the given directory",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return err
			}
			header := &doc.GenManHeader{
				Title:   "PLANWERK-REVIEW",
				Section: "1",
				Source:  "planwerk-review " + version,
			}
			return doc.GenManTree(rootCmd, header, dir)
		},
	}
	rootCmd.AddCommand(genManCmd)

	if err := rootCmd.Execute(); err != nil {
		// Route the final error through slog so it honors --log-format.
		slog.Error(err.Error())
		os.Exit(1)
	}
}
