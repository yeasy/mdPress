// Package variables expands template variables inside Markdown content.
// Variables such as {{ book.title }} and {{ book.author }} are replaced before parsing.
package variables

import (
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
)

// varPattern matches template variables in the form {{ key }} or {{key}}.
var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// Expand replaces template variables in Markdown source using config values.
func Expand(source []byte, cfg *config.BookConfig) []byte {
	expanded, _ := ExpandWithUnknown(source, cfg)
	return expanded
}

// ExpandWithUnknown replaces template variables and also reports the ones it
// did not recognize, so the caller can warn instead of shipping a literal
// "{{ book.autor }}" to the reader.
//
// Substitution deliberately skips fenced code blocks and inline code spans. A
// book that documents a templating syntax — including mdpress's own manual —
// shows that syntax inside code, and rewriting it there corrupts the example
// the page exists to display.
func ExpandWithUnknown(source []byte, cfg *config.BookConfig) ([]byte, []string) {
	if cfg == nil {
		return source, nil
	}

	vars := buildVarMap(cfg)
	var unknown []string
	seen := map[string]bool{}

	substitute := func(segment string) string {
		return varPattern.ReplaceAllStringFunc(segment, func(match string) string {
			parts := varPattern.FindStringSubmatch(match)
			if len(parts) < 2 {
				return match
			}
			key := strings.TrimSpace(parts[1])
			if val, ok := vars[key]; ok {
				return val
			}
			// Leave unknown variables unchanged, but tell the caller: a typo
			// that renders as literal braces looks like a mdpress bug to the
			// reader and gives the author nothing to search for.
			if !seen[key] {
				seen[key] = true
				unknown = append(unknown, key)
			}
			return match
		})
	}

	result := markdown.ProcessOutsideCode(string(source), func(chunk string) string {
		return markdown.ProcessOutsideCodeSpans(chunk, substitute)
	})
	return []byte(result), unknown
}

// buildVarMap builds the variable lookup table from config.
func buildVarMap(cfg *config.BookConfig) map[string]string {
	vars := builtinVars(cfg)
	// User-defined variables come last so a book can override a built-in.
	for k, v := range cfg.Variables {
		vars[k] = v
	}
	return vars
}

func builtinVars(cfg *config.BookConfig) map[string]string {
	return map[string]string{
		"book.title":       cfg.Book.Title,
		"book.subtitle":    cfg.Book.Subtitle,
		"book.author":      cfg.Book.Author,
		"book.version":     cfg.Book.Version,
		"book.language":    cfg.Book.Language,
		"book.description": cfg.Book.Description,
		"style.theme":      cfg.Style.Theme,
		"style.page_size":  cfg.Style.PageSize,
		"output.filename":  cfg.Output.Filename,
	}
}
