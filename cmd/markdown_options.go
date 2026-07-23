// markdown_options.go maps book.yaml settings onto Markdown parser options.
// It lives apart from the pipeline so every place that spins up a parser —
// the orchestrator and each parse worker — derives the same configuration.
package cmd

import (
	"github.com/yeasy/mdpress/internal/config"
	"github.com/yeasy/mdpress/internal/markdown"
)

// markdownParserOptions returns the parser options implied by book.yaml.
func markdownParserOptions(cfg *config.BookConfig, codeTheme string) []markdown.ParserOption {
	opts := []markdown.ParserOption{markdown.WithCodeTheme(codeTheme)}
	if cfg != nil {
		opts = append(opts, markdown.WithAllowHTML(cfg.AllowRawHTML()))
	}
	return opts
}

// parserVariantKey identifies the parser configuration for the parsed-chapter
// cache. Anything that changes the rendered HTML for unchanged Markdown has to
// appear here, or flipping the setting would serve stale output from the cache.
func parserVariantKey(cfg *config.BookConfig, codeTheme string) string {
	if cfg != nil && !cfg.AllowRawHTML() {
		return codeTheme + "|no-raw-html"
	}
	return codeTheme
}
