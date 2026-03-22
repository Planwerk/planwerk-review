package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planwerk/planwerk-review/internal/cache"
	"github.com/planwerk/planwerk-review/internal/cli"
	"github.com/planwerk/planwerk-review/internal/report"
	"github.com/planwerk/planwerk-review/internal/review"
)

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

			if minSeverity != "" {
				sev, err := report.ParseSeverity(minSeverity)
				if err != nil {
					return err
				}
				cfg.MinSeverity = sev
			} else {
				cfg.MinSeverity = report.SeverityInfo
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

	rootCmd.Version = version

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
