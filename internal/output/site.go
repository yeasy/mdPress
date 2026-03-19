// site.go generates a multi-page static site similar to GitBook.
// It includes sidebar navigation, previous/next links, search, and responsive layout.
package output

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/yeasy/mdpress/pkg/utils"
)

// SiteChapter stores rendered chapter data for site output.
type SiteChapter struct {
	Title    string
	ID       string
	Filename string // Output HTML filename, for example "ch01.html".
	Content  string // Rendered HTML content.
	Depth    int
	Headings []SiteNavHeading
	Children []SiteChapter
}

// SiteNavHeading stores an in-chapter navigation tree.
type SiteNavHeading struct {
	Title    string
	ID       string
	Children []SiteNavHeading
}

// SiteMeta stores site-wide metadata.
type SiteMeta struct {
	Title    string
	Author   string
	Language string
	Theme    string // CSS theme name.
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

// AddChapter appends a chapter.
func (g *SiteGenerator) AddChapter(ch SiteChapter) {
	g.Chapters = append(g.Chapters, ch)
}

// SetCSS sets the site CSS.
func (g *SiteGenerator) SetCSS(css string) {
	g.CSS = css
}

// Generate writes the static site to the output directory.
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
	}

	// Parse the page template.
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
		"safeCSS":  func(s string) template.CSS { return template.CSS(s) },
	}).Parse(sitePageTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse page template: %w", err)
	}

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

		sidebarHTML := g.buildSidebar(g.Chapters, page.Filename)

		data := pageData{
			SiteTitle:   g.Meta.Title,
			Author:      g.Meta.Author,
			Language:    g.Meta.Language,
			PageTitle:   page.Title,
			Content:     page.Content,
			CSS:         g.CSS,
			SidebarHTML: sidebarHTML,
			PrevLink:    prevLink,
			PrevTitle:   prevTitle,
			NextLink:    nextLink,
			NextTitle:   nextTitle,
			ActiveFile:  page.Filename,
			TotalPages:  len(flatPages),
			CurrentPage: i + 1,
		}

		var buf strings.Builder
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to render page %s: %w", page.Filename, err)
		}

		outPath := filepath.Join(outputDir, page.Filename)
		if err := os.WriteFile(outPath, []byte(buf.String()), 0644); err != nil {
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
		idxData := pageData{
			SiteTitle:   g.Meta.Title,
			Author:      g.Meta.Author,
			Language:    g.Meta.Language,
			PageTitle:   firstPage.Title,
			Content:     firstPage.Content,
			CSS:         g.CSS,
			SidebarHTML: g.buildSidebar(g.Chapters, firstPage.Filename),
			NextLink:    nextLink,
			NextTitle:   nextTitle,
			ActiveFile:  firstPage.Filename,
			TotalPages:  len(flatPages),
			CurrentPage: 1,
		}
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
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		return fmt.Errorf("failed to write index.html: %w", err)
	}

	return nil
}

// flattenChapters expands nested chapters into a flat list.
func (g *SiteGenerator) flattenChapters(chapters []SiteChapter) []SiteChapter {
	var result []SiteChapter
	for _, ch := range chapters {
		result = append(result, SiteChapter{
			Title:    ch.Title,
			ID:       ch.ID,
			Filename: ch.Filename,
			Content:  ch.Content,
		})
		if len(ch.Children) > 0 {
			result = append(result, g.flattenChapters(ch.Children)...)
		}
	}
	return result
}

// buildSidebar renders the sidebar navigation HTML.
func (g *SiteGenerator) buildSidebar(chapters []SiteChapter, activeFile string) string {
	var b strings.Builder
	g.renderSidebarItems(&b, chapters, activeFile)
	return b.String()
}

func (g *SiteGenerator) renderSidebarItems(b *strings.Builder, chapters []SiteChapter, activeFile string) {
	for _, ch := range chapters {
		filename := ch.Filename
		if filename == "" {
			filename = "#"
		}
		groupClass := "nav-group"
		hasChildren := len(ch.Headings) > 0 || len(ch.Children) > 0
		if hasChildren {
			if g.isChapterBranchActive(ch, activeFile) {
				groupClass += " expanded"
			} else {
				groupClass += " collapsed"
			}
		}

		fmt.Fprintf(b, `<div class="%s" data-group-file="%s">`, groupClass, template.HTMLEscapeString(filename))
		b.WriteString(`<div class="nav-row">`)
		if hasChildren {
			expanded := "false"
			if g.isChapterBranchActive(ch, activeFile) {
				expanded = "true"
			}
			fmt.Fprintf(b, `<button class="nav-toggle" type="button" aria-label="Toggle section" aria-expanded="%s"></button>`, expanded)
		} else {
			b.WriteString(`<span class="nav-toggle nav-toggle-placeholder"></span>`)
		}
		fmt.Fprintf(b,
			`<a class="nav-item nav-chapter nav-depth-%d" href="%s" data-file="%s" data-group-link="%t">%s</a>`,
			ch.Depth+1,
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(filename),
			hasChildren,
			template.HTMLEscapeString(ch.Title))
		b.WriteString(`</div>`)

		if hasChildren {
			b.WriteString(`<div class="nav-children">`)
			b.WriteString(`<div class="nav-children-inner">`)
			if len(ch.Headings) > 0 {
				g.renderSidebarHeadings(b, ch.Filename, ch.Headings, 0)
			}
			if len(ch.Children) > 0 {
				g.renderSidebarItems(b, ch.Children, activeFile)
			}
			b.WriteString(`</div>`)
			b.WriteString(`</div>`)
		}
		b.WriteString(`</div>` + "\n")
	}
}

func (g *SiteGenerator) isChapterBranchActive(ch SiteChapter, activeFile string) bool {
	if ch.Filename == activeFile {
		return true
	}
	for _, child := range ch.Children {
		if g.isChapterBranchActive(child, activeFile) {
			return true
		}
	}
	return false
}

func (g *SiteGenerator) renderSidebarHeadings(b *strings.Builder, filename string, headings []SiteNavHeading, depth int) {
	for _, heading := range headings {
		fmt.Fprintf(b,
			`<a class="nav-item nav-heading nav-heading-depth-%d" href="%s#%s" data-file="%s" data-target="%s">%s</a>`,
			depth+1,
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(filename),
			template.HTMLEscapeString(heading.ID),
			template.HTMLEscapeString(heading.Title))
		if len(heading.Children) > 0 {
			g.renderSidebarHeadings(b, filename, heading.Children, depth+1)
		}
	}
}

type pageData struct {
	SiteTitle   string
	Author      string
	Language    string
	PageTitle   string
	Content     string
	CSS         string
	SidebarHTML string
	PrevLink    string
	PrevTitle   string
	NextLink    string
	NextTitle   string
	ActiveFile  string
	TotalPages  int
	CurrentPage int
}

var sitePageTemplate = `<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.PageTitle}} - {{.SiteTitle}}</title>
<style>
@view-transition {
  navigation: auto;
}

/* ===== Reset & Base ===== */
* { box-sizing: border-box; margin: 0; padding: 0; }
html { font-size: 16px; scroll-behavior: smooth; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", "Helvetica Neue", Arial, sans-serif;
  line-height: 1.7; color: #333; background: #fff;
  display: flex; min-height: 100vh;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 280px; min-width: 280px;
  background: #fafafa; border-right: 1px solid #e8e8e8;
  padding: 20px 0; overflow-y: auto;
  position: fixed; top: 0; bottom: 0; left: 0; z-index: 100;
  transition: transform 0.3s;
  view-transition-name: site-sidebar;
}
.sidebar-header {
  padding: 10px 20px 20px; border-bottom: 1px solid #e8e8e8;
  margin-bottom: 10px;
}
.sidebar-header h1 {
  font-size: 1.1rem; color: #333; font-weight: 600; line-height: 1.3;
}
.sidebar-header .author { font-size: 0.8rem; color: #999; margin-top: 4px; }
.sidebar-nav { padding: 0 10px; }

.nav-group { margin: 2px 0; }
.nav-row {
  display: flex; align-items: center; gap: 4px;
  padding-right: 8px;
}
.nav-toggle {
  width: 22px; height: 22px; border: none; background: transparent;
  color: #666; cursor: pointer; border-radius: 4px; flex: 0 0 22px;
}
.nav-toggle::before {
  content: "▾"; display: block; font-size: 0.72rem;
  transition: transform 0.2s ease;
}
.nav-group.collapsed .nav-toggle::before { transform: rotate(-90deg); }
.nav-toggle-placeholder::before { content: ""; }
.nav-item {
  display: block; color: #555; text-decoration: none;
  font-size: 0.9rem; border-radius: 4px; margin: 1px 0; transition: all 0.15s;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.nav-item:hover { background: #e8e8e8; color: #111; }
.nav-item.active { background: #4285f4; color: #fff; font-weight: 500; }
.nav-chapter { flex: 1; padding: 6px 10px 6px 8px; font-weight: 600; }
.nav-heading { padding: 5px 12px; font-size: 0.84rem; margin-left: 26px; }
.nav-depth-1 { padding-left: 8px; }
.nav-depth-2 { padding-left: 22px; }
.nav-depth-3 { padding-left: 36px; }
.nav-depth-4 { padding-left: 50px; }
.nav-heading-depth-2 { padding-left: 26px; }
.nav-heading-depth-3 { padding-left: 40px; font-size: 0.8rem; }
.nav-heading-depth-4 { padding-left: 54px; font-size: 0.78rem; }
.nav-children {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transition: grid-template-rows 0.24s ease, opacity 0.18s ease;
}
.nav-children-inner {
  min-height: 0;
  overflow: hidden;
  padding-bottom: 2px;
}
.nav-group.expanded > .nav-children {
  grid-template-rows: 1fr;
  opacity: 1;
}

/* ===== Main Content ===== */
.main {
  margin-left: 280px; flex: 1; min-width: 0;
  view-transition-name: site-main;
}
.content {
  max-width: 860px; margin: 0 auto; padding: 40px 50px 80px; overflow-wrap: anywhere;
  view-transition-name: site-content;
  transition: opacity 0.18s ease, transform 0.22s ease;
}
.content.is-navigating {
  opacity: 0.72;
  transform: translate3d(0, 8px, 0);
  pointer-events: none;
}
.route-progress {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 3px;
  z-index: 320;
  pointer-events: none;
}
.route-progress-bar {
  width: 0;
  height: 100%;
  opacity: 0;
  background: linear-gradient(90deg, #4285f4 0%, #74a7ff 100%);
  box-shadow: 0 0 14px rgba(66, 133, 244, 0.35);
  transition: width 0.22s ease, opacity 0.18s ease;
}
.route-progress.is-active .route-progress-bar {
  width: 68%;
  opacity: 1;
}
.route-progress.is-finishing .route-progress-bar {
  width: 100%;
  opacity: 1;
}
.content h1[id], .content h2[id], .content h3[id], .content h4[id], .content h5[id], .content h6[id] {
  scroll-margin-top: 24px;
}
.content h1 { font-size: 2em; margin: 0 0 0.8em; color: #1a1a2e; border-bottom: 2px solid #4285f4; padding-bottom: 0.3em; }
.content h2 { font-size: 1.5em; margin: 1.5em 0 0.6em; color: #333; }
.content h3 { font-size: 1.2em; margin: 1.3em 0 0.5em; color: #444; }
.content h4 { font-size: 1.05em; margin: 1em 0 0.4em; color: #555; }
.content p { margin: 0.6em 0; text-align: justify; }
.content img { max-width: 100%; height: auto; display: block; margin: 1em auto; border-radius: 4px; }
.content blockquote {
  border-left: 4px solid #4285f4; background: #f4f7ff; margin: 1em 0;
  padding: 12px 16px; color: #555; border-radius: 0 4px 4px 0;
}
.content blockquote p { margin: 0.3em 0; }
.content code {
  background: #f0f0f0; padding: 2px 6px; border-radius: 3px;
  font-family: "Fira Code", "Consolas", "Monaco", monospace; font-size: 0.9em;
  overflow-wrap: anywhere; word-break: break-word;
}
.content pre {
  background: #2d2d2d; color: #f8f8f2; padding: 16px 20px;
  border-radius: 6px; overflow-x: auto; margin: 1em 0; line-height: 1.5;
  font-size: 0.88em; white-space: pre; word-break: normal;
}
.content pre code { background: transparent; color: inherit; padding: 0; font-size: inherit; display: block; }
.content table { border-collapse: collapse; width: 100%; margin: 1em 0; table-layout: auto; }
.content th, .content td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; overflow-wrap: anywhere; word-break: break-word; }
.content th { background: #f5f5f5; font-weight: 600; }
.content tr:nth-child(even) { background: #fafafa; }
.content a { color: #4285f4; text-decoration: none; }
.content a:hover { text-decoration: underline; }
.content ul, .content ol { padding-left: 1.8em; margin: 0.5em 0; }
.content li { margin: 0.3em 0; }
.content hr { border: none; height: 1px; background: #e0e0e0; margin: 2em 0; }

/* ===== Glossary terms ===== */
.glossary-term {
  border-bottom: 1px dashed #4285f4; cursor: help;
}

/* ===== Page Navigation ===== */
.page-nav {
  display: flex; justify-content: space-between; margin-top: 3em;
  padding-top: 1.5em; border-top: 1px solid #e8e8e8;
}
.page-nav a {
  color: #4285f4; text-decoration: none; font-size: 0.95em;
  display: flex; align-items: center; gap: 6px; max-width: 45%;
}
.page-nav a:hover { text-decoration: underline; }
.page-nav .prev::before { content: "← "; }
.page-nav .next::after { content: " →"; }

/* ===== Build Meta ===== */
.build-meta {
  margin-top: 2.5rem;
  padding-top: 1rem;
  border-top: 1px solid #e8e8e8;
  color: #999;
  font-size: 0.82rem;
  text-align: center;
}
.build-meta a {
  color: inherit;
  text-decoration: none;
}
.build-meta a:hover {
  text-decoration: underline;
}

/* ===== Page Transition ===== */
@keyframes mdpress-page-out {
  from {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
  to {
    opacity: 0;
    transform: translate3d(-14px, 0, 0);
  }
}

@keyframes mdpress-page-in {
  from {
    opacity: 0;
    transform: translate3d(18px, 0, 0);
  }
  to {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
}

@keyframes mdpress-sidebar-in {
  from {
    opacity: 0.84;
    transform: translate3d(-8px, 0, 0);
  }
  to {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
}

::view-transition-old(site-content) {
  animation: mdpress-page-out 180ms cubic-bezier(0.4, 0, 1, 1) both;
}

::view-transition-new(site-content) {
  animation: mdpress-page-in 260ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

::view-transition-old(site-main),
::view-transition-new(site-main) {
  animation-duration: 220ms;
}

::view-transition-old(site-sidebar),
::view-transition-new(site-sidebar) {
  animation-duration: 180ms;
}

body.page-entering .content {
  animation: mdpress-page-in 260ms cubic-bezier(0.22, 1, 0.36, 1);
}

body.page-entering .sidebar {
  animation: mdpress-sidebar-in 220ms ease-out;
}

/* ===== Mobile Toggle ===== */
.sidebar-toggle {
  display: none; position: fixed; top: 12px; left: 12px; z-index: 200;
  background: #4285f4; color: #fff; border: none; border-radius: 4px;
  width: 36px; height: 36px; font-size: 1.2rem; cursor: pointer;
  align-items: center; justify-content: center;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .sidebar { transform: translateX(-100%); }
  .sidebar.open { transform: translateX(0); box-shadow: 2px 0 8px rgba(0,0,0,.15); }
  .sidebar-toggle { display: flex; }
  .main { margin-left: 0; }
  .content { padding: 50px 20px 60px; }
}

@media (prefers-reduced-motion: reduce) {
  html { scroll-behavior: auto; }
  .sidebar, .nav-toggle::before, .nav-children, .nav-item { transition: none; }
  .sidebar, .main, .content { view-transition-name: none; }
  body.page-entering .content,
  body.page-entering .sidebar,
  ::view-transition-old(site-content),
  ::view-transition-new(site-content),
  ::view-transition-old(site-main),
  ::view-transition-new(site-main),
  ::view-transition-old(site-sidebar),
  ::view-transition-new(site-sidebar) {
    animation: none;
  }
}

/* ===== Custom Theme CSS ===== */
{{safeCSS .CSS}}

/* ===== Site Layout Overrides ===== */
body {
  margin: 0 !important;
  padding: 0 !important;
}
</style>
</head>
<body>
  <div class="route-progress" id="route-progress" aria-hidden="true">
    <div class="route-progress-bar"></div>
  </div>
  <button class="sidebar-toggle" onclick="document.querySelector('.sidebar').classList.toggle('open')">☰</button>

  <nav class="sidebar">
    <div class="sidebar-header">
      <h1>{{.SiteTitle}}</h1>
      {{if .Author}}<div class="author">{{.Author}}</div>{{end}}
    </div>
    <div class="sidebar-nav">
      {{safeHTML .SidebarHTML}}
    </div>
  </nav>

  <div class="main">
    <div class="content">
      {{safeHTML .Content}}

      <nav class="page-nav">
        {{if .PrevLink}}<a class="prev" href="{{.PrevLink}}">{{.PrevTitle}}</a>{{else}}<span></span>{{end}}
        {{if .NextLink}}<a class="next" href="{{.NextLink}}">{{.NextTitle}}</a>{{else}}<span></span>{{end}}
      </nav>

      <div class="build-meta">
        <a href="https://github.com/yeasy/mdpress" target="_blank" rel="noopener noreferrer">Built with mdpress</a>
      </div>
    </div>
  </div>

  <script>
  var sidebar = document.querySelector('.sidebar');
  var body = document.body;
  var mainContent = document.querySelector('.content');
  var routeProgress = document.getElementById('route-progress');
  var prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var navUpdateFrame = null;
  var scrollSaveFrame = null;
  var lastActiveLink = null;
  var prefetchedPages = Object.create(null);
  var pageCache = Object.create(null);
  var pendingNavigation = null;
  var internalNavStateKey = 'mdpress-site-nav';
  var scrollStoreKey = 'mdpress-site-scroll';
  var currentFile = '{{.ActiveFile}}';
  var navLinksByCurrentFile = [];
  var navChapterLinks = [];
  var navHeadingLinks = [];
  var headings = [];

  try {
    var navState = window.sessionStorage.getItem(internalNavStateKey);
    if (navState && !prefersReducedMotion) {
      var parsedState = JSON.parse(navState);
      if (parsedState && typeof parsedState.ts === 'number' && (Date.now() - parsedState.ts) < 4000) {
        body.classList.add('page-entering');
        window.requestAnimationFrame(function() {
          window.requestAnimationFrame(function() {
            body.classList.remove('page-entering');
          });
        });
      }
    }
    window.sessionStorage.removeItem(internalNavStateKey);
  } catch (e) {}

  function getInternalPageURL(href) {
    if (!href) return null;
    try {
      var url = new URL(href, window.location.href);
      if (url.origin !== window.location.origin) return null;
      if (url.pathname === window.location.pathname) return null;
      if (!/\.html$/i.test(url.pathname)) return null;
      url.hash = '';
      return url.toString();
    } catch (e) {
      return null;
    }
  }

  function prefetchPage(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL || prefetchedPages[pageURL]) return;
    prefetchedPages[pageURL] = true;

    var link = document.createElement('link');
    link.rel = 'prefetch';
    link.href = pageURL;
    link.as = 'document';
    document.head.appendChild(link);
  }

  function warmPageCache(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL) return;
    var targetURL = new URL(pageURL, window.location.href);
    fetchPagePayload(targetURL).catch(function() {});
  }

  function rememberInternalNavigation(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL) return;
    try {
      window.sessionStorage.setItem(internalNavStateKey, JSON.stringify({
        ts: Date.now(),
        href: pageURL
      }));
    } catch (e) {}
  }

  function getFileFromPathname(pathname) {
    if (!pathname) return currentFile || 'index.html';
    var clean = pathname.replace(/\/+$/, '');
    var segments = clean.split('/');
    var last = segments[segments.length - 1];
    return last || 'index.html';
  }

  function refreshPageContext() {
    navLinksByCurrentFile = Array.from(document.querySelectorAll('.nav-item[data-file="' + currentFile + '"]'));
    navChapterLinks = Array.from(document.querySelectorAll('.nav-chapter[data-file="' + currentFile + '"]'));
    navHeadingLinks = Array.from(document.querySelectorAll('.nav-heading[data-file="' + currentFile + '"]'));
    headings = Array.from(document.querySelectorAll('.content h1[id], .content h2[id], .content h3[id], .content h4[id], .content h5[id], .content h6[id]'));
  }

  function setNavigating(isNavigating) {
    if (!mainContent) return;
    mainContent.classList.toggle('is-navigating', isNavigating);
  }

  function beginRouteProgress() {
    if (!routeProgress || prefersReducedMotion) return;
    routeProgress.classList.remove('is-finishing');
    routeProgress.classList.add('is-active');
  }

  function endRouteProgress() {
    if (!routeProgress || prefersReducedMotion) return;
    routeProgress.classList.remove('is-active');
    routeProgress.classList.add('is-finishing');
    window.setTimeout(function() {
      routeProgress.classList.remove('is-finishing');
    }, 220);
  }

  function readScrollStore() {
    try {
      return JSON.parse(window.sessionStorage.getItem(scrollStoreKey) || '{}');
    } catch (e) {
      return {};
    }
  }

  function writeScrollStore(store) {
    try {
      window.sessionStorage.setItem(scrollStoreKey, JSON.stringify(store));
    } catch (e) {}
  }

  function saveScrollPosition(pathname) {
    if (!pathname) return;
    var store = readScrollStore();
    store[pathname] = window.scrollY || window.pageYOffset || 0;
    writeScrollStore(store);
  }

  function getSavedScrollPosition(pathname) {
    if (!pathname) return null;
    var store = readScrollStore();
    return typeof store[pathname] === 'number' ? store[pathname] : null;
  }

  function expandGroupChain(group) {
    var current = group;
    while (current) {
      setGroupExpanded(current, true);
      current = current.parentElement ? current.parentElement.closest('.nav-group') : null;
    }
  }

  function setGroupExpanded(group, shouldExpand) {
    if (!group || !group.querySelector('.nav-children')) return;
    group.classList.toggle('collapsed', !shouldExpand);
    group.classList.toggle('expanded', shouldExpand);
    var toggle = group.querySelector('.nav-toggle');
    if (toggle) toggle.setAttribute('aria-expanded', shouldExpand ? 'true' : 'false');
  }

  function toggleGroup(group) {
    if (!group) return;
    setGroupExpanded(group, group.classList.contains('collapsed'));
  }

  function smoothScrollToElement(element, hash) {
    if (!element) return;
    element.scrollIntoView({
      behavior: prefersReducedMotion ? 'auto' : 'smooth',
      block: 'start'
    });
    if (hash) {
      window.history.pushState(null, '', hash);
    }
  }

  function scrollToHashTarget(hash, shouldPushHistory) {
    if (!hash) {
      window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
      return;
    }

    var targetId = hash.charAt(0) === '#' ? hash.slice(1) : hash;
    var target = targetId ? document.getElementById(targetId) : null;
    if (!target) {
      if (shouldPushHistory) {
        window.history.pushState(null, '', '#' + targetId);
      }
      return;
    }

    target.scrollIntoView({
      behavior: prefersReducedMotion ? 'auto' : 'smooth',
      block: 'start'
    });
    if (shouldPushHistory) {
      window.history.pushState(null, '', '#' + targetId);
    }
  }

  function keepActiveLinkVisible(link) {
    if (!link || !sidebar) return;
    if (lastActiveLink === link) return;
    lastActiveLink = link;
    link.scrollIntoView({
      block: 'nearest',
      behavior: prefersReducedMotion ? 'auto' : 'smooth'
    });
  }

  document.querySelectorAll('.nav-group').forEach(function(group) {
    var toggle = group.querySelector('.nav-toggle');
    var chapterLink = group.querySelector('.nav-chapter[data-group-link="true"]');

    if (toggle) {
      toggle.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        toggleGroup(group);
      });
    }

    if (chapterLink) {
      chapterLink.addEventListener('pointerenter', function() {
        prefetchPage(chapterLink.href);
      }, { passive: true });
      chapterLink.addEventListener('focus', function() {
        prefetchPage(chapterLink.href);
      });
      chapterLink.addEventListener('touchstart', function() {
        prefetchPage(chapterLink.href);
      }, { passive: true });
      chapterLink.addEventListener('click', function(e) {
        expandGroupChain(group);
        rememberInternalNavigation(chapterLink.href);
        if (chapterLink.getAttribute('data-file') === currentFile) {
          e.preventDefault();
          window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
        }
      });
    }
  });

  function findActiveHeading() {
    var candidate = null;
    var threshold = 140;

    for (var i = 0; i < headings.length; i++) {
      var top = headings[i].getBoundingClientRect().top;
      if (top <= threshold) {
        candidate = headings[i].id;
      } else {
        break;
      }
    }

    return candidate;
  }

  function updateActiveNavigation() {
    var activeHeading = findActiveHeading();

    document.querySelectorAll('.nav-item.active').forEach(function(link) {
      link.classList.remove('active');
    });

    var headingMatched = false;
    if (activeHeading) {
      for (var j = 0; j < navHeadingLinks.length; j++) {
        if (navHeadingLinks[j].getAttribute('data-target') === activeHeading) {
          navHeadingLinks[j].classList.add('active');
          var activeGroup = navHeadingLinks[j].closest('.nav-group');
          expandGroupChain(activeGroup);
          keepActiveLinkVisible(navHeadingLinks[j]);
          headingMatched = true;
          break;
        }
      }
    }

    if (!headingMatched && navChapterLinks.length > 0) {
      navChapterLinks[0].classList.add('active');
      var activeChapterGroup = navChapterLinks[0].closest('.nav-group');
      expandGroupChain(activeChapterGroup);
      keepActiveLinkVisible(navChapterLinks[0]);
    }
  }

  function syncSidebarForCurrentFile() {
    document.querySelectorAll('.nav-group[data-group-file]').forEach(function(group) {
      if (group.getAttribute('data-group-file') === currentFile) {
        expandGroupChain(group);
      }
    });
  }

  function scheduleNavigationUpdate() {
    if (navUpdateFrame !== null) return;
    navUpdateFrame = window.requestAnimationFrame(function() {
      updateActiveNavigation();
      navUpdateFrame = null;
    });
  }

  window.addEventListener('scroll', scheduleNavigationUpdate, { passive: true });
  window.addEventListener('resize', scheduleNavigationUpdate);
  window.addEventListener('hashchange', function() {
    scheduleNavigationUpdate();
  });

  // loadCDNScript is a helper to load a script from CDN only once.
  // tag: data attribute name used to deduplicate; src: CDN URL; onReady: callback.
  function loadCDNScript(tag, src, onReady) {
    var attrName = 'data-mdpress-' + tag.replace(/[A-Z]/g, function(ch) {
      return '-' + ch.toLowerCase();
    });
    var existing = document.querySelector('script[' + attrName + ']');
    if (existing) {
      if (existing.dataset.mdpressLoaded === 'true') {
        if (onReady) onReady();
      } else if (onReady) {
        existing.addEventListener('load', onReady, { once: true });
      }
      return;
    }

    var s = document.createElement('script');
    s.src = src;
    s.setAttribute(attrName, 'true');
    s.addEventListener('load', function() {
      s.dataset.mdpressLoaded = 'true';
      if (onReady) onReady();
    }, { once: true });
    document.body.appendChild(s);
  }

  function ensureMermaid() {
    var nodes = document.querySelectorAll('.mermaid');
    if (!nodes.length) return;

    function runMermaid() {
      if (!window.mermaid) return;
      try {
        window.mermaid.initialize({ startOnLoad: true, theme: 'default' });
        if (window.mermaid.run) {
          window.mermaid.run({ nodes: nodes });
        } else if (window.mermaid.init) {
          window.mermaid.init(undefined, nodes);
        }
      } catch (e) {
        console.warn('[mdpress] Mermaid re-init failed', e);
      }
    }

    if (window.mermaid) { runMermaid(); return; }
    loadCDNScript('mermaid', '` + utils.MermaidCDNURL + `', runMermaid);
  }

  // ensureKaTeX loads KaTeX and triggers auto-render when math elements are found.
  // Called on initial load and after each client-side navigation.
  function ensureKaTeX() {
    if (!document.querySelector('.math')) return;

    function runKaTeX() {
      if (typeof renderMathInElement !== 'function') return;
      try {
        renderMathInElement(document.body, {
          delimiters: [
            {left: '$$', right: '$$', display: true},
            {left: '$',  right: '$',  display: false}
          ],
          throwOnError: false
        });
      } catch (e) {
        console.warn('[mdpress] KaTeX render failed', e);
      }
    }

    if (typeof renderMathInElement === 'function') { runKaTeX(); return; }

    // Load KaTeX CSS if not already loaded.
    if (!document.querySelector('link[data-mdpress-katex-css]')) {
      var link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = '` + utils.KaTeXCSSURL + `';
      link.dataset.mdpressKatexCss = 'true';
      document.head.appendChild(link);
    }

    loadCDNScript('katex', '` + utils.KaTeXJSURL + `', function() {
      loadCDNScript('katexAutoRender', '` + utils.KaTeXAutoRenderURL + `', runKaTeX);
    });
  }

  function getClientNavigation(anchor) {
    if (!anchor || !anchor.href) return null;
    try {
      var url = new URL(anchor.href, window.location.href);
      if (url.origin !== window.location.origin) return null;
      if (!/\.html$/i.test(url.pathname) && url.pathname !== window.location.pathname) return null;
      return {
        url: url,
        file: getFileFromPathname(url.pathname),
        hash: url.hash || ''
      };
    } catch (e) {
      return null;
    }
  }

  function parseFetchedPage(html, fallbackURL) {
    var doc = new DOMParser().parseFromString(html, 'text/html');
    var content = doc.querySelector('.content');
    if (!content) return null;
    return {
      title: doc.title || document.title,
      contentHTML: content.innerHTML,
      url: fallbackURL
    };
  }

  function getCachedPage(cacheKey) {
    return pageCache[cacheKey] || null;
  }

  function cachePage(cacheKey, payload) {
    pageCache[cacheKey] = payload;
    return payload;
  }

  function fetchPagePayload(targetURL, signal) {
    var cacheKey = targetURL.origin + targetURL.pathname;
    var cached = getCachedPage(cacheKey);
    if (cached) return Promise.resolve(cached);

    return fetch(cacheKey, {
      credentials: 'same-origin',
      signal: signal
    }).then(function(response) {
      if (!response.ok) throw new Error('HTTP ' + response.status);
      return response.text().then(function(html) {
        var responseURL = new URL(response.url || cacheKey, window.location.href);
        var payload = parseFetchedPage(html, responseURL);
        if (!payload) throw new Error('Missing .content in fetched page');
        return cachePage(cacheKey, payload);
      });
    });
  }

  function finalizeNavigation(targetURL, options) {
    currentFile = getFileFromPathname(targetURL.pathname);
    refreshPageContext();
    syncSidebarForCurrentFile();
    updateActiveNavigation();
    ensureMermaid();
    ensureKaTeX();

    if (window.innerWidth <= 768) {
      sidebar.classList.remove('open');
    }

    if (options.updateHistory === 'push') {
      window.history.pushState({ path: targetURL.pathname }, '', targetURL.pathname + targetURL.search + targetURL.hash);
    } else if (options.updateHistory === 'replace') {
      window.history.replaceState({ path: targetURL.pathname }, '', targetURL.pathname + targetURL.search + targetURL.hash);
    }

    if (options.hash) {
      scrollToHashTarget(options.hash, false);
    } else if (options.restoreScroll === true) {
      var savedScroll = getSavedScrollPosition(targetURL.pathname);
      window.scrollTo({
        top: savedScroll || 0,
        behavior: prefersReducedMotion ? 'auto' : 'auto'
      });
    } else if (options.scrollToTop !== false) {
      window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
    }
  }

  function swapPageContent(payload, targetURL, options) {
    function applySwap() {
      mainContent.innerHTML = payload.contentHTML;
      document.title = payload.title;
    }

    if (document.startViewTransition && !prefersReducedMotion) {
      var transition = document.startViewTransition(applySwap);
      return transition.finished.catch(function() {}).then(function() {
        finalizeNavigation(targetURL, options);
      });
    }

    body.classList.add('page-entering');
    applySwap();
    finalizeNavigation(targetURL, options);
    return Promise.resolve().then(function() {
      window.requestAnimationFrame(function() {
        window.requestAnimationFrame(function() {
          body.classList.remove('page-entering');
        });
      });
    });
  }

  function navigateClientSide(target, options) {
    options = options || {};
    if (!mainContent) {
      window.location.href = target.url.toString();
      return Promise.resolve();
    }

    if (pendingNavigation) {
      pendingNavigation.abort();
    }
    saveScrollPosition(window.location.pathname);
    pendingNavigation = new AbortController();
    setNavigating(true);
    beginRouteProgress();

    return fetchPagePayload(target.url, pendingNavigation.signal)
      .then(function(payload) {
        if (pendingNavigation.signal.aborted) return;
        var targetURL = new URL(target.url.toString(), window.location.href);
        return swapPageContent(payload, targetURL, {
          updateHistory: options.updateHistory || 'push',
          hash: target.hash,
          scrollToTop: options.scrollToTop,
          restoreScroll: options.restoreScroll === true
        });
      })
      .catch(function(err) {
        if (err && err.name === 'AbortError') return;
        console.warn('[mdpress] Falling back to full navigation', err);
        window.location.href = target.url.toString();
      })
      .finally(function() {
        setNavigating(false);
        endRouteProgress();
      });
  }

  refreshPageContext();
  window.history.replaceState({ path: window.location.pathname }, '', window.location.pathname + window.location.search + window.location.hash);
  syncSidebarForCurrentFile();
  updateActiveNavigation();
  ensureMermaid();
  ensureKaTeX();

  document.addEventListener('mouseover', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  }, { passive: true });

  document.addEventListener('focusin', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  });

  document.addEventListener('touchstart', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  }, { passive: true });

  document.addEventListener('click', function(e) {
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    if (link.target && link.target !== '_self') return;
    if (link.hasAttribute('download')) return;

    var target = getClientNavigation(link);
    if (!target) return;

    rememberInternalNavigation(link.href);

    if (target.file === currentFile) {
      if (target.hash) {
        var samePageTarget = document.getElementById(target.hash.slice(1));
        if (samePageTarget) {
          e.preventDefault();
          expandGroupChain(link.closest('.nav-group'));
          scrollToHashTarget(target.hash, true);
          scheduleNavigationUpdate();
        }
      } else if (target.url.pathname === window.location.pathname) {
        e.preventDefault();
        window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
      }
      return;
    }

    e.preventDefault();
    expandGroupChain(link.closest('.nav-group'));
    navigateClientSide(target, {
      updateHistory: 'push',
      scrollToTop: !target.hash
    });
  });

  window.addEventListener('popstate', function() {
    var target = getClientNavigation({ href: window.location.href });
    if (!target) return;
    if (target.file === currentFile) {
      if (target.hash) {
        scrollToHashTarget(target.hash, false);
      } else {
        window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
      }
      scheduleNavigationUpdate();
      return;
    }
    navigateClientSide(target, {
      updateHistory: null,
      scrollToTop: !target.hash,
      restoreScroll: !target.hash
    });
  });

  window.addEventListener('scroll', function() {
    if (scrollSaveFrame !== null) return;
    scrollSaveFrame = window.requestAnimationFrame(function() {
      saveScrollPosition(window.location.pathname);
      scrollSaveFrame = null;
    });
  }, { passive: true });
  </script>
</body>
</html>`
