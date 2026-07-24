package cmd

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resetCommandTree puts every flag in the Cobra tree back to its default and
// drops the args/out/err a previous Execute left behind.
//
// The values live on the package-level commands, so they survive an Execute:
// a test that ran `themes --help` left themesCmd's auto-generated help flag
// set to true, and cobra then answered the *next* `themes lst` with the help
// text and a nil error. That is why `go test -count=3 ./cmd/` reported a red
// suite on a healthy tree while -count=1 was green.
func resetCommandTree(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	c.SetArgs(nil)
	c.SetOut(nil)
	c.SetErr(nil)
	for _, sub := range c.Commands() {
		resetCommandTree(sub)
	}
}

// restoreGlobalFlags snapshots the package-level flag variables the root
// command writes into, so a test that runs a command through the Cobra tree
// cannot leak its flags into the tests that follow — and clears whatever the
// tests that ran *before* it left on the shared tree.
func restoreGlobalFlags(t *testing.T) {
	t.Helper()
	resetCommandTree(rootCmd)
	origCfg, origQuiet, origVerbose := cfgFile, quiet, verbose
	origFormat, origOutput, origSubDir, origBranch, origSummary := buildFormat, buildOutput, buildSubDir, buildBranch, buildSummary
	origReport, origStrict := doctorReportPath, doctorStrict
	origHandler := slog.Default().Handler()
	t.Cleanup(func() {
		resetCommandTree(rootCmd)
		cfgFile, quiet, verbose = origCfg, origQuiet, origVerbose
		buildFormat, buildOutput, buildSubDir, buildBranch, buildSummary = origFormat, origOutput, origSubDir, origBranch, origSummary
		doctorReportPath, doctorStrict = origReport, origStrict
		slog.SetDefault(slog.New(origHandler))
	})
}

// TestGlobalQuietFlagAppliesToEverySubcommand covers finding 2: --quiet and
// --verbose were advertised as global but only build and serve installed the
// logger they configure, so every other command kept slog's default handler
// (INFO level, stdlib format). `version` is used here precisely because it
// never touched the logger itself.
func TestGlobalQuietFlagAppliesToEverySubcommand(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	rootCmd.SetArgs([]string{"version", "--quiet"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version --quiet returned error: %v", err)
	}
	if slog.Default().Enabled(context.Background(), slog.LevelWarn) {
		t.Error("--quiet should silence warnings for every subcommand, but the default logger still accepts WARN")
	}

	// Cobra keeps flag values between Execute calls in-process; a real second
	// invocation would start from the defaults.
	quiet = false
	rootCmd.SetArgs([]string{"version", "--verbose"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version --verbose returned error: %v", err)
	}
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("--verbose should enable debug logging for every subcommand")
	}
}

// TestThemesUnknownSubcommandSuggests covers finding 1: `mdpress themes lst`
// printed nothing and exited 0.
func TestThemesUnknownSubcommandSuggests(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	rootCmd.SetArgs([]string{"themes", "lst"})
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("a mistyped themes sub-command should be an error, not a silent success")
	}
	if !strings.Contains(err.Error(), "list") {
		t.Errorf("error should suggest the intended sub-command, got: %v", err)
	}
}

// TestThemesPreviewCreatesOutputParentDirs covers finding 33: the documented
// example wrote into ./artifacts/, which failed because nothing created it.
func TestThemesPreviewCreatesOutputParentDirs(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "artifacts", "themes", "preview.html")

	rootCmd.SetArgs([]string{"themes", "preview", "--output", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("themes preview into a missing directory returned error: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected preview file at %s: %v", target, err)
	}
}

// TestParseFormatFlag covers finding 9: mixed case was rejected, a stray comma
// aborted the build with `unsupported format ""`, and an empty --format was
// silently treated as "pdf".
func TestParseFormatFlag(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "uppercase", raw: "PDF", want: []string{"pdf"}},
		{name: "surrounding spaces", raw: " html ", want: []string{"html"}},
		{name: "stray comma", raw: "pdf,,html", want: []string{"pdf", "html"}},
		{name: "trailing comma", raw: "pdf,", want: []string{"pdf"}},
		{name: "mixed case list", raw: "PDF, Site", want: []string{"pdf", "site"}},
		{name: "only separators", raw: ",,", wantErr: true},
		{name: "only spaces", raw: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFormatFlag(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseFormatFlag(%q) should fail, got %v", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFormatFlag(%q) returned error: %v", tt.raw, err)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("parseFormatFlag(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// TestBuildRejectsEmptyFormatFlag covers the `--format ""` half of finding 9:
// by the time executeBuild runs, an empty value is indistinguishable from an
// unset one, so the typo used to build a PDF instead of being reported.
func TestBuildRejectsEmptyFormatFlag(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	t.Chdir(t.TempDir())
	rootCmd.SetArgs([]string{"build", "--format", ""})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("build --format '' should be rejected, not silently treated as pdf")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Errorf("error should name the offending flag, got: %v", err)
	}
}

// TestBuildSubDirUsesSubDirConfig covers finding 14: for a local source
// --subdir was silently ignored — the sub-directory's book.yaml was dropped,
// zero-config discovery ran on the parent, and the artifact was named after
// the first heading it found and written beside the wrong directory.
func TestBuildSubDirUsesSubDirConfig(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	workspace := t.TempDir()
	bookDir := filepath.Join(workspace, "mybook")
	if err := os.MkdirAll(bookDir, 0o755); err != nil {
		t.Fatalf("create book dir: %v", err)
	}
	writeTestFile(t, filepath.Join(bookDir, "ch1.md"), "# Chapter One\n\nBody.\n")
	writeTestFile(t, filepath.Join(bookDir, "book.yaml"), `book:
  title: "Sub Book"
chapters:
  - title: "Chapter One"
    file: "ch1.md"
output:
  formats: ["html"]
`)

	t.Chdir(workspace)
	cfgFile = defaultConfigName
	buildFormat = "html"
	buildOutput = ""
	buildSummary = ""
	buildSubDir = "mybook"
	buildBranch = ""
	quiet = true

	if err := executeBuild(context.Background(), ""); err != nil {
		t.Fatalf("executeBuild() returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(bookDir, "Sub-Book.html")); err != nil {
		// The exact sanitization of the title is owned elsewhere; assert on the
		// directory and on the fact that nothing landed in the parent.
		matches, _ := filepath.Glob(filepath.Join(bookDir, "*.html"))
		if len(matches) == 0 {
			t.Fatalf("expected the artifact inside %s, found none", bookDir)
		}
	}
	parentMatches, _ := filepath.Glob(filepath.Join(workspace, "*.html"))
	if len(parentMatches) > 0 {
		t.Fatalf("--subdir build wrote into the parent directory: %v", parentMatches)
	}
}

// TestBuildRejectsBranchWithoutRemoteSource covers the other half of finding
// 14: --branch only means something for a cloned source, and accepting it for
// a local build made a mistyped command look like it had honored it.
func TestBuildRejectsBranchWithoutRemoteSource(t *testing.T) {
	restoreGlobalFlags(t)
	defer suppressOutput(t)()

	t.Chdir(t.TempDir())
	cfgFile = defaultConfigName
	buildFormat = "html"
	buildSubDir = ""
	buildBranch = "main"
	quiet = true

	err := executeBuild(context.Background(), "")
	if err == nil {
		t.Fatal("--branch without a Git source should be rejected")
	}
	if !strings.Contains(err.Error(), "--branch") {
		t.Errorf("error should name --branch, got: %v", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
