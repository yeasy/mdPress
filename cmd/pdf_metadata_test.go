package cmd

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/internal/config"
)

func TestPDFDocumentMetadataFromBookConfig(t *testing.T) {
	cfg := &config.BookConfig{}
	cfg.Book.Title = "Metadata Probe Book"
	cfg.Book.Subtitle = "A Subtitle"
	cfg.Book.Author = "Jane Author"
	cfg.Book.Description = "A book used to probe PDF metadata."

	meta := pdfDocumentMetadata(cfg)

	if want := "Metadata Probe Book: A Subtitle"; meta.Title != want {
		t.Errorf("Title = %q, want %q", meta.Title, want)
	}
	if meta.Author != "Jane Author" {
		t.Errorf("Author = %q, want %q", meta.Author, "Jane Author")
	}
	if meta.Subject != cfg.Book.Description {
		t.Errorf("Subject = %q, want %q", meta.Subject, cfg.Book.Description)
	}
	// Chrome otherwise labels every mdPress PDF as written by HeadlessChrome.
	if !strings.HasPrefix(meta.Creator, "mdPress ") {
		t.Errorf("Creator = %q, want it to name mdPress", meta.Creator)
	}
}

func TestPDFDocumentMetadataWithoutSubtitle(t *testing.T) {
	cfg := &config.BookConfig{}
	cfg.Book.Title = "Solo Title"

	if got := pdfDocumentMetadata(cfg).Title; got != "Solo Title" {
		t.Errorf("Title = %q, want %q", got, "Solo Title")
	}
}
