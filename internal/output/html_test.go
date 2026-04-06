package output

import (
	"archive/zip"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

func TestHTMLGeneratorBasic(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "site")

	gen := NewHTMLGenerator()
	err := gen.Generate("<html><body>Hello</body></html>", outDir, nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	indexPath := filepath.Join(outDir, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index.html failed: %v", err)
	}
	if string(data) != "<html><body>Hello</body></html>" {
		t.Error("index.html content mismatch")
	}
}

func TestHTMLGeneratorWithChapters(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "site")

	chapters := map[string]string{
		"chapter-1": "<h1>Ch1</h1>",
		"chapter-2": "<h1>Ch2</h1>",
	}

	gen := NewHTMLGenerator()
	err := gen.Generate("<html></html>", outDir, chapters)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check chapter files exist
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	if len(entries) < 3 { // index.html + 2 chapters
		t.Errorf("expected at least 3 files, got %d", len(entries))
	}
}

func TestSiteGeneratorNestedSidebar(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})
	// Use Children (sub-chapters) to trigger nav-group with expand/collapse.
	// With maxSidebarHeadingDepth = 0, only Children create collapsible groups.
	gen.AddChapter(SiteChapter{
		Title:    "Part 1",
		ID:       "part1",
		Filename: "part1.html",
		Content:  "<h1>Part 1</h1><p>Overview</p>",
		Children: []SiteChapter{
			{
				Title:    "Chapter 1",
				ID:       "ch1",
				Filename: "ch1.html",
				Content:  "<h1>Chapter 1</h1><h2 id=\"sec-1\">Section 1</h2>",
				Depth:    1,
			},
		},
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pagePath := filepath.Join(dir, "ch1.html")
	data, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, "nav-group") {
		t.Error("sidebar should contain collapsible nav group")
	}
	if !strings.Contains(html, "href=\"/ch1.html\"") {
		t.Error("sidebar should contain child chapter link")
	}
}

func TestSiteGeneratorInteractiveSidebar(t *testing.T) {
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Test Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Use Children (sub-chapters) to create collapsible groups.
	// With maxSidebarHeadingDepth = 0, Headings alone don't trigger groups.
	chapters := []SiteChapter{
		{
			Title:    "Part 1",
			ID:       "part1",
			Filename: "part1.html",
			Children: []SiteChapter{
				{Title: "Ch 1.1", ID: "ch1-1", Filename: "ch1-1.html", Depth: 1},
			},
		},
		{
			Title:    "Part 2",
			ID:       "part2",
			Filename: "part2.html",
			Children: []SiteChapter{
				{Title: "Ch 2.1", ID: "ch2-1", Filename: "ch2-1.html", Depth: 1},
			},
		},
	}

	sidebar := gen.buildSidebar(chapters, "ch2-1.html")
	if !strings.Contains(sidebar, `class="nav-group collapsed" data-group-file="part1.html"`) {
		t.Error("inactive chapter groups should be collapsed by default")
	}
	if !strings.Contains(sidebar, `class="nav-group expanded" data-group-file="part2.html"`) {
		t.Error("active chapter groups should be expanded in initial markup")
	}

	dir := t.TempDir()
	gen.Chapters = chapters
	gen.Chapters[0].Content = "<h1>Part 1</h1><p>Overview</p>"
	gen.Chapters[0].Children[0].Content = "<h1>Chapter 1.1</h1><p>Content</p>"
	gen.Chapters[1].Content = "<h1>Part 2</h1><p>Overview</p>"
	gen.Chapters[1].Children[0].Content = "<h1>Chapter 2.1</h1><p>Content</p>"

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	pagePath := filepath.Join(dir, "ch2-1.html")
	data, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}

	html := string(data)
	if strings.Contains(html, "function collapseOtherGroups(currentGroup)") {
		t.Error("generated page should not include accordion-only sidebar helpers")
	}
	if !strings.Contains(html, "function setGroupExpanded(group, shouldExpand)") {
		t.Error("generated page should include multi-expand sidebar helper")
	}
	if !strings.Contains(html, "grid-template-rows 0.28s cubic-bezier") {
		t.Error("generated page should include expand/collapse transition styles")
	}
	if !strings.Contains(html, "window.requestAnimationFrame") {
		t.Error("generated page should schedule scroll spy updates with requestAnimationFrame")
	}
	if !strings.Contains(html, "margin: 0 !important;") {
		t.Error("generated site page should reset body margin for site layout")
	}
	if !strings.Contains(html, "link.rel = 'prefetch'") {
		t.Error("generated site page should prefetch adjacent chapter pages")
	}
	if !strings.Contains(html, "navigateClientSide(target") {
		t.Error("generated site page should include client-side chapter navigation")
	}
	if !strings.Contains(html, "fetchPagePayload(target.url") {
		t.Error("generated site page should fetch target pages for SPA-like navigation")
	}
	if !strings.Contains(html, "scrollToTopImmediate()") {
		t.Error("generated site page should reset scroll immediately before cross-page navigation")
	}
	if !strings.Contains(html, "headingTextForTOC(h)") {
		t.Error("generated site page should strip header anchor text from the page TOC")
	}
	if !strings.Contains(html, "route-progress") {
		t.Error("generated site page should include a route progress indicator")
	}
	if !strings.Contains(html, "saveScrollPosition(window.location.pathname)") {
		t.Error("generated site page should persist scroll position during client-side navigation")
	}
	if !strings.Contains(html, "warmPageCache(link.href)") {
		t.Error("generated site page should warm the page cache for likely next navigations")
	}
}

func TestEpubGeneratorBasic(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.epub")
	coverPath := filepath.Join(dir, "cover.svg")
	coverSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="600" height="800" viewBox="0 0 600 800"><rect width="600" height="800" fill="#0f172a"/><text x="50%" y="50%" dominant-baseline="middle" text-anchor="middle" fill="#f8fafc" font-size="42">Cover</text></svg>`
	if err := os.WriteFile(coverPath, []byte(coverSVG), 0o644); err != nil {
		t.Fatalf("write cover fixture failed: %v", err)
	}

	gen := NewEpubGenerator(EpubMeta{
		Title:          "Test Book",
		Subtitle:       "A Better EPUB",
		Author:         "Author",
		Language:       "en-US",
		Version:        "1.0",
		Description:    "Test description",
		IncludeCover:   true,
		CoverImagePath: coverPath,
	})
	gen.SetCSS("body { font-size: 12pt; }")
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     "<h1>Hello</h1><p>World</p>",
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 2",
		ID:       "ch2",
		Filename: "ch2.xhtml",
		HTML:     "<h1>Chapter 2</h1>",
	})

	err := gen.Generate(outPath)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check file exists and has content
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("epub file not created: %v", err)
	}
	if info.Size() < 100 {
		t.Error("epub file too small")
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	if len(reader.File) == 0 {
		t.Fatal("epub archive should contain files")
	}
	if reader.File[0].Name != "mimetype" {
		t.Fatalf("first epub entry should be mimetype, got %q", reader.File[0].Name)
	}
	if reader.File[0].Method != zip.Store {
		t.Fatal("mimetype entry must be stored without compression")
	}

	opf := readZipEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `version="3.0"`) {
		t.Error("content.opf should declare EPUB 3 package version")
	}
	if !strings.Contains(opf, `properties="nav"`) {
		t.Error("content.opf should include the EPUB 3 nav manifest item")
	}
	if !strings.Contains(opf, `dcterms:modified`) {
		t.Error("content.opf should include dcterms:modified metadata")
	}
	if !strings.Contains(opf, `href="cover.xhtml"`) {
		t.Error("content.opf should include the generated cover page")
	}
	if !strings.Contains(opf, `properties="cover-image"`) {
		t.Error("content.opf should include the packaged cover image manifest item")
	}

	nav := readZipEntry(t, reader.File, "OEBPS/nav.xhtml")
	if !strings.Contains(nav, "Chapter 1") || !strings.Contains(nav, "Chapter 2") {
		t.Error("nav.xhtml should link all chapters")
	}
	if !strings.Contains(nav, "Cover") {
		t.Error("nav.xhtml should include the generated cover page")
	}

	cover := readZipEntry(t, reader.File, "OEBPS/cover.xhtml")
	if !strings.Contains(cover, "Test Book") || !strings.Contains(cover, "A Better EPUB") {
		t.Error("cover.xhtml should render title and subtitle")
	}
	if !strings.Contains(cover, `src="assets/cover.svg"`) {
		t.Error("cover.xhtml should reference the packaged cover image")
	}

	coverAsset := readZipEntry(t, reader.File, "OEBPS/assets/cover.svg")
	if !strings.Contains(coverAsset, "<svg") {
		t.Error("epub archive should include the packaged cover image asset")
	}
}

func TestEpubGeneratorNoCSS(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "T", Author: "A", Language: "en"})
	gen.AddChapter(EpubChapter{
		Title: "Ch", ID: "ch1", Filename: "ch1.xhtml", HTML: "<p>test</p>",
	})

	err := gen.Generate(outPath)
	if err != nil {
		t.Fatalf("Generate without CSS failed: %v", err)
	}
}

func TestNormalizeHTMLForXHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		wants []string
	}{
		{
			name:  "void elements self-close",
			input: `<p>Line<br><img src="cover.png"><input type="checkbox"></p>`,
			wants: []string{`<br />`, `<img src="cover.png" />`, `<input type="checkbox" />`},
		},
		{
			name:  "already self-closed elements unchanged",
			input: `<br /><img src="a.png" />`,
			wants: []string{`<br />`, `<img src="a.png" />`},
		},
		{
			name:  "bare ampersands escaped",
			input: `<p>A&B and C&amp;D and &#123; and &#x1f;</p>`,
			wants: []string{`A&amp;B`, `C&amp;D`, `&#123;`, `&#x1f;`},
		},
		{
			name:  "boolean attributes expanded",
			input: `<input checked type="checkbox"><details open><select disabled multiple>`,
			wants: []string{`checked="checked"`, `open="open"`, `disabled="disabled"`, `multiple="multiple"`},
		},
		{
			name:  "additional void elements",
			input: `<source src="a.mp3"><track src="sub.vtt"><wbr><embed src="x">`,
			wants: []string{`<source src="a.mp3" />`, `<track src="sub.vtt" />`, `<wbr />`, `<embed src="x" />`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHTMLForXHTML(tt.input)
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Errorf("normalized HTML should contain %q, got:\n%s", want, got)
				}
			}
		})
	}
}

func TestEpubGeneratorPackagesChapterImageAssets(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "chapter-images.epub")
	onePixelPNG := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wn2lXQAAAAASUVORK5CYII="

	gen := NewEpubGenerator(EpubMeta{
		Title:    "Image Book",
		Author:   "Author",
		Language: "en-US",
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     `<p><img alt="pixel" src="data:image/png;base64,` + onePixelPNG + `"></p>`,
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	opf := readZipEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `id="asset-img-000"`) {
		t.Error("content.opf should register chapter image assets")
	}
	if !strings.Contains(opf, `media-type="image/png"`) {
		t.Error("content.opf should preserve chapter image media type")
	}

	chapter := readZipEntry(t, reader.File, "OEBPS/ch1.xhtml")
	if !strings.Contains(chapter, `src="assets/img-000.png"`) {
		t.Error("chapter XHTML should rewrite image sources to packaged asset paths")
	}
	if strings.Contains(chapter, `data:image/png;base64`) {
		t.Error("chapter XHTML should not keep data URI images once packaged")
	}

	asset := readZipEntry(t, reader.File, "OEBPS/assets/img-000.png")
	if len(asset) == 0 {
		t.Error("epub archive should contain packaged chapter image bytes")
	}
}

func TestEpubGeneratorPackagesRelativeChapterImageAssets(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "relative-images.epub")
	imagePath := filepath.Join(dir, "diagram.png")
	onePixelPNG, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wn2lXQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture failed: %v", err)
	}
	if err := os.WriteFile(imagePath, onePixelPNG, 0o644); err != nil {
		t.Fatalf("write relative image fixture failed: %v", err)
	}

	gen := NewEpubGenerator(EpubMeta{
		Title:    "Relative Image Book",
		Author:   "Author",
		Language: "en-US",
	})
	gen.AddChapter(EpubChapter{
		Title:     "Chapter 1",
		ID:        "ch1",
		Filename:  "ch1.xhtml",
		HTML:      `<p><img alt="diagram" src="diagram.png"></p>`,
		SourceDir: dir,
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	chapter := readZipEntry(t, reader.File, "OEBPS/ch1.xhtml")
	if !strings.Contains(chapter, `src="assets/diagram-000.png"`) {
		t.Error("chapter XHTML should rewrite relative image paths to packaged asset paths")
	}

	asset := readZipEntry(t, reader.File, "OEBPS/assets/diagram-000.png")
	if len(asset) == 0 {
		t.Error("epub archive should contain packaged relative image bytes")
	}
}

func TestEpubGeneratorPackagesRemoteChapterImageAssets(t *testing.T) {
	// Disable SSRF check since the test uses a local httptest server.
	utils.DisableSSRFCheck()
	t.Cleanup(utils.EnableSSRFCheck)

	onePixelPNG, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wn2lXQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture failed: %v", err)
	}

	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(onePixelPNG)
	}))
	defer imageServer.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "remote-images.epub")

	gen := NewEpubGenerator(EpubMeta{
		Title:    "Remote Image Book",
		Author:   "Author",
		Language: "en-US",
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     `<p><img alt="remote" src="` + imageServer.URL + `/cover.png"></p>`,
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	chapter := readZipEntry(t, reader.File, "OEBPS/ch1.xhtml")
	if !strings.Contains(chapter, `src="assets/cover-000.png"`) {
		t.Error("chapter XHTML should rewrite remote image paths to packaged asset paths")
	}

	opf := readZipEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `href="assets/cover-000.png"`) {
		t.Error("content.opf should include remote image assets in the manifest")
	}

	asset := readZipEntry(t, reader.File, "OEBPS/assets/cover-000.png")
	if len(asset) == 0 {
		t.Error("epub archive should contain packaged remote image bytes")
	}
}

func TestEpubGeneratorDeduplicatesSharedImageAssets(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "dedupe-images.epub")
	imagePath := filepath.Join(dir, "shared.png")
	onePixelPNG, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Wn2lXQAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode png fixture failed: %v", err)
	}
	if err := os.WriteFile(imagePath, onePixelPNG, 0o644); err != nil {
		t.Fatalf("write shared image fixture failed: %v", err)
	}

	gen := NewEpubGenerator(EpubMeta{
		Title:    "Shared Image Book",
		Author:   "Author",
		Language: "en-US",
	})
	gen.AddChapter(EpubChapter{
		Title:     "Chapter 1",
		ID:        "ch1",
		Filename:  "ch1.xhtml",
		HTML:      `<p><img alt="shared" src="shared.png"></p>`,
		SourceDir: dir,
	})
	gen.AddChapter(EpubChapter{
		Title:     "Chapter 2",
		ID:        "ch2",
		Filename:  "ch2.xhtml",
		HTML:      `<p><img alt="shared" src="shared.png"></p>`,
		SourceDir: dir,
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	ch1 := readZipEntry(t, reader.File, "OEBPS/ch1.xhtml")
	ch2 := readZipEntry(t, reader.File, "OEBPS/ch2.xhtml")
	if !strings.Contains(ch1, `src="assets/shared-000.png"`) {
		t.Error("first chapter should reference the shared packaged asset")
	}
	if !strings.Contains(ch2, `src="assets/shared-000.png"`) {
		t.Error("second chapter should reuse the shared packaged asset")
	}

	opf := readZipEntry(t, reader.File, "OEBPS/content.opf")
	if strings.Count(opf, `href="assets/shared-000.png"`) != 1 {
		t.Error("content.opf should register shared image assets only once")
	}

	count := 0
	for _, file := range reader.File {
		if file.Name == "OEBPS/assets/shared-000.png" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("shared asset should be written once, got %d copies", count)
	}
}

func readZipEntry(t *testing.T, files []*zip.File, name string) string {
	t.Helper()

	for _, file := range files {
		if file.Name != name {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s failed: %v", name, err)
		}
		defer rc.Close() //nolint:errcheck

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read zip entry %s failed: %v", name, err)
		}
		return string(data)
	}

	t.Fatalf("zip entry %s not found", name)
	return ""
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Chapter 1", "chapter-1"},
		{"中文标题", "中文标题"},
		{"A&B", "ab"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHTMLGeneratorSlugCollision(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "site")

	// Two chapters that slugify to the same value.
	chapters := map[string]string{
		"Introduction!": "<h1>First</h1>",
		"Introduction?": "<h1>Second</h1>",
	}

	gen := NewHTMLGenerator()
	err := gen.Generate("<html></html>", outDir, chapters)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	htmlFiles := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".html") && e.Name() != "index.html" {
			htmlFiles++
		}
	}

	if htmlFiles != 2 {
		t.Errorf("expected 2 chapter HTML files (deduplicated slugs), got %d", htmlFiles)
	}
}

func TestHTMLGeneratorSlugCollisionThreeWay(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "site")

	// Three chapters: two slugify to "intro" and one naturally slugifies to
	// "intro-2", which is the dedup name the second "intro" would claim.
	chapters := map[string]string{
		"Intro!":  "<h1>First</h1>",
		"Intro?":  "<h1>Second</h1>",
		"Intro-2": "<h1>Third</h1>",
	}

	gen := NewHTMLGenerator()
	err := gen.Generate("<html></html>", outDir, chapters)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	htmlFiles := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".html") && e.Name() != "index.html" {
			htmlFiles++
		}
	}

	if htmlFiles != 3 {
		t.Errorf("expected 3 chapter HTML files (no overwrites), got %d", htmlFiles)
	}
}

func TestHTMLGeneratorEmptySlugFallback(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "site")

	// Chapter name with only special characters slugifies to empty.
	chapters := map[string]string{
		"!!!": "<h1>Special</h1>",
	}

	gen := NewHTMLGenerator()
	err := gen.Generate("<html></html>", outDir, chapters)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should create chapter.html as fallback.
	pagePath := filepath.Join(outDir, "chapter.html")
	if _, err := os.Stat(pagePath); err != nil {
		t.Error("empty slug should fallback to chapter.html")
	}
}

// Site generator tests

func TestSiteGeneratorEmptyChapters(t *testing.T) {
	dir := t.TempDir()
	gen := NewSiteGenerator(SiteMeta{
		Title:    "Empty Book",
		Author:   "Author",
		Language: "en-US",
	})

	// Generate with no chapters should not panic
	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate with empty chapters failed: %v", err)
	}

	// Verify index.html is created
	indexPath := filepath.Join(dir, "index.html")
	_, err := os.Stat(indexPath)
	if err != nil {
		t.Errorf("index.html should be created even with no chapters: %v", err)
	}
}

func TestSiteGeneratorCSSInjection(t *testing.T) {
	dir := t.TempDir()
	customCSS := "body { background-color: #fff; color: #000; }"

	gen := NewSiteGenerator(SiteMeta{
		Title:    "CSS Test Book",
		Author:   "Author",
		Language: "en-US",
	})
	gen.SetCSS(customCSS)
	gen.AddChapter(SiteChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.html",
		Content:  "<h1>Chapter 1</h1>",
	})

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify CSS appears in generated page
	pagePath := filepath.Join(dir, "ch1.html")
	data, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read page failed: %v", err)
	}

	html := string(data)
	if !strings.Contains(html, customCSS) {
		t.Error("generated page should contain custom CSS")
	}
}

func TestSiteGeneratorMultiplePages(t *testing.T) {
	dir := t.TempDir()

	gen := NewSiteGenerator(SiteMeta{
		Title:    "Multi-page Book",
		Author:   "Author",
		Language: "en-US",
	})

	for i := 1; i <= 3; i++ {
		gen.AddChapter(SiteChapter{
			Title:    fmt.Sprintf("Chapter %d", i),
			ID:       fmt.Sprintf("ch%d", i),
			Filename: fmt.Sprintf("ch%d.html", i),
			Content:  fmt.Sprintf("<h1>Chapter %d</h1>", i),
		})
	}

	if err := gen.Generate(dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify 3 chapter files + index are created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory failed: %v", err)
	}

	fileCount := 0
	hasIndex := false
	chapterFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			fileCount++
			name := entry.Name()
			if name == "index.html" {
				hasIndex = true
			}
			if strings.HasPrefix(name, "ch") && strings.HasSuffix(name, ".html") {
				chapterFiles++
			}
		}
	}

	if !hasIndex {
		t.Error("index.html should be generated")
	}
	if chapterFiles < 3 {
		t.Errorf("expected at least 3 chapter files, got %d", chapterFiles)
	}
}

// EPUB generator tests

func TestEpubGeneratorNoCover(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "no-cover.epub")

	gen := NewEpubGenerator(EpubMeta{
		Title:        "No Cover Book",
		Author:       "Author",
		Language:     "en",
		IncludeCover: false, // No cover
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     "<h1>Hello</h1><p>Content</p>",
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	// Verify cover.xhtml does not exist
	for _, file := range reader.File {
		if file.Name == "OEBPS/cover.xhtml" {
			t.Error("cover.xhtml should not be included when IncludeCover=false")
		}
	}

	// Verify nav.xhtml does not reference cover
	nav := readZipEntry(t, reader.File, "OEBPS/nav.xhtml")
	if strings.Contains(nav, "cover.xhtml") {
		t.Error("nav.xhtml should not reference cover when IncludeCover=false")
	}
}

func TestEpubGeneratorEmptyLanguage(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "empty-lang.epub")

	gen := NewEpubGenerator(EpubMeta{
		Title:    "Empty Language Book",
		Author:   "Author",
		Language: "", // Empty language
	})
	gen.AddChapter(EpubChapter{
		Title:    "Chapter 1",
		ID:       "ch1",
		Filename: "ch1.xhtml",
		HTML:     "<h1>Test</h1>",
	})

	if err := gen.Generate(outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	reader, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("open epub zip failed: %v", err)
	}
	defer reader.Close() //nolint:errcheck

	// Verify nav.xhtml defaults to "en"
	nav := readZipEntry(t, reader.File, "OEBPS/nav.xhtml")
	if !strings.Contains(nav, `lang="en"`) && !strings.Contains(nav, `xml:lang="en"`) {
		t.Error("nav.xhtml should default to 'en' language when empty")
	}

	// Also check OPF for language metadata
	opf := readZipEntry(t, reader.File, "OEBPS/content.opf")
	if !strings.Contains(opf, `<dc:language>en</dc:language>`) && !strings.Contains(opf, `<dc:language></dc:language>`) {
		// At minimum, verify language handling is present
		if !strings.Contains(opf, "language") {
			t.Error("content.opf should include language metadata")
		}
	}
}

func TestNormalizeHTMLForXHTMLBooleanAttrEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wants    []string
		notWants []string
	}{
		{
			name:     "valued data attribute not modified",
			input:    `<input data-checked="true">`,
			wants:    []string{`data-checked="true"`},
			notWants: []string{`data-checked="data-checked"`},
		},
		{
			name:     "multiple consecutive boolean attributes",
			input:    `<select disabled multiple></select>`,
			wants:    []string{`disabled="disabled"`, `multiple="multiple"`},
			notWants: []string{},
		},
		{
			name:     "mixed boolean and valued attributes",
			input:    `<input type="text" required data-required="yes">`,
			wants:    []string{`type="text"`, `required="required"`, `data-required="yes"`},
			notWants: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeHTMLForXHTML(tt.input)
			for _, want := range tt.wants {
				if !strings.Contains(got, want) {
					t.Errorf("normalized HTML should contain %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tt.notWants {
				if strings.Contains(got, notWant) {
					t.Errorf("normalized HTML should NOT contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}
