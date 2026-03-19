// Package variables expands template variables inside Markdown content.
// Variables such as {{ book.title }} and {{ book.author }} are replaced before parsing.
package variables

import (
	"regexp"
	"strings"

	"github.com/yeasy/mdpress/internal/config"
)

// varPattern matches template variables in the form {{ key }} or {{key}}.
var varPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.]+)\s*\}\}`)

// Expand replaces template variables in Markdown source using config values.
func Expand(source []byte, cfg *config.BookConfig) []byte {
	if cfg == nil {
		return source
	}

	vars := buildVarMap(cfg)

	result := varPattern.ReplaceAllFunc(source, func(match []byte) []byte {
		parts := varPattern.FindSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		key := strings.TrimSpace(string(parts[1]))
		if val, ok := vars[key]; ok {
			return []byte(val)
		}
		// Leave unknown variables unchanged.
		return match
	})

	return result
}

// ExpandString is the string wrapper for Expand.
func ExpandString(source string, cfg *config.BookConfig) string {
	return string(Expand([]byte(source), cfg))
}

// buildVarMap builds the variable lookup table from config.
func buildVarMap(cfg *config.BookConfig) map[string]string {
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
