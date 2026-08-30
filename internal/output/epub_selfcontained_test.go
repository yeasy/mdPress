package output

import (
	"archive/zip"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"testing"
)

// Embedded resources — the things a reading system must fetch to render the
// page. A hyperlink in prose is deliberately not in this set: EPUB 3 lets a
// book link out to the web, it just may not depend on the web to render.
var (
	scriptSrcPattern = regexp.MustCompile(`<script[^>]*\ssrc="([^"]+)"`)
	linkHrefPattern  = regexp.MustCompile(`<link[^>]*\shref="([^"]+)"`)
	imgSrcPattern    = regexp.MustCompile(`<img[^>]*\ssrc="([^"]+)"`)
	cssURLPattern    = regexp.MustCompile(`url\(([^)]+)\)`)
	opfItemPattern   = regexp.MustCompile(`<item[^>]*\shref="([^"]+)"`)
)

// epubEntries reads every file out of an .epub into a map keyed by archive path.
func epubEntries(t *testing.T, epubPath string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		t.Fatalf("open epub: %v", err)
	}
	defer r.Close() //nolint:errcheck
	out := make(map[string][]byte, len(r.File))
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close() //nolint:errcheck
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		out[f.Name] = data
	}
	return out
}

// embeddedRefs returns every resource the given packaged file needs, resolved
// to archive paths.
func embeddedRefs(name, content string) []string {
	var patterns []*regexp.Regexp
	switch path.Ext(name) {
	case ".xhtml", ".html":
		patterns = []*regexp.Regexp{scriptSrcPattern, linkHrefPattern, imgSrcPattern}
	case ".css":
		patterns = []*regexp.Regexp{cssURLPattern}
	default:
		return nil
	}
	dir := path.Dir(name)
	var refs []string
	for _, p := range patterns {
		for _, m := range p.FindAllStringSubmatch(content, -1) {
			ref := strings.Trim(strings.TrimSpace(m[1]), `'"`)
			if ref == "" || strings.HasPrefix(ref, "#") || strings.HasPrefix(ref, "data:") {
				continue
			}
			refs = append(refs, resolveEpubRef(dir, ref))
		}
	}
	return refs
}

// resolveEpubRef turns a reference as written into the archive path it names,
// leaving absolute URLs alone so the caller can report them.
func resolveEpubRef(dir, ref string) string {
	if strings.Contains(ref, "://") {
		return ref
	}
	if decoded, err := url.PathUnescape(ref); err == nil {
		ref = decoded
	}
	if idx := strings.IndexAny(ref, "#?"); idx >= 0 {
		ref = ref[:idx]
	}
	return path.Join(dir, ref)
}

// checkEpubIsSelfContained runs the checks epubcheck would run for the defect
// class this file exists to prevent: nothing embedded may live off the
// network (OPF-014 / RSC-006), every reference must resolve inside the
// container (RSC-007), and every packaged file must be declared in the
// manifest (OPF-003).
func checkEpubIsSelfContained(t *testing.T, epubPath string) map[string][]byte {
	t.Helper()
	entries := epubEntries(t, epubPath)

	opf, ok := entries["OEBPS/content.opf"]
	if !ok {
		t.Fatal("epub has no OEBPS/content.opf")
	}
	manifested := map[string]bool{}
	for _, m := range opfItemPattern.FindAllStringSubmatch(string(opf), -1) {
		manifested[resolveEpubRef("OEBPS", m[1])] = true
	}

	for name, data := range entries {
		for _, ref := range embeddedRefs(name, string(data)) {
			if strings.Contains(ref, "://") {
				t.Errorf("%s embeds a remote resource: %s", name, ref)
				continue
			}
			if _, ok := entries[ref]; !ok {
				t.Errorf("%s references %s, which is not in the archive", name, ref)
			}
		}
		// mimetype and the container descriptor are the two files EPUB 3
		// defines outside the manifest; the manifest cannot list itself.
		switch {
		case name == "mimetype", strings.HasPrefix(name, "META-INF/"), name == "OEBPS/content.opf":
			continue
		}
		if !manifested[name] {
			t.Errorf("%s is packaged but not declared in the OPF manifest", name)
		}
	}
	return entries
}

// TestEpubWithMathIsSelfContained is the regression test for rank 7: math used
// to be rendered by scripts, a stylesheet and fonts fetched from jsDelivr, so
// every formula in an EPUB depended on the reader being online — which EPUB
// 3.3 does not allow and an e-reader routinely is not. KaTeX now ships inside
// the book.
func TestEpubWithMathIsSelfContained(t *testing.T) {
	dir := t.TempDir()
	out := path.Join(dir, "math.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Math Book", Language: "en-US"})
	gen.AddChapter(EpubChapter{
		Title:    "Math",
		Filename: "math.xhtml",
		HTML:     `<p>Inline <span class="math math-inline">$E = mc^2$</span> and a block.</p>`,
	})
	if err := gen.Generate(out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	entries := checkEpubIsSelfContained(t, out)

	// The runtime a reader actually needs must all be present.
	for _, want := range []string{
		"OEBPS/katex/katex.min.css",
		"OEBPS/katex/katex.min.js",
		"OEBPS/katex/auto-render.min.js",
		"OEBPS/katex/LICENSE", // KaTeX is MIT; the license travels with the code
	} {
		if _, ok := entries[want]; !ok {
			t.Errorf("%s is missing from the epub", want)
		}
	}
	var fonts int
	for name := range entries {
		if strings.HasPrefix(name, "OEBPS/katex/fonts/") {
			fonts++
		}
	}
	if fonts < 20 {
		t.Errorf("packaged %d KaTeX fonts, want the full set", fonts)
	}

	chapter := string(entries["OEBPS/math.xhtml"])
	if !strings.Contains(chapter, `href="katex/katex.min.css"`) {
		t.Errorf("math chapter does not link the packaged stylesheet:\n%s", chapter)
	}
	if strings.Contains(chapter, "cdn.jsdelivr.net") {
		t.Errorf("math chapter still points at the CDN:\n%s", chapter)
	}
}

// TestEpubWithoutMathPackagesNoKaTeX: the runtime is ~600 KB, and a book with
// no formulas has no reason to carry it.
func TestEpubWithoutMathPackagesNoKaTeX(t *testing.T) {
	dir := t.TempDir()
	out := path.Join(dir, "plain.epub")

	gen := NewEpubGenerator(EpubMeta{Title: "Plain Book", Language: "en-US"})
	gen.AddChapter(EpubChapter{Title: "Plain", Filename: "plain.xhtml", HTML: `<p>No math here.</p>`})
	if err := gen.Generate(out); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	entries := checkEpubIsSelfContained(t, out)
	for name := range entries {
		if strings.Contains(name, "katex") {
			t.Errorf("a book without math should not package %s", name)
		}
	}
}

// TestEpubMathChapterWithRemoteImageKeepsBothProperties: the two manifest
// properties are independent. Math implies scripted; only a resource that
// genuinely lives off the network implies remote-resources — and a math
// chapter can still have one when an image download failed and the build
// degraded to the original URL.
func TestEpubMathChapterWithRemoteImageKeepsBothProperties(t *testing.T) {
	gen := NewEpubGenerator(EpubMeta{Title: "Mixed", Language: "en-US"})
	chapters := []EpubChapter{
		{Title: "Both", ID: "both", Filename: "both.xhtml",
			HTML: `<p><span class="math math-inline">$x$</span></p><p><img src="https://example.com/a.png" alt="a"/></p>`},
		{Title: "MathOnly", ID: "mathonly", Filename: "mathonly.xhtml",
			HTML: `<p><span class="math math-inline">$y$</span></p>`},
	}
	opf := gen.generateOPF(chapters, nil, nil, nil)

	for _, want := range []string{
		`href="both.xhtml" media-type="application/xhtml+xml" properties="scripted remote-resources"`,
		`href="mathonly.xhtml" media-type="application/xhtml+xml" properties="scripted"`,
	} {
		if !strings.Contains(opf, want) {
			t.Errorf("manifest is missing %s:\n%s", want, opf)
		}
	}
}
