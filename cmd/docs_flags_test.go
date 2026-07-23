// docs_flags_test.go checks every `mdpress …` command line in the docs
// against the real CLI.
//
// Documentation that describes flags the binary does not have is worse than
// missing documentation: the user copies the command, it fails with a bare
// "unknown flag", and they have no way to tell whether they made a mistake or
// the docs did. This has happened repeatedly (the GitBook migration guide
// documented `migrate --output`, which never existed), so it is checked
// mechanically rather than by review.
package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// commandLinePattern matches a documented invocation, optionally prefixed with
// a shell prompt: "$ mdpress build --format pdf".
var commandLinePattern = regexp.MustCompile(`(?m)^\s*(?:\$\s*)?mdpress\s+(.+)$`)

// flagPattern matches a long flag, ignoring any =value suffix.
var flagPattern = regexp.MustCompile(`--([a-z][a-z0-9-]*)`)

// resolveDocumentedCommand walks the argument list to the deepest matching
// subcommand, mirroring how cobra dispatches.
func resolveDocumentedCommand(args []string) (*cobra.Command, bool) {
	current := rootCmd
	matched := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		var next *cobra.Command
		for _, sub := range current.Commands() {
			if sub.Name() == arg {
				next = sub
				break
			}
		}
		if next == nil {
			break
		}
		current = next
		matched = true
	}
	return current, matched
}

func TestDocumentedFlagsExist(t *testing.T) {
	docsRoot, err := filepath.Abs(filepath.Join("..", "docs"))
	if err != nil {
		t.Fatal(err)
	}

	var docFiles []string
	err = filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil //nolint:nilerr // unreadable paths simply contribute nothing
		}
		if strings.HasSuffix(path, ".md") {
			docFiles = append(docFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}
	if len(docFiles) == 0 {
		t.Fatal("no documentation files found; the path is wrong")
	}

	// Flags shown in docs that belong to an example's surrounding shell rather
	// than to mdpress itself would produce false positives; there are none
	// today, and any that appear should be rewritten rather than allowlisted.
	for _, file := range docFiles {
		data, err := os.ReadFile(file) //nolint:gosec // G304: walking the repo's own docs
		if err != nil {
			t.Errorf("read %s: %v", file, err)
			continue
		}
		rel, relErr := filepath.Rel(filepath.Join(docsRoot, ".."), file)
		if relErr != nil {
			rel = file
		}

		for _, match := range commandLinePattern.FindAllStringSubmatch(string(data), -1) {
			line := match[1]
			// Skip lines that continue into a shell pipeline or substitution,
			// where trailing flags belong to another program.
			if idx := strings.IndexAny(line, "|>"); idx >= 0 {
				line = line[:idx]
			}
			args := strings.Fields(line)
			command, matched := resolveDocumentedCommand(args)
			if !matched {
				continue // `mdpress <source>` or a placeholder, not a subcommand
			}
			// cobra registers --help lazily, on execution.
			command.InitDefaultHelpFlag()

			for _, flagMatch := range flagPattern.FindAllStringSubmatch(line, -1) {
				name := flagMatch[1]
				if command.Flags().Lookup(name) != nil || command.InheritedFlags().Lookup(name) != nil {
					continue
				}
				t.Errorf("%s documents `mdpress %s --%s`, but that command has no --%s flag",
					rel, command.Name(), name, name)
			}
		}
	}
}
