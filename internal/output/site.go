// site.go generates a multi-page static site similar to GitBook.
// It includes sidebar navigation, previous/next links, search, and responsive layout.
package output

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yeasy/mdpress/pkg/utils"
)

const (
	// maxSearchTextLength limits how much text is included per page in the search index.
	maxSearchTextLength = 50000
	// searchSnippetExtraRunes is the number of runes after a heading title to include in the search snippet.
	searchSnippetExtraRunes = 500
)

// SiteChapter stores rendered chapter data for site output.
type SiteChapter struct {
	Title    string
	ID       string
	Filename string // Output HTML filename, for example "ch01.html".
	Content  string // Rendered HTML content.
	Markdown string // Source markdown after variable expansion.
	// SourcePath is the chapter's markdown source path relative to the book
	// root (e.g. "chapter01/section1.md"), used to build "edit this page"
	// links. When empty, the path is derived from Filename as a best effort.
	SourcePath string
	Depth      int
	Headings   []SiteNavHeading
	Children   []SiteChapter
}

// SiteNavHeading stores an in-chapter navigation tree.
type SiteNavHeading struct {
	Title    string
	ID       string
	Children []SiteNavHeading
}

// SiteMeta stores site-wide metadata.
type SiteMeta struct {
	Title            string
	Subtitle         string
	Description      string
	Author           string
	Language         string
	Theme            string // CSS theme name.
	ThemeDescription string
	// SiteURL is the public base URL of the deployed site (e.g.
	// https://user.github.io/repo). When set, an absolute-URL sitemap.xml is
	// generated. Empty disables the sitemap.
	SiteURL string
	// EditBase is the base URL for "edit this page" links (e.g.
	// https://github.com/user/repo/edit/main/). Empty disables the links.
	EditBase string
}

// SiteGenerator generates the static site.
type SiteGenerator struct {
	Meta     SiteMeta
	Chapters []SiteChapter
	CSS      string // Theme CSS plus custom CSS.
}

// NewSiteGenerator creates a site generator.
func NewSiteGenerator(meta SiteMeta) *SiteGenerator {
	return &SiteGenerator{Meta: meta}
}

// validateFilename ensures that filename does not escape outputDir via path traversal.
// It rejects absolute paths and paths containing ".." to prevent writing outside the
// intended output directory. This is critical when filenames come from user-controlled
// sources such as SUMMARY.md chapter paths.
func validateFilename(outputDir, filename string) error {
	// Reject absolute paths
	if filepath.IsAbs(filename) {
		return fmt.Errorf("filename must be relative, not absolute: %q", filename)
	}

	cleaned := filepath.Clean(filename)

	// After cleaning, ".." only remains as a path component if the path escapes.
	for _, seg := range strings.Split(cleaned, string(filepath.Separator)) {
		if seg == ".." {
			return fmt.Errorf("filename must not contain '..': %q", filename)
		}
	}

	// Resolve the output directory (including symlinks) so that the containment
	// check works on macOS where /var -> /private/var.
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}
	if evaled, evalErr := filepath.EvalSymlinks(absOutputDir); evalErr == nil {
		absOutputDir = evaled
	}

	// Build the full path from the resolved base. The target file may not exist
	// yet, so EvalSymlinks would fail on it. Resolve the nearest existing ancestor
	// and re-append the non-existing tail to catch symlink-based escapes.
	absFullPath := filepath.Join(absOutputDir, cleaned)
	absFullPath = utils.EvalSymlinksAncestor(absFullPath)

	// Ensure the resolved path is within outputDir
	if !strings.HasPrefix(absFullPath, absOutputDir+string(filepath.Separator)) &&
		absFullPath != absOutputDir {
		return fmt.Errorf("filename escapes output directory: %q", filename)
	}

	return nil
}

// AddChapter appends a chapter.
func (g *SiteGenerator) AddChapter(ch SiteChapter) {
	g.Chapters = append(g.Chapters, ch)
}

// SetCSS sets the site CSS, sanitizing it to prevent style-tag breakout.
func (g *SiteGenerator) SetCSS(css string) {
	g.CSS = utils.SanitizeCSS(css)
}

// Generate generates the static site pages, sitemap, and search index.
func (g *SiteGenerator) Generate(outputDir string) error {
	if err := utils.EnsureDir(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Flatten nested chapters for previous/next navigation.
	flatPages := g.flattenChapters(g.Chapters)

	// Assign filenames to pages that do not already have one.
	for i := range flatPages {
		if flatPages[i].Filename == "" {
			flatPages[i].Filename = fmt.Sprintf("page_%d.html", i)
		}
		// Validate filename to prevent path traversal attacks
		if err := validateFilename(outputDir, flatPages[i].Filename); err != nil {
			return fmt.Errorf("invalid filename for page %d: %w", i, err)
		}
	}

	// Parse the page template.
	// SECURITY: safeHTML and safeCSS functions bypass the template escaper and must only be
	// called with internally-generated content. Currently used for:
	// - SidebarHTML: generated by buildSidebar() (trusted)
	// - Content: rendered Markdown output (trusted, never raw user input from forms/URLs)
	// DO NOT pass untrusted data through these functions; doing so creates XSS vulnerabilities.
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"safeCSS":  func(s string) template.CSS { return template.CSS(s) },
	}).Parse(sitePageTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse page template: %w", err)
	}

	// sitemap.xml is only generated when the public site URL is configured,
	// so pages only reference it in that case.
	hasSitemap := strings.TrimSpace(g.Meta.SiteURL) != ""

	// Render every page.
	for i, page := range flatPages {
		var prevLink, nextLink, prevTitle, nextTitle string
		if i > 0 {
			prevLink = flatPages[i-1].Filename
			prevTitle = flatPages[i-1].Title
		}
		if i < len(flatPages)-1 {
			nextLink = flatPages[i+1].Filename
			nextTitle = flatPages[i+1].Title
		}

		sitemapLink := ""
		if hasSitemap {
			sitemapLink = relativeSiteHref(page.Filename, "sitemap.xml")
		}
		sidebarHTML := g.buildSidebar(g.Chapters, page.Filename)
		data := pageData{
			SiteTitle:        g.Meta.Title,
			SiteSubtitle:     g.Meta.Subtitle,
			SiteDescription:  g.Meta.Description,
			Author:           g.Meta.Author,
			Language:         g.Meta.Language,
			ThemeName:        g.Meta.Theme,
			ThemeDescription: g.Meta.ThemeDescription,
			PageTitle:        page.Title,
			Description:      extractDescription(page.Content),
			Breadcrumbs:      resolveBreadcrumbs(g.buildBreadcrumbs(g.Chapters, page.Filename), page.Filename),
			Content:          page.Content,
			CSS:              g.CSS,
			SidebarHTML:      sidebarHTML,
			HomeLink:         relativeSiteHref(page.Filename, "index.html"),
			SitemapLink:      sitemapLink,
			PrevLink:         relativeSiteHref(page.Filename, prevLink),
			PrevTitle:        prevTitle,
			NextLink:         relativeSiteHref(page.Filename, nextLink),
			NextTitle:        nextTitle,
			EditLink:         g.editPageLink(page),
			ActiveFile:       page.Filename,
			TotalPages:       len(flatPages),
			CurrentPage:      i + 1,
			ShowTitle:        !contentStartsWithTitle(page.Content, page.Title),
		}
		populateUIStrings(&data)

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to render page %s: %w", page.Filename, err)
		}

		outPath := filepath.Join(outputDir, page.Filename)
		if err := utils.EnsureDir(filepath.Dir(outPath)); err != nil {
			return fmt.Errorf("failed to create page directory for %s: %w", page.Filename, err)
		}
		if err := os.WriteFile(outPath, []byte(buf.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", page.Filename, err)
		}

	}

	// Generate index.html as a full first-chapter page so the SPA loads instantly
	// at the site root without an HTTP redirect flicker.  The SPA router takes
	// over from there for subsequent navigation.
	var indexHTML string
	if len(flatPages) > 0 {
		firstPage := flatPages[0]
		// index.html shows the first chapter content with the sidebar active on
		// that chapter.  The "previous" nav link is omitted; "next" points to
		// the second page when it exists.
		var nextLink, nextTitle string
		if len(flatPages) > 1 {
			nextLink = flatPages[1].Filename
			nextTitle = flatPages[1].Title
		}
		idxSitemapLink := ""
		if hasSitemap {
			idxSitemapLink = "sitemap.xml"
		}
		idxData := pageData{
			SiteTitle:        g.Meta.Title,
			SiteSubtitle:     g.Meta.Subtitle,
			SiteDescription:  g.Meta.Description,
			Author:           g.Meta.Author,
			Language:         g.Meta.Language,
			ThemeName:        g.Meta.Theme,
			ThemeDescription: g.Meta.ThemeDescription,
			PageTitle:        firstPage.Title,
			Description:      firstNonEmpty(g.Meta.Description, extractDescription(firstPage.Content)),
			Breadcrumbs:      nil,
			Content:          firstPage.Content,
			CSS:              g.CSS,
			SidebarHTML:      g.buildSidebar(g.Chapters, "index.html"),
			HomeLink:         "index.html",
			SitemapLink:      idxSitemapLink,
			NextLink:         relativeSiteHref("index.html", nextLink),
			NextTitle:        nextTitle,
			EditLink:         g.editPageLink(firstPage),
			ActiveFile:       "index.html",
			TotalPages:       len(flatPages),
			CurrentPage:      1,
			ShowTitle:        !contentStartsWithTitle(firstPage.Content, firstPage.Title),
		}
		populateUIStrings(&idxData)
		var buf strings.Builder
		if err := tmpl.Execute(&buf, idxData); err != nil {
			return fmt.Errorf("failed to render index.html: %w", err)
		}
		indexHTML = buf.String()
	} else {
		indexHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>` +
			template.HTMLEscapeString(g.Meta.Title) +
			`</title></head><body><p>No chapters available.</p></body></html>`
	}
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		return fmt.Errorf("failed to write index.html: %w", err)
	}
	// Generate sitemap.xml for search engine indexing.  The sitemap protocol
	// requires fully-qualified URLs, so the file is only written when the
	// public base URL of the site is configured via output.site_url.
	if len(flatPages) > 0 && hasSitemap {
		base := strings.TrimRight(strings.TrimSpace(g.Meta.SiteURL), "/")
		lastMod := time.Now().Format("2006-01-02")
		var sm strings.Builder
		sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		sm.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		fmt.Fprintf(&sm, "  <url><loc>%s/</loc><lastmod>%s</lastmod><priority>1.0</priority></url>\n",
			template.HTMLEscapeString(base), lastMod)
		for _, page := range flatPages {
			fmt.Fprintf(&sm, "  <url><loc>%s/%s</loc><lastmod>%s</lastmod></url>\n",
				template.HTMLEscapeString(base), template.HTMLEscapeString(page.Filename), lastMod)
		}
		sm.WriteString("</urlset>\n")
		if err := os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), []byte(sm.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write sitemap.xml: %w", err)
		}
	} else if len(flatPages) > 0 {
		slog.Debug("Skipping sitemap.xml: output.site_url is not configured")
	}

	// Generate a 404 fallback page (served automatically by GitHub Pages,
	// Netlify, and similar hosts for unknown URLs).
	if err := g.generate404(outputDir); err != nil {
		return err
	}

	// Generate search-index.json for client-side full-text search.
	if len(flatPages) > 0 {
		entries := make([]searchEntry, 0, len(flatPages))
		for _, page := range flatPages {
			plainText := htmlTagPattern.ReplaceAllString(page.Content, " ")
			plainText = html.UnescapeString(plainText)
			plainText = strings.Join(strings.Fields(plainText), " ")
			if utf8.RuneCountInString(plainText) > maxSearchTextLength {
				plainText = string([]rune(plainText)[:maxSearchTextLength])
			}
			crumbs := g.buildBreadcrumbs(g.Chapters, page.Filename)
			var pathParts []string
			for _, c := range crumbs {
				if c.Filename != page.Filename {
					pathParts = append(pathParts, c.Title)
				}
			}
			pathStr := strings.Join(pathParts, " > ")
			entries = append(entries, searchEntry{
				Title:    page.Title,
				Filename: page.Filename,
				Text:     plainText,
				Path:     pathStr,
			})
			entries = append(entries, searchEntriesForHeadings(page, plainText)...)
		}
		indexJSON, err := json.Marshal(entries)
		if err != nil {
			return fmt.Errorf("failed to marshal search index: %w", err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "search-index.json"), indexJSON, 0o644); err != nil {
			return fmt.Errorf("failed to write search-index.json: %w", err)
		}
	}
	return nil
}

// flattenChapters expands nested chapters into a flat list.
func (g *SiteGenerator) flattenChapters(chapters []SiteChapter) []SiteChapter {
	var result []SiteChapter
	for _, ch := range chapters {
		flat := ch
		flat.Children = nil
		result = append(result, flat)
		if len(ch.Children) > 0 {
			result = append(result, g.flattenChapters(ch.Children)...)
		}
	}
	return result
}

// breadcrumb represents a navigation breadcrumb segment, containing the title
// and filename for one level in the breadcrumb trail.
type breadcrumb struct {
	Title    string
	Filename string
}

// pageData contains all the information needed to render a single page of the site,
// including site metadata, page content, navigation elements, and styling.
type pageData struct {
	SiteTitle        string
	SiteSubtitle     string
	SiteDescription  string
	Author           string
	Language         string
	ThemeName        string
	ThemeDescription string
	PageTitle        string
	Description      string // First ~160 chars of plain text for meta description.
	Breadcrumbs      []breadcrumb
	Content          string
	CSS              string
	SidebarHTML      string
	HomeLink         string
	SitemapLink      string
	PrevLink         string
	PrevTitle        string
	NextLink         string
	NextTitle        string
	EditLink         string // "Edit this page" URL; empty disables the link.
	ActiveFile       string
	TotalPages       int
	CurrentPage      int
	ShowTitle        bool // true when Content lacks an <h1>, so the template should insert one.

	// Localized UI strings.
	UIprevious          string
	UInext              string
	UIsearchPlaceholder string
	UIsearchButton      string
	UInoResults         string
	UIsearchUnavailable string
	UIsearchResultsOne  string
	UIsearchResults     string
	UIrecentPages       string
	UIrecentEmpty       string
	UIsearchNavigate    string
	UIsearchOpen        string
	UIsearchClose       string
	UIsearchMatchTitle  string
	UIsearchMatchPath   string
	UIsearchMatchText   string
	UIsearchMatched     string
	UIonThisPage        string
	UIeditPage          string
	UIcopy              string
	UIcopied            string
	UIhideSidebar       string
	UIlightMode         string
	UIdarkMode          string
	UIsystemDefault     string
	UIsearchKbd         string
	UIpageOf            string
	UIbuiltWith         string
}

type searchEntry struct {
	Title    string `json:"t"`
	Filename string `json:"f"`
	Text     string `json:"x"`
	Path     string `json:"p"`
}

func searchEntriesForHeadings(page SiteChapter, plainText string) []searchEntry {
	entries := make([]searchEntry, 0)
	// Convert to rune slice once for safe UTF-8 slicing across all headings.
	runeText := []rune(plainText)
	var walk func([]SiteNavHeading)
	walk = func(items []SiteNavHeading) {
		for _, item := range items {
			if item.ID != "" {
				// Include only a short snippet around the heading to avoid
				// duplicating the entire page text for every heading entry.
				snippet := item.Title
				if idx := strings.Index(plainText, item.Title); idx >= 0 {
					runeIdx := utf8.RuneCountInString(plainText[:idx])
					end := runeIdx + utf8.RuneCountInString(item.Title) + searchSnippetExtraRunes
					if end > len(runeText) {
						end = len(runeText)
					}
					snippet = string(runeText[runeIdx:end])
				}
				entries = append(entries, searchEntry{
					Title:    item.Title,
					Filename: page.Filename + "#" + item.ID,
					Text:     snippet,
					Path:     page.Title,
				})
			}
			walk(item.Children)
		}
	}
	walk(page.Headings)
	return entries
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// relativeSiteHref returns a href for target computed relative to the page at
// fromFile.  Both arguments are slash-separated paths from the site root.
// Relative hrefs keep the generated site fully relocatable: it works when
// served from a subdirectory (e.g. a GitHub Pages project site at
// https://user.github.io/repo/) and when opened directly via file://.  The
// SPA router rewrites the static sidebar links to absolute URLs at load time
// so they stay correct after client-side navigation changes the browser URL.
// Uses the path package (not filepath) since these are URL paths.
func relativeSiteHref(fromFile, target string) string {
	if target == "" {
		return ""
	}
	target = path.Clean(strings.TrimPrefix(target, "/"))
	fromDir := path.Dir(path.Clean(strings.TrimPrefix(fromFile, "/")))
	if fromDir == "." {
		return target
	}
	fromParts := strings.Split(fromDir, "/")
	targetParts := strings.Split(target, "/")
	// Skip the directory components shared by both paths, keeping at least
	// the target's final (file) component.
	common := 0
	for common < len(fromParts) && common < len(targetParts)-1 && fromParts[common] == targetParts[common] {
		common++
	}
	return strings.Repeat("../", len(fromParts)-common) + strings.Join(targetParts[common:], "/")
}

func resolveBreadcrumbs(crumbs []breadcrumb, fromFile string) []breadcrumb {
	if len(crumbs) == 0 {
		return nil
	}
	out := make([]breadcrumb, len(crumbs))
	for i, crumb := range crumbs {
		out[i] = breadcrumb{
			Title:    crumb.Title,
			Filename: relativeSiteHref(fromFile, crumb.Filename),
		}
	}
	return out
}

// editSourcePath returns the markdown source path used for the
// "edit this page" link.  It prefers the chapter's explicit SourcePath and
// falls back to deriving one from the page filename, mirroring how page
// filenames are produced from sources (index.html -> README.md).
func editSourcePath(ch SiteChapter) string {
	if ch.SourcePath != "" {
		return ch.SourcePath
	}
	f := strings.TrimPrefix(ch.Filename, "/")
	if f == "" || !strings.HasSuffix(f, ".html") {
		return ""
	}
	if path.Base(f) == "index.html" {
		dir := path.Dir(f)
		if dir == "." {
			return "README.md"
		}
		return dir + "/README.md"
	}
	return strings.TrimSuffix(f, ".html") + ".md"
}

// editPageLink joins the configured edit base URL with the chapter's markdown
// source path.  It returns "" when edit links are not configured or the
// source path cannot be determined.
func (g *SiteGenerator) editPageLink(ch SiteChapter) string {
	base := strings.TrimSpace(g.Meta.EditBase)
	if base == "" {
		return ""
	}
	src := editSourcePath(ch)
	if src == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(src, "/")
}

// generate404 writes a small standalone 404 page linking back to the site
// home.  Hosts may serve the page at any URL depth, so the home link uses the
// configured absolute site URL when available and falls back to a relative
// link that works at the site root.
func (g *SiteGenerator) generate404(outputDir string) error {
	homeLink := "index.html"
	if url := strings.TrimSpace(g.Meta.SiteURL); url != "" {
		homeLink = strings.TrimRight(url, "/") + "/"
	}
	tmpl, err := template.New("404").Parse(site404Template)
	if err != nil {
		return fmt.Errorf("failed to parse 404 template: %w", err)
	}
	data := struct {
		Language  string
		SiteTitle string
		Title     string
		HomeLink  string
		HomeLabel string
	}{
		Language:  g.Meta.Language,
		SiteTitle: g.Meta.Title,
		Title:     uiString(g.Meta.Language, "not_found_title"),
		HomeLink:  homeLink,
		HomeLabel: uiString(g.Meta.Language, "not_found_home"),
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to render 404.html: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "404.html"), []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("failed to write 404.html: %w", err)
	}
	return nil
}
