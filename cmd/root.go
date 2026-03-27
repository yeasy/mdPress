package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	// Version is overridden at build time via -ldflags.
	Version = "0.6.6"
	// BuildTime is overridden at build time via -ldflags.
	BuildTime = "unknown"
	// rootCmd is the root command for the mdpress application.
	rootCmd *cobra.Command
	// cfgFile stores the config file path.
	cfgFile string
	// verbose enables detailed logging, showing all debug information and individual warnings.
	verbose bool
	// quiet enables quiet mode, outputting only errors and suppressing warnings and info logs.
	quiet bool
	// cacheDir overrides the runtime cache directory.
	cacheDir string
	// noCache disables mdpress runtime caches for the current command.
	noCache bool
)

// init configures the root command.
func init() {
	rootCmd = &cobra.Command{
		Use:   "mdpress",
		Short: "mdpress - Markdown book publishing tool",
		Long: `mdpress is a Markdown publishing tool for building high-quality
site, PDF, HTML, and ePub output from book-style content.

Features:
  - Generate site, PDF, HTML, and ePub from Markdown sources
  - Auto-generate a table of contents and cover page
  - Support SUMMARY.md (GitBook compatible) and GLOSSARY.md
  - Support custom themes, cross references, and template variables
  - Run a local preview server with live refresh via mdpress serve
  - Auto-discover .md files when book.yaml is missing
  - Build directly from a GitHub repository URL

Quick start:
  mdpress quickstart my-book   Create a sample project
  cd my-book && mdpress build  Build a PDF

Common commands:
  mdpress build                Build documents (PDF by default)
  mdpress build --format html  Build HTML
  mdpress serve                Start live preview
  mdpress validate             Validate project configuration
  mdpress doctor               Check environment and project readiness`,
		Version: Version,
	}

	// Define global persistent flags.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "book.yaml", "Path to the config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (show all warnings in detail)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode: only output errors, suppress warnings and info")
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "Override mdpress runtime cache directory")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Disable mdpress runtime caches for this command")

	// Configure cache environment AFTER Cobra parses flags.
	// This must be in PersistentPreRun (not before ExecuteContext) so that
	// --cache-dir and --no-cache flags have been parsed by Cobra.
	// Note: if a subcommand defines its own PersistentPreRun, it must
	// manually call configureRuntimeCacheEnv() — Cobra does not chain them.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		configureRuntimeCacheEnv()
	}

	// Register subcommands.
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(quickstartCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(themesCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}

func configureRuntimeCacheEnv() {
	if cacheDir != "" {
		if err := os.Setenv("MDPRESS_CACHE_DIR", cacheDir); err != nil {
			slog.Debug("Failed to set MDPRESS_CACHE_DIR environment variable", slog.String("error", err.Error()))
		}
	}
	if noCache {
		if err := os.Setenv("MDPRESS_DISABLE_CACHE", "1"); err != nil {
			slog.Debug("Failed to set MDPRESS_DISABLE_CACHE environment variable", slog.String("error", err.Error()))
		}
	}
}
