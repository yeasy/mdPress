package glossary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GLOSSARY.md")
	content := `# Glossary

## API
Application Programming Interface.

## Markdown
A lightweight markup language.

## GFM
GitHub Flavored Markdown, an extension of Markdown.
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write GLOSSARY.md failed: %v", err)
	}

	g, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(g.Terms) != 3 {
		t.Fatalf("expected 3 terms, got %d", len(g.Terms))
	}
	if g.Terms[0].Name != "API" {
		t.Errorf("first term: got %q", g.Terms[0].Name)
	}
	if g.Terms[0].Definition != "Application Programming Interface." {
		t.Errorf("first definition: got %q", g.Terms[0].Definition)
	}
}

func TestParseFileMultiLineDef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GLOSSARY.md")
	content := `## Docker
A platform for building,
shipping, and running
applications in containers.
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write GLOSSARY.md failed: %v", err)
	}

	g, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(g.Terms) != 1 {
		t.Fatalf("expected 1 term, got %d", len(g.Terms))
	}
	if !strings.Contains(g.Terms[0].Definition, "platform") {
		t.Error("multi-line definition should be joined")
	}
}

func TestParseFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GLOSSARY.md")
	if err := os.WriteFile(path, []byte("# Glossary\n"), 0644); err != nil {
		t.Fatalf("write GLOSSARY.md failed: %v", err)
	}

	g, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}
	if len(g.Terms) != 0 {
		t.Errorf("expected 0 terms, got %d", len(g.Terms))
	}
}

func TestParseFileNonExistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/GLOSSARY.md")
	if err == nil {
		t.Error("should fail for non-existent file")
	}
}

func TestProcessHTML(t *testing.T) {
	g := &Glossary{
		Terms: []Term{
			{Name: "API", Definition: "Application Programming Interface"},
			{Name: "Markdown", Definition: "A markup language"},
		},
	}

	html := "<p>This API uses Markdown format.</p>"
	result := g.ProcessHTML(html)

	if !strings.Contains(result, `class="glossary-term"`) {
		t.Error("should add glossary-term class")
	}
	if !strings.Contains(result, `title="Application Programming Interface"`) {
		t.Error("should add tooltip with definition")
	}
}

func TestProcessHTMLNoTerms(t *testing.T) {
	g := &Glossary{}
	html := "<p>Hello world</p>"
	result := g.ProcessHTML(html)
	if result != html {
		t.Error("empty glossary should not modify HTML")
	}
}

func TestProcessHTMLSkipsTags(t *testing.T) {
	g := &Glossary{
		Terms: []Term{{Name: "href", Definition: "test"}},
	}
	html := `<a href="test">Click href here</a>`
	result := g.ProcessHTML(html)
	// href inside <a> tag attribute should not be highlighted
	// but "href" in text content may be
	if strings.Contains(result, `<a <span`) {
		t.Error("should not modify HTML tag attributes")
	}
}

func TestRenderHTML(t *testing.T) {
	g := &Glossary{
		Terms: []Term{
			{Name: "Zebra", Definition: "An animal"},
			{Name: "Apple", Definition: "A fruit"},
		},
	}

	html := g.RenderHTML()
	if !strings.Contains(html, "glossary-page") {
		t.Error("should have glossary-page class")
	}
	if !strings.Contains(html, "<dl") {
		t.Error("should use definition list")
	}
	// Should be alphabetically sorted
	appleIdx := strings.Index(html, "Apple")
	zebraIdx := strings.Index(html, "Zebra")
	if appleIdx > zebraIdx {
		t.Error("terms should be sorted alphabetically")
	}
}

func TestRenderHTMLEmpty(t *testing.T) {
	g := &Glossary{}
	if g.RenderHTML() != "" {
		t.Error("empty glossary should return empty string")
	}
}

func TestRenderHTMLEscaping(t *testing.T) {
	g := &Glossary{
		Terms: []Term{{Name: "<script>", Definition: `"alert('xss')"`}},
	}
	html := g.RenderHTML()
	if strings.Contains(html, "<script>") {
		t.Error("should escape HTML in term names")
	}
}
