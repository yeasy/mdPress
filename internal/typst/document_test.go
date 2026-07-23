package typst

import (
	"strings"
	"testing"
	"time"
)

// The Typst backend used to emit no document information at all, so every PDF
// it produced showed up untitled and author-less in library software.
func TestRenderTypstDocumentSetsDocumentMetadata(t *testing.T) {
	out, err := renderTypstDocument(TypstTemplateData{
		Title:       "Metadata Probe Book",
		Author:      "Jane Author",
		Description: "A book used to probe PDF metadata.",
		BuildTime:   time.Date(2023, time.November, 14, 22, 13, 20, 0, time.UTC),
		Content:     "Body",
	})
	if err != nil {
		t.Fatalf("renderTypstDocument: %v", err)
	}

	want := `#set document(title: "Metadata Probe Book", author: ("Jane Author",), ` +
		`description: "A book used to probe PDF metadata.", ` +
		`date: datetime(year: 2023, month: 11, day: 14, hour: 22, minute: 13, second: 20))`
	if !strings.Contains(out, want) {
		t.Errorf("rendered document is missing %q:\n%s", want, out[:min(len(out), 400)])
	}
	// Typst requires set rules before any content is produced.
	if !strings.HasPrefix(out, "#set document(") {
		t.Errorf("#set document must come first, got:\n%s", out[:min(len(out), 120)])
	}
}

func TestRenderTypstDocumentOmitsEmptyMetadata(t *testing.T) {
	out, err := renderTypstDocument(TypstTemplateData{Content: "Body"})
	if err != nil {
		t.Fatalf("renderTypstDocument: %v", err)
	}
	if strings.Contains(out, "#set document(") {
		t.Errorf("no metadata should mean no #set document call:\n%s", out[:min(len(out), 200)])
	}
}

// Metadata goes into a Typst string literal, where the markup escaping applied
// to body text would surface as stray backslashes in the PDF.
func TestRenderTypstDocumentEscapesStringLiterals(t *testing.T) {
	out, err := renderTypstDocument(TypstTemplateData{
		Title:   `Say "hi" #now`,
		Author:  `A \ B`,
		Content: "Body",
	})
	if err != nil {
		t.Fatalf("renderTypstDocument: %v", err)
	}
	if !strings.Contains(out, `title: "Say \"hi\" #now"`) {
		t.Errorf("title literal not escaped as expected:\n%s", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, `author: ("A \\ B",)`) {
		t.Errorf("author literal not escaped as expected:\n%s", out[:min(len(out), 200)])
	}
}

func TestBuildTimeHonorsSourceDateEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1700000000")
	if got, want := buildTime(), time.Unix(1700000000, 0).UTC(); !got.Equal(want) {
		t.Errorf("buildTime() = %v, want %v", got, want)
	}
	if got, want := currentDate(), "2023-11-14"; got != want {
		t.Errorf("currentDate() = %q, want %q", got, want)
	}
}
