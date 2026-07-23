// validate_orphans.go finds Markdown files that sit in the project directory
// but are not reachable from the chapter list.
//
// An orphan is invisible: it is not built, not searchable, not linked, and
// nothing in the output hints that it exists. The usual causes are a chapter
// that was renamed on disk but not in book.yaml, and a chapter someone wrote
// and forgot to register. Both look like a working project until a reader
// asks where the page went.
package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/linkrewrite"
)

// orphanExemptNames are Markdown files a book project keeps beside its
// chapters and never builds as one: mdpress's own structural files plus the
// repository-housekeeping set every project on a forge carries. Keys are
// upper-cased so the comparison is case-insensitive.
var orphanExemptNames = map[string]struct{}{
	"SUMMARY.MD":               {},
	"GLOSSARY.MD":              {},
	"LANGS.MD":                 {},
	"CHANGELOG.MD":             {},
	"CONTRIBUTING.MD":          {},
	"LICENSE.MD":               {},
	"CODE_OF_CONDUCT.MD":       {},
	"SECURITY.MD":              {},
	"AUTHORS.MD":               {},
	"CODEOWNERS.MD":            {},
	"PULL_REQUEST_TEMPLATE.MD": {},
	"ISSUE_TEMPLATE.MD":        {},
}

// orphanSkipDirs are directories that never hold chapters: build output,
// dependencies, vendored trees, and the conventional asset directories.
//
// A Markdown file inside an asset directory documents the assets — mdpress's
// own `quickstart` scaffolds images/README.md explaining what to put there —
// so reporting it as a forgotten chapter made the tool's own scaffold fail its
// own `validate --strict`.
var orphanSkipDirs = map[string]struct{}{
	"node_modules": {},
	"_book":        {},
	"vendor":       {},
	"dist":         {},
	"images":       {},
	"img":          {},
	"assets":       {},
	"static":       {},
	"public":       {},
	"media":        {},
}

// findOrphanMarkdownFiles returns the project-relative paths of Markdown files
// under the config directory that no chapter entry points at, sorted for a
// stable report. Paths are relative to the project root, matching how chapters
// are written in book.yaml.
func findOrphanMarkdownFiles(cfg *config.BookConfig) ([]string, error) {
	root := cfg.BaseDir()
	if root == "" {
		return nil, nil
	}
	// In a multi-language project the chapters belong to the per-language
	// configs, so every file under the root looks unreferenced from here.
	if cfg.LangsFile != "" {
		return nil, nil
	}

	referenced := make(map[string]struct{})
	for _, ch := range config.FlattenChapters(cfg.Chapters) {
		referenced[linkrewrite.NormalizePath(ch.File)] = struct{}{}
	}
	if len(referenced) == 0 {
		// Nothing to compare against; "every file is an orphan" is not a
		// useful thing to say about a project with no chapters.
		return nil, nil
	}
	for _, aux := range []string{cfg.GlossaryFile, cfg.LangsFile} {
		if aux == "" {
			continue
		}
		if rel, err := filepath.Rel(root, aux); err == nil {
			referenced[linkrewrite.NormalizePath(rel)] = struct{}{}
		}
	}

	var orphans []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Unreadable entries are the file checks' problem, not this one.
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if _, skip := orphanSkipDirs[base]; skip {
				return filepath.SkipDir
			}
			// A subdirectory with its own book.yaml is a separate project
			// (the multi-language layout, or examples/ in a repo); its files
			// belong to that project's chapter list, not this one.
			if _, err := os.Stat(filepath.Join(path, "book.yaml")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = linkrewrite.NormalizePath(rel)
		if _, ok := referenced[rel]; ok {
			return nil
		}
		if _, exempt := orphanExemptNames[strings.ToUpper(filepath.Base(rel))]; exempt {
			return nil
		}
		// A README.md at the project root is the repository's front page far
		// more often than it is a forgotten chapter. Nested ones
		// (chapter01/README.md) are the GitBook chapter convention, so they
		// are checked normally.
		if strings.EqualFold(rel, "README.md") {
			return nil
		}
		orphans = append(orphans, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(orphans)
	return orphans, nil
}
