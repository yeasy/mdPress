// Package linkrewrite handles Markdown link rewriting across different output formats.
// It transforms .md file references in HTML content to appropriate targets
// (anchor links for single-page, page links for multi-page site).
package linkrewrite

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Mode determines how links are rewritten.
type Mode string

const (
	// ModeSingle rewrites .md links to #anchor links within a single page.
	ModeSingle Mode = "single"
	// ModeSite rewrites .md links to separate page filenames.
	ModeSite Mode = "site"
)

// Target describes the rewrite destination for a chapter.
type Target struct {
	ChapterID    string
	PageFilename string
}

var hrefAttrPattern = regexp.MustCompile(`href="([^"]+)"|href='([^']+)'`)

// RewriteLinks rewrites Markdown .md links in HTML content to the appropriate targets
// based on the output mode.
func RewriteLinks(htmlContent string, currentFile string, targets map[string]Target, mode Mode) string {
	if htmlContent == "" || currentFile == "" || len(targets) == 0 {
		return htmlContent
	}

	currentDir := filepath.Dir(NormalizePath(currentFile))
	return hrefAttrPattern.ReplaceAllStringFunc(htmlContent, func(match string) string {
		parts := hrefAttrPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		quote := `"`
		href := parts[1]
		if href == "" {
			quote = `'`
			href = parts[2]
		}

		rewritten, ok, unresolvedMarkdown := rewriteHref(href, currentDir, targets, mode)
		if !ok {
			if unresolvedMarkdown {
				return `href=` + quote + href + quote + ` data-mdpress-link="unresolved-markdown" title="Markdown link target is outside the current build graph"`
			}
			return match
		}

		return `href=` + quote + rewritten + quote
	})
}

// NormalizePath normalizes a chapter file path for consistent map lookups.
func NormalizePath(path string) string {
	cleaned := filepath.Clean(path)
	if cleaned == "." {
		return ""
	}
	return filepath.ToSlash(cleaned)
}

// rewriteHref processes a single href value.
// Returns (rewritten, ok, unresolvedMarkdown).
func rewriteHref(href string, currentDir string, targets map[string]Target, mode Mode) (string, bool, bool) {
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "/") {
		return "", false, false
	}

	lowerHref := strings.ToLower(href)
	for _, prefix := range []string{"http://", "https://", "mailto:", "tel:", "javascript:", "data:"} {
		if strings.HasPrefix(lowerHref, prefix) {
			return "", false, false
		}
	}
	if strings.HasPrefix(href, "//") {
		return "", false, false
	}

	pathPart := href
	fragment := ""
	if idx := strings.Index(pathPart, "#"); idx >= 0 {
		fragment = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}

	if pathPart == "" || strings.ToLower(filepath.Ext(pathPart)) != ".md" {
		return "", false, false
	}

	targetPath := NormalizePath(filepath.Join(currentDir, pathPart))
	target, ok := targets[targetPath]
	if !ok {
		return "", false, true
	}

	switch mode {
	case ModeSite:
		if target.PageFilename == "" {
			return "", false, true
		}
		if fragment != "" {
			return target.PageFilename + "#" + fragment, true, false
		}
		return target.PageFilename, true, false
	default:
		if fragment != "" {
			return "#" + fragment, true, false
		}
		if target.ChapterID == "" {
			return "", false, true
		}
		return "#" + target.ChapterID, true, false
	}
}
