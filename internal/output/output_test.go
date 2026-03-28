// output_test.go tests output format registry and render request handling.
// Tests cover format registration, retrieval, metadata handling, and concurrency.
package output

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers: Mock OutputFormat implementations
// ---------------------------------------------------------------------------

type mockFormat struct {
	name        string
	description string
	generateErr error
}

func (m *mockFormat) Name() string {
	return m.name
}

func (m *mockFormat) Description() string {
	return m.description
}

func (m *mockFormat) Generate(ctx context.Context, req *RenderRequest, outputPath string) error {
	if m.generateErr != nil {
		return m.generateErr
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test cases: NewRegistry
// ---------------------------------------------------------------------------

func TestNewRegistry_Creation(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("expected non-nil registry")
	}
}

func TestNewRegistry_EmptyOnCreation(t *testing.T) {
	reg := NewRegistry()
	formats := reg.List()
	if len(formats) != 0 {
		t.Errorf("expected empty registry, got %d formats", len(formats))
	}
}

// ---------------------------------------------------------------------------
// Test cases: Register
// ---------------------------------------------------------------------------

func TestRegistry_Register_SingleFormat(t *testing.T) {
	reg := NewRegistry()
	format := &mockFormat{name: "pdf", description: "PDF Format"}

	reg.Register(format)

	if !reg.Has("pdf") {
		t.Error("expected 'pdf' format to be registered")
	}
}

func TestRegistry_Register_MultipleFormats(t *testing.T) {
	reg := NewRegistry()
	formats := []string{"pdf", "html", "epub"}

	for _, fmt := range formats {
		reg.Register(&mockFormat{name: fmt, description: fmt + " format"})
	}

	if len(reg.List()) != 3 {
		t.Errorf("expected 3 formats, got %d", len(reg.List()))
	}

	for _, fmt := range formats {
		if !reg.Has(fmt) {
			t.Errorf("expected '%s' to be registered", fmt)
		}
	}
}

func TestRegistry_Register_Replace(t *testing.T) {
	reg := NewRegistry()

	// Register first format
	format1 := &mockFormat{name: "pdf", description: "PDF v1"}
	reg.Register(format1)

	f, err := reg.Get("pdf")
	if err != nil {
		t.Fatalf("expected no error getting initial format, got: %v", err)
	}
	if f.Description() != "PDF v1" {
		t.Error("expected initial description")
	}

	// Register replacement
	format2 := &mockFormat{name: "pdf", description: "PDF v2"}
	reg.Register(format2)

	f, err = reg.Get("pdf")
	if err != nil {
		t.Fatalf("expected no error getting replaced format, got: %v", err)
	}
	if f.Description() != "PDF v2" {
		t.Error("expected format to be replaced with new description")
	}
}

func TestRegistry_Register_NilFormat(t *testing.T) {
	reg := NewRegistry()
	// Registering nil should panic because the implementation calls f.Name().
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		reg.Register(nil)
	}()
	if !panicked {
		t.Error("expected Register(nil) to panic, but it did not")
	}
}

// ---------------------------------------------------------------------------
// Test cases: Get
// ---------------------------------------------------------------------------

func TestRegistry_Get_ExistingFormat(t *testing.T) {
	reg := NewRegistry()
	format := &mockFormat{name: "html", description: "HTML Format"}
	reg.Register(format)

	retrieved, err := reg.Get("html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil format")
	}
	if retrieved.Name() != "html" {
		t.Errorf("expected name 'html', got %q", retrieved.Name())
	}
}

func TestRegistry_Get_NonExistentFormat(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent format")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegistry_Get_EmptyRegistry(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("any")
	if err == nil {
		t.Error("expected error when retrieving from empty registry")
	}
}

func TestRegistry_Get_AvailableFormatsInError(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})
	reg.Register(&mockFormat{name: "html"})

	_, err := reg.Get("epub")
	if err == nil {
		t.Fatal("expected error")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "pdf") && !strings.Contains(errMsg, "html") {
		t.Errorf("error message should list available formats: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test cases: List
// ---------------------------------------------------------------------------

func TestRegistry_List_Empty(t *testing.T) {
	reg := NewRegistry()
	formats := reg.List()
	if len(formats) != 0 {
		t.Errorf("expected empty list, got %d formats", len(formats))
	}
}

func TestRegistry_List_SingleFormat(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	formats := reg.List()
	if len(formats) != 1 {
		t.Fatalf("expected 1 format, got %d", len(formats))
	}
	if formats[0] != "pdf" {
		t.Errorf("expected 'pdf', got %q", formats[0])
	}
}

func TestRegistry_List_MultipleFormats(t *testing.T) {
	reg := NewRegistry()
	expected := []string{"pdf", "html", "epub", "site"}

	for _, fmt := range expected {
		reg.Register(&mockFormat{name: fmt})
	}

	formats := reg.List()
	if len(formats) != len(expected) {
		t.Fatalf("expected %d formats, got %d", len(expected), len(formats))
	}

	// Convert to map for order-independent comparison
	formatMap := make(map[string]bool)
	for _, f := range formats {
		formatMap[f] = true
	}

	for _, e := range expected {
		if !formatMap[e] {
			t.Errorf("expected format %q not found in list", e)
		}
	}
}

func TestRegistry_List_ReturnsCopy(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	list1 := reg.List()
	list2 := reg.List()

	// Modify first list element in place
	if len(list1) == 0 {
		t.Fatal("expected non-empty list")
	}
	list1[0] = "modified"

	// Check that second list is not affected
	if list2[0] == "modified" {
		t.Error("modifying returned list should not affect other callers")
	}
}

// ---------------------------------------------------------------------------
// Test cases: Has
// ---------------------------------------------------------------------------

func TestRegistry_Has_ExistingFormat(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	if !reg.Has("pdf") {
		t.Error("expected Has to return true for registered format")
	}
}

func TestRegistry_Has_NonExistentFormat(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	if reg.Has("html") {
		t.Error("expected Has to return false for non-registered format")
	}
}

func TestRegistry_Has_EmptyRegistry(t *testing.T) {
	reg := NewRegistry()

	if reg.Has("any") {
		t.Error("expected Has to return false for empty registry")
	}
}

func TestRegistry_Has_CaseSensitive(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	if reg.Has("PDF") {
		t.Error("expected Has to be case-sensitive")
	}
}

// ---------------------------------------------------------------------------
// Test cases: RenderRequest
// ---------------------------------------------------------------------------

func TestRenderRequest_Creation(t *testing.T) {
	req := &RenderRequest{
		FullHTML: "<html></html>",
		CSS:      "body { color: black; }",
		Meta: DocumentMeta{
			Title:    "Test Book",
			Author:   "Test Author",
			Language: "en",
			Version:  "1.0",
		},
	}

	if req.FullHTML != "<html></html>" {
		t.Error("FullHTML not set correctly")
	}
	if req.CSS != "body { color: black; }" {
		t.Error("CSS not set correctly")
	}
	if req.Meta.Title != "Test Book" {
		t.Error("Meta title not set correctly")
	}
}

func TestRenderRequest_WithChapters(t *testing.T) {
	chapters := []ChapterContent{
		{
			Title:    "Chapter 1",
			ID:       "ch1",
			HTML:     "<h1>Chapter 1</h1>",
			Filename: "ch_001.xhtml",
		},
		{
			Title:    "Chapter 2",
			ID:       "ch2",
			HTML:     "<h1>Chapter 2</h1>",
			Filename: "ch_002.xhtml",
		},
	}

	req := &RenderRequest{
		Chapters: chapters,
	}

	if len(req.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(req.Chapters))
	}

	if req.Chapters[0].Title != "Chapter 1" {
		t.Error("first chapter title mismatch")
	}
	if req.Chapters[1].Filename != "ch_002.xhtml" {
		t.Error("second chapter filename mismatch")
	}
}

func TestRenderRequest_EmptyChapters(t *testing.T) {
	req := &RenderRequest{
		Chapters: []ChapterContent{},
	}

	if len(req.Chapters) != 0 {
		t.Error("expected empty chapters slice")
	}
}

// ---------------------------------------------------------------------------
// Test cases: ChapterContent
// ---------------------------------------------------------------------------

func TestChapterContent_Creation(t *testing.T) {
	chapter := ChapterContent{
		Title:    "Introduction",
		ID:       "intro",
		HTML:     "<p>Welcome</p>",
		Filename: "ch_000.xhtml",
	}

	if chapter.Title != "Introduction" {
		t.Error("title mismatch")
	}
	if chapter.ID != "intro" {
		t.Error("id mismatch")
	}
	if chapter.HTML != "<p>Welcome</p>" {
		t.Error("html mismatch")
	}
	if chapter.Filename != "ch_000.xhtml" {
		t.Error("filename mismatch")
	}
}

// ---------------------------------------------------------------------------
// Test cases: Concurrency
// ---------------------------------------------------------------------------

func TestRegistry_Concurrent_Register(t *testing.T) {
	reg := NewRegistry()
	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("format-%d", idx)
			reg.Register(&mockFormat{name: name})
		}(i)
	}
	wg.Wait()

	formats := reg.List()
	if len(formats) != numGoroutines {
		t.Errorf("expected %d formats after concurrent register, got %d", numGoroutines, len(formats))
	}
}

func TestRegistry_Concurrent_GetAndList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})
	reg.Register(&mockFormat{name: "html"})

	var wg sync.WaitGroup
	errors := make(chan error, 20)
	numGoroutines := 20

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				_, err := reg.Get("pdf")
				if err != nil {
					errors <- err
				}
			} else {
				list := reg.List()
				if len(list) == 0 {
					errors <- fmt.Errorf("unexpected empty list")
				}
			}
		}(i)
	}
	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestRegistry_Concurrent_Has(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockFormat{name: "pdf"})

	var wg sync.WaitGroup
	numGoroutines := 10

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if !reg.Has("pdf") {
				t.Error("concurrent Has should find registered format")
			}
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestRegistry_CompleteWorkflow(t *testing.T) {
	reg := NewRegistry()

	// Register multiple formats
	formats := []string{"pdf", "html", "epub", "site"}
	for _, fmt := range formats {
		reg.Register(&mockFormat{name: fmt, description: fmt + " format"})
	}

	// Verify list
	list := reg.List()
	if len(list) != len(formats) {
		t.Fatalf("expected %d formats, got %d", len(formats), len(list))
	}

	// Verify each can be retrieved
	for _, fmt := range formats {
		f, err := reg.Get(fmt)
		if err != nil {
			t.Fatalf("failed to get %q: %v", fmt, err)
		}
		if f.Name() != fmt {
			t.Errorf("format name mismatch for %q", fmt)
		}
	}

	// Verify Has works
	for _, fmt := range formats {
		if !reg.Has(fmt) {
			t.Errorf("Has should return true for %q", fmt)
		}
	}

	// Verify error for non-existent
	_, err := reg.Get("unknown")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

func TestRenderRequest_CompleteMetadata(t *testing.T) {
	meta := DocumentMeta{
		Title:    "Complete Guide to Go",
		Author:   "Rob Pike",
		Language: "en",
		Version:  "2.0.0",
	}

	chapters := []ChapterContent{
		{
			Title:    "Getting Started",
			ID:       "ch1",
			HTML:     "<div>content</div>",
			Filename: "ch_001.xhtml",
		},
	}

	req := &RenderRequest{
		FullHTML: "<!DOCTYPE html>...</html>",
		CSS:      "body { font-family: serif; }",
		Chapters: chapters,
		Meta:     meta,
	}

	// Verify all fields
	if req.Meta.Title != meta.Title {
		t.Error("meta title not preserved")
	}
	if len(req.Chapters) != 1 {
		t.Error("chapters not preserved")
	}
	if req.CSS == "" {
		t.Error("css not preserved")
	}
}
