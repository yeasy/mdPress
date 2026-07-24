// docs_config_test.go checks the book.yaml the documentation tells people to
// write against the real config schema, and the flag shorthands printed in
// documentation tables against the real CLI.
//
// docs_flags_test.go already covers `mdpress … --flag` command lines, but the
// two things that actually went wrong were invisible to it: the GitBook
// migration guide's "converts to mdPress book.yaml" block used seven keys that
// do not exist (a reader who copied it got a book titled "Untitled Book"), the
// Custom CSS page invented `style.custom_css_file`, and the CLI reference gave
// `mdpress doctor`'s report flag as `-o` when the binary rejects it. All three
// live in YAML blocks and Markdown tables, not in command lines.
package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/yeasy/mdpress/internal/config"
	"gopkg.in/yaml.v3"
)

// docYAMLBlockPattern matches a fenced ```yaml block and captures its body.
var docYAMLBlockPattern = regexp.MustCompile("(?s)```ya?ml\\n(.*?)```")

// docFlagCellPattern matches a "-r, --report" style shorthand+long pair as it
// appears in a documentation flag table.
var docFlagCellPattern = regexp.MustCompile(`-([a-zA-Z]), --([a-z][a-z0-9-]*)`)

// bookConfigTopLevelKeys are the top-level keys that identify a YAML block as
// a book.yaml rather than a theme file, a plugin manifest or a CI workflow.
// "variables" and "plugins" are deliberately absent: GitLab CI pipelines in
// the CI/CD pages use "variables:" too, and matching on it made every workflow
// example in the manual look like a broken book.yaml.
var bookConfigTopLevelKeys = map[string]bool{
	"book":     true,
	"chapters": true,
	"style":    true,
	"output":   true,
	"markdown": true,
}

// collectDocFiles returns every Markdown file that ships as user-facing
// documentation: the manual and command docs under docs/, plus the READMEs.
// ROADMAP and CHANGELOG are excluded — they record history, including config
// shapes that were once wrong.
func collectDocFiles(t *testing.T) []string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	walkErr := filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // unreadable paths simply contribute nothing
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".md") || strings.HasPrefix(base, "ROADMAP") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk docs: %v", walkErr)
	}
	for _, name := range []string{"README.md", "README_zh.md"} {
		files = append(files, filepath.Join(root, name))
	}
	if len(files) == 0 {
		t.Fatal("no documentation files found; the path is wrong")
	}
	return files
}

// relativeDocPath shortens an absolute doc path for error messages.
func relativeDocPath(t *testing.T, path string) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func TestDocumentedConfigKeysExist(t *testing.T) {
	for _, file := range collectDocFiles(t) {
		data, err := os.ReadFile(file) //nolint:gosec // G304: reading the repo's own docs
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}
		rel := relativeDocPath(t, file)

		for _, match := range docYAMLBlockPattern.FindAllStringSubmatch(string(data), -1) {
			body := match[1]
			var top map[string]any
			if yaml.Unmarshal([]byte(body), &top) != nil || top == nil {
				continue // not a mapping: a list example, a fragment, a snippet with placeholders
			}
			isBookConfig := false
			for key := range top {
				if bookConfigTopLevelKeys[key] {
					isBookConfig = true
					break
				}
			}
			if !isBookConfig {
				continue
			}
			for _, unknown := range config.FindUnknownKeys([]byte(body)) {
				t.Errorf("%s documents book.yaml key %q, which mdpress does not recognize (%s)",
					rel, unknown.Path, unknown.Hint())
			}
		}
	}
}

// TestDocumentedCustomCSSIsAPath guards the specific mistake that made the
// Custom CSS page useless: style.custom_css is a path that the build passes to
// os.Stat, so a documented `custom_css: |` block is silently discarded and the
// warning prints the reader's own stylesheet as a filename. FindUnknownKeys
// cannot see this — the key is real, only the value shape is wrong.
func TestDocumentedCustomCSSIsAPath(t *testing.T) {
	for _, file := range collectDocFiles(t) {
		data, err := os.ReadFile(file) //nolint:gosec // G304: reading the repo's own docs
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "custom_css: |" || trimmed == "custom_css: >" {
				t.Errorf("%s:%d documents inline CSS after `custom_css:`, but the value is a file path",
					relativeDocPath(t, file), i+1)
			}
		}
	}
}

// TestDocumentedFlagShorthandsExist checks "-x, --long" pairs printed in
// documentation tables. `mdpress doctor -o report.json` was documented for two
// releases after the shorthand became -r; anyone scripting it from the CLI
// reference got "unknown shorthand flag: 'o' in -o".
func TestDocumentedFlagShorthandsExist(t *testing.T) {
	shorthands := map[string]map[string]bool{}
	var collect func(*cobra.Command)
	collect = func(command *cobra.Command) {
		command.InitDefaultHelpFlag()
		command.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Shorthand == "" {
				return
			}
			if shorthands[f.Name] == nil {
				shorthands[f.Name] = map[string]bool{}
			}
			shorthands[f.Name][f.Shorthand] = true
		})
		for _, sub := range command.Commands() {
			collect(sub)
		}
	}
	collect(rootCmd)

	for _, file := range collectDocFiles(t) {
		data, err := os.ReadFile(file) //nolint:gosec // G304: reading the repo's own docs
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}
		rel := relativeDocPath(t, file)
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				continue // only flag tables; prose uses long flags
			}
			for _, match := range docFlagCellPattern.FindAllStringSubmatch(line, -1) {
				short, long := match[1], match[2]
				known, documented := shorthands[long]
				if !documented {
					continue // a long flag with no shorthand anywhere; not this test's business
				}
				if !known[short] {
					t.Errorf("%s documents `-%s, --%s`, but --%s has no -%s shorthand in the CLI",
						rel, short, long, long, short)
				}
			}
		}
	}
}

// documentedDefaultPattern matches a "Defaults" bullet in the configuration
// reference: "- `book.language`: `en-US`" (or the Chinese page's full-width
// colon). Only the first backticked value is captured; the rest of the bullet
// is prose.
var documentedDefaultPattern = regexp.MustCompile("(?m)^- `([a-z_]+\\.[a-z_]+)`[:：]\\s*`([^`]+)`")

// TestDocumentedDefaultsMatchConfig checks the "Defaults" section of the
// configuration reference against config.DefaultConfig().
//
// This list has gone stale twice: it still claimed `book.language: zh-CN`
// after the default became en-US in 0.7.15, and `output.filename: output.pdf`
// after 0.8.0 removed that literal. A Chinese author who trusted the first one
// omitted `language:` and got an English site UI with no way to tell why.
func TestDocumentedDefaultsMatchConfig(t *testing.T) {
	defaults := config.DefaultConfig()
	actual := map[string]string{
		"book.title":                defaults.Book.Title,
		"book.language":             defaults.Book.Language,
		"style.theme":               defaults.Style.Theme,
		"style.page_size":           defaults.Style.PageSize,
		"style.code_theme":          defaults.Style.CodeTheme,
		"output.filename":           defaults.Output.Filename,
		"output.toc_max_depth":      strconv.Itoa(defaults.Output.TOCMaxDepth),
		"output.generate_bookmarks": strconv.FormatBool(defaults.Output.GenerateBookmarks),
		"output.show_theme_badge":   strconv.FormatBool(defaults.Output.ShowThemeBadge),
		"markdown.allow_html":       strconv.FormatBool(defaults.AllowRawHTML()),
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"en", "zh"} {
		path := filepath.Join(root, "docs", "manual", lang, "reference", "configuration.md")
		data, readErr := os.ReadFile(path) //nolint:gosec // G304: reading the repo's own docs
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		matches := documentedDefaultPattern.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			t.Errorf("%s/configuration.md: no `key`: `value` default bullets found; the format changed", lang)
		}
		for _, match := range matches {
			key, documented := match[1], match[2]
			want, tracked := actual[key]
			if !tracked {
				continue // a default this test does not know how to resolve
			}
			if documented != want {
				t.Errorf("%s/configuration.md documents %s default as %q, but DefaultConfig gives %q",
					lang, key, documented, want)
			}
		}
	}
}
