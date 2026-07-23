package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// defaultVersion is the compiled-in version. When mdpress is built via
// -ldflags (goreleaser release), Version is overridden to the real tag.
// For `go install`/source builds it stays at defaultVersion, and we fall
// back to runtime/debug build info at startup (see initBuildInfo).
const defaultVersion = "0.8.0"

var (
	// Version is the release version, injected at build time via -ldflags.
	//
	// It starts empty, and empty means "not injected" — a `go install` or a
	// plain `go build`, where initBuildInfo falls back to the embedded VCS
	// data and finally to defaultVersion. It must not start at defaultVersion:
	// goreleaser injects exactly the string defaultVersion holds (both are the
	// release tag), so comparing the two to decide whether a value was
	// injected is always true on a release build. The release binary then
	// discarded the injected tag and reported the module version instead,
	// which reads "<tag>+dirty" whenever a before-hook touches the tree — and
	// `go mod tidy` plus `go test ./...` run as before-hooks, so every
	// published binary since at least v0.7.15 called itself dirty.
	Version = ""
	// BuildTime is overridden at build time via -ldflags.
	BuildTime = "unknown"
	// Commit is overridden at build time via -ldflags (git commit hash).
	// Left empty when not injected; populated from build info for source builds.
	Commit = ""
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

// resolveVersion picks the version to report. injected is the -ldflags value,
// empty when the binary was not stamped; moduleVersion is Go's own
// info.Main.Version, which is "" or "(devel)" when unknown and carries a
// "+dirty" suffix when the tree was modified at build time.
//
// An injected value always wins, including when it equals defaultVersion. The
// two are equal on every correctly prepared release — goreleaser injects the
// tag and the release checklist bumps defaultVersion to that same tag — so
// treating the equality as "nothing was injected" discarded the real tag.
func resolveVersion(injected, moduleVersion string) string {
	if injected != "" {
		return injected
	}
	if moduleVersion != "" && moduleVersion != "(devel)" {
		return strings.TrimPrefix(moduleVersion, "v")
	}
	return defaultVersion
}

// initBuildInfo backfills Version/Commit/BuildTime from the embedded Go build
// info for builds that were NOT stamped via -ldflags (e.g. `go install`).
// It never overrides values already injected at release time.
func initBuildInfo() {
	if info, ok := debug.ReadBuildInfo(); ok {
		Version = resolveVersion(Version, info.Main.Version)

		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if Commit == "" && s.Value != "" {
					Commit = s.Value
				}
			case "vcs.time":
				if (BuildTime == "" || BuildTime == "unknown") && s.Value != "" {
					BuildTime = s.Value
				}
			}
		}
	}

	// Nothing injected and no usable build info: a plain `go build` of the
	// source tree.
	if Version == "" {
		Version = defaultVersion
	}
}

// init configures the root command.
func init() {
	initBuildInfo()

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
  mdpress config show          Print the effective configuration
  mdpress doctor               Check environment and project readiness`,
		Version: Version,
		// Cobra otherwise prints the error itself AND dumps full usage on any
		// RunE failure. We silence both here so Execute() is the single error
		// print path and runtime failures don't bury the message under usage.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Define global persistent flags.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "book.yaml", "Path to the config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging (show all warnings in detail)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode: only output errors, suppress warnings and info")
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "Override mdpress runtime cache directory")
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Disable mdpress runtime caches for this command")

	// Without these, completing a flag value offered every file in the
	// directory — including the ones that can never be right.
	_ = rootCmd.MarkPersistentFlagFilename("config", "yaml", "yml")
	_ = rootCmd.MarkPersistentFlagDirname("cache-dir")

	// Configure cache environment AFTER Cobra parses flags.
	// This must be in PersistentPreRun (not before ExecuteContext) so that
	// --cache-dir and --no-cache flags have been parsed by Cobra.
	// Note: if a subcommand defines its own PersistentPreRun, it must
	// manually call configureRuntimeCacheEnv() — Cobra does not chain them.
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		configureRuntimeCacheEnv()
		// --quiet/--verbose are advertised as global flags, but only build and
		// serve used to install the logger they configure. Every other command
		// kept slog's default handler, so `mdpress init -q` still printed INFO
		// lines — in stdlib log format, on a different stream. Installing it
		// here makes the flags mean the same thing for every subcommand.
		initLogger()
	}

	// Register subcommands.
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(quickstartCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(themesCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(cacheCmd)
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
		// Usage mistakes are the one class where the next step is always the
		// same, and a bare one-liner left the user guessing.
		if isUsageError(err) {
			fmt.Fprintln(os.Stderr, "\nRun 'mdpress --help' to see available commands and flags.")
		}
		return err
	}
	return nil
}

// usageErrorPhrases are the messages cobra produces for a malformed command
// line, as opposed to a command that ran and failed.
var usageErrorPhrases = []string{
	"unknown flag",
	"unknown shorthand flag",
	"unknown command",
	"accepts ",
	"requires at least",
	"invalid argument",
	"flag needs an argument",
}

// isUsageError reports whether err is about how the command was invoked.
func isUsageError(err error) bool {
	msg := err.Error()
	for _, phrase := range usageErrorPhrases {
		if strings.Contains(msg, phrase) {
			return true
		}
	}
	return false
}

// initLogger creates a logger based on the global quiet/verbose flags.
// Used by executeBuild and executeServe to avoid duplicating the setup.
func initLogger() *slog.Logger {
	// Warn, not Info, by default. The INFO stream is a running commentary on
	// internal stages; printed alongside the step progress it interleaved with
	// it mid-line and buried the parts a user acts on. --verbose brings it back
	// (and more).
	logLevel := slog.LevelWarn
	switch {
	case quiet:
		logLevel = slog.LevelError
	case verbose:
		logLevel = slog.LevelDebug
	}
	// Diagnostics belong on stderr. On stdout they were swallowed by
	// `mdpress build > build.log` along with every warning, and they polluted
	// any attempt to pipe the build's real output.
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func configureRuntimeCacheEnv() {
	if cacheDir != "" {
		if err := os.Setenv("MDPRESS_CACHE_DIR", cacheDir); err != nil {
			slog.Debug("Failed to set MDPRESS_CACHE_DIR environment variable", slog.Any("error", err))
		}
	}
	if noCache {
		if err := os.Setenv("MDPRESS_DISABLE_CACHE", "1"); err != nil {
			slog.Debug("Failed to set MDPRESS_DISABLE_CACHE environment variable", slog.Any("error", err))
		}
	}
}
