// Package linkrewrite handles Markdown link rewriting across different output formats.
// It transforms .md file references in HTML content to appropriate targets
// (anchor links for single-page, page links for multi-page site).
package linkrewrite

import (
	"net/url"
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
	// ModeEpub rewrites .md links to the flat <chapterID>.xhtml documents an
	// ePub's OEBPS directory is made of. Without it a cross-chapter link ships
	// as a dead .md href, which epubcheck reports as RSC-007 and readers
	// silently ignore.
	ModeEpub Mode = "epub"
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
	// A query string has to come off before the extension check, otherwise
	// filepath.Ext("install.md?os=linux") is ".md?os=linux", the .md guard below
	// bails out, and the link ships to the site as a dead href to a .md file that
	// was never published.
	query := ""
	if idx := strings.Index(pathPart, "?"); idx >= 0 {
		query = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}

	// goldmark percent-encodes link destinations, so a chapter under "user guide/"
	// arrives here as "user%20guide/README.md" and misses the target map, which is
	// keyed on raw file paths. Without decoding, every link into a directory or file
	// whose name has a space (or any non-ASCII character) becomes a 404 in the
	// published output.
	if decoded, err := url.PathUnescape(pathPart); err == nil {
		pathPart = decoded
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
		// The query is only carried over for the site, where pages are served over
		// HTTP and "install.html?os=linux" still means something. In an ePub or a
		// single HTML file the target is a packaged document or an in-page anchor,
		// so a query would just be noise appended to a local resource name.
		return withSuffix(relativeSiteTarget(currentFile, target.PageFilename), query, fragment), true, false
	case ModeEpub:
		// ePub chapters are flat siblings in OEBPS/, so the target is just the
		// chapter's own document name regardless of the source directory.
		if target.ChapterID == "" {
			return "", false, true
		}
		return withSuffix(target.ChapterID+".xhtml", "", fragment), true, false
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

// withSuffix reattaches the query and fragment that were split off the href
// before the chapter lookup, in URL order (?query then #fragment).
func withSuffix(base, query, fragment string) string {
	if query != "" {
		base += "?" + query
	}
	if fragment != "" {
		base += "#" + fragment
	}
	return base
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
