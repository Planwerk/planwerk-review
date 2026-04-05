package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/planwerk/planwerk-review/internal/audit"
	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/claude"
	"github.com/planwerk/planwerk-review/internal/cli"
	"github.com/planwerk/planwerk-review/internal/patterns"
	"github.com/planwerk/planwerk-review/internal/propose"
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/review"
)

// envMaxPatterns is the environment variable used to override the default
// maximum number of review patterns injected into the prompt.
const envMaxPatterns = "PLANWERK_MAX_PATTERNS"

// resolveMaxPatterns returns the effective max-patterns limit, preferring
// an explicitly set CLI flag, then PLANWERK_MAX_PATTERNS, then the default.
// A value of 0 or negative disables truncation.
func resolveMaxPatterns(flagValue int, flagSet bool) (int, error) {
	if flagSet {
		return flagValue, nil
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

var version = "dev"

func main() {
	var cfg cli.Config
	var minSeverity string

	rootCmd := &cobra.Command{
		Use:   "planwerk-review <pr-ref>",
		Short: "AI-powered code review for GitHub Pull Requests",
		Long: `planwerk-review uses Claude CLI to analyze GitHub PR changes and produces
structured, categorized review results as Markdown or JSON output.

PR reference can be a URL (https://github.com/owner/repo/pull/123)
or short form (owner/repo#123).`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.ClearCache {
				return cache.Clear()
			}

			if len(args) == 0 {
				return fmt.Errorf("requires a PR reference argument")
			}
			cfg.PRRef = args[0]

			maxPatterns, err := resolveMaxPatterns(cfg.MaxPatterns, cmd.Flags().Changed("max-patterns"))
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
			case "markdown", "json":
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json", cfg.Format)
			}

			if cfg.PostReview && cfg.Format == "json" {
				return fmt.Errorf("--post-review cannot be used with --format json")
			}

			// --inline implies --post-review
			if cfg.InlineReview {
				cfg.PostReview = true
			}
			if cfg.InlineReview && cfg.Format == "json" {
				return fmt.Errorf("--inline cannot be used with --format json")
			}

			opts := cfg.ToReviewOptions(version)
			return review.Run(os.Stdout, opts)
		},
	}

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

			switch proposeCfg.Format {
			case "markdown", "json", "issues":
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json, issues", proposeCfg.Format)
			}

			opts := proposeCfg.ToProposeOptions(version)
			return propose.Run(os.Stdout, opts, claude.Propose)
		},
	}

	proposeFlags := proposeCmd.Flags()
	proposeFlags.BoolVar(&proposeCfg.NoCache, "no-cache", false, "Ignore cache, force a fresh analysis")
	proposeFlags.StringVar(&proposeCfg.Format, "format", "markdown", "Output format (markdown, json, issues)")
	proposeFlags.BoolVar(&proposeCfg.CreateIssues, "create-issues", false, "Interactively create GitHub issues from proposals")

	rootCmd.AddCommand(proposeCmd)

	// audit subcommand
	var auditCfg cli.AuditConfig
	var auditMinSeverity string

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

			maxPatterns, err := resolveMaxPatterns(auditCfg.MaxPatterns, cmd.Flags().Changed("max-patterns"))
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
			case "markdown", "json":
			default:
				return fmt.Errorf("unknown format %q, supported: markdown, json", auditCfg.Format)
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

	rootCmd.AddCommand(auditCmd)
	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
