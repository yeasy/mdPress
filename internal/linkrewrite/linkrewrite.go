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

		rewritten, ok, unresolvedMarkdown := rewriteHref(href, NormalizePath(currentFile), currentDir, targets, mode)
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
// It cleans the path, converts to forward slashes, and lowercases the file
// extension so that "chapter.MD" and "chapter.md" resolve to the same key.
func NormalizePath(path string) string {
	cleaned := filepath.Clean(path)
	if cleaned == "." {
		return ""
	}
	result := filepath.ToSlash(cleaned)
	if ext := filepath.Ext(result); ext != "" {
		result = strings.TrimSuffix(result, ext) + strings.ToLower(ext)
	}
	return result
}

// rewriteHref processes a single href value.
// Returns (rewritten, ok, unresolvedMarkdown).
func rewriteHref(href string, currentFile string, currentDir string, targets map[string]Target, mode Mode) (string, bool, bool) {
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
		baseTarget := relativeSiteTarget(currentFile, target.PageFilename)
		if fragment != "" {
			return baseTarget + "#" + fragment, true, false
		}
		return baseTarget, true, false
	case ModeSingle: // also handles unknown modes via default
		fallthrough
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

func relativeSiteTarget(currentFile, targetFile string) string {
	currentDir := filepath.Dir(filepath.Clean(currentFile))
	targetPath := filepath.Clean(targetFile)
	rel, err := filepath.Rel(currentDir, targetPath)
	if err != nil {
		return filepath.ToSlash(targetPath)
	}
	return filepath.ToSlash(rel)
}
