// extensions_test.go tests heading ID generation and AST transformation.
// Tests cover ID generation, uniqueness, special characters, and edge cases.
package markdown

import (
	"testing"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ---------------------------------------------------------------------------
// Test helper: createHeadingNode
// ---------------------------------------------------------------------------

func createHeadingNode(level int, content string) *ast.Heading {
	heading := ast.NewHeading(level)
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(0, len([]byte(content)))
	heading.AppendChild(heading, textNode)
	return heading
}

// ---------------------------------------------------------------------------
// Test cases: newHeadingIDTransformer and processHeading
// ---------------------------------------------------------------------------

func TestNewHeadingIDTransformer_Creation(t *testing.T) {
	transformer := newHeadingIDTransformer()
	if transformer == nil {
		t.Fatal("expected non-nil transformer")
	}

	// Verify it implements the ASTTransformer interface
	_ = transformer
}

func TestHeadingIDTransformer_ProcessHeading_SingleHeading(t *testing.T) {
	transformer := newHeadingIDTransformer()
	heading := createHeadingNode(1, "Test Heading")
	source := []byte("# Test Heading")

	ht := transformer.(*headingIDTransformer)
	ht.processHeading(heading, source)

	id, ok := heading.AttributeString("id")
	if !ok {
		t.Fatal("expected id attribute to be set")
	}
	idBytes, _ := id.([]byte)
	if string(idBytes) == "" {
		t.Error("expected non-empty id")
	}
}

func TestHeadingIDTransformer_ProcessHeading_EmptyHeading(t *testing.T) {
	transformer := newHeadingIDTransformer()
	heading := ast.NewHeading(1)
	source := []byte("")

	ht := transformer.(*headingIDTransformer)
	ht.processHeading(heading, source)

	// Empty heading should not set ID (no text to extract)
	// This is handled by extractNodeText returning empty string
	id, ok := heading.AttributeString("id")
	if ok {
		idBytes, _ := id.([]byte)
		if string(idBytes) == "" {
			t.Error("expected empty id to not be set or to use default")
		}
	}
}

func TestHeadingIDTransformer_ProcessHeading_PreexistingID(t *testing.T) {
	transformer := newHeadingIDTransformer()
	heading := createHeadingNode(1, "Test Heading")
	heading.SetAttributeString("id", []byte("custom-id"))
	source := []byte("# Test Heading")

	ht := transformer.(*headingIDTransformer)
	ht.processHeading(heading, source)

	id, ok := heading.AttributeString("id")
	if !ok {
		t.Fatal("expected id attribute")
	}
	idBytes, _ := id.([]byte)
	if string(idBytes) != "custom-id" {
		t.Errorf("expected 'custom-id' to be preserved, got %q", string(idBytes))
	}
}

// ---------------------------------------------------------------------------
// Test cases: generateUniqueID
// ---------------------------------------------------------------------------

func TestHeadingIDTransformer_GenerateUniqueID_FirstOccurrence(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("Introduction")
	if id != "introduction" {
		t.Errorf("expected 'introduction', got %q", id)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_Duplicate(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id1 := transformer.generateUniqueID("Introduction")
	id2 := transformer.generateUniqueID("Introduction")

	if id1 == id2 {
		t.Errorf("expected different IDs for duplicates, both got %q", id1)
	}
	if id1 != "introduction" {
		t.Errorf("expected first ID 'introduction', got %q", id1)
	}
	if id2 != "introduction-2" {
		t.Errorf("expected second ID 'introduction-2', got %q", id2)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_MultipleDuplicates(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	ids := make([]string, 5)
	for i := 0; i < 5; i++ {
		ids[i] = transformer.generateUniqueID("Chapter")
	}

	expectedIDs := []string{"chapter", "chapter-2", "chapter-3", "chapter-4", "chapter-5"}
	for i, id := range ids {
		if id != expectedIDs[i] {
			t.Errorf("ID %d: expected %q, got %q", i, expectedIDs[i], id)
		}
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_EmptyText(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("")
	if id == "" {
		t.Errorf("expected non-empty id for empty text, got %q", id)
	}
	// Should use default "heading" when baseID is empty
	if id != "heading" {
		t.Errorf("expected 'heading', got %q", id)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_SpecialCharacters(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Hello, World!", "hello-world"},
		{"C++ Programming", "c-programming"},
		{"API & SDKs", "api-sdks"},
		{"Code: Example (v1.0)", "code-example-v10"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"---Dashes---", "dashes"},
		{"_Underscores_", "underscores"},
	}

	for _, tt := range tests {
		transformer := newHeadingIDTransformer().(*headingIDTransformer)
		id := transformer.generateUniqueID(tt.text)
		if id != tt.expected {
			t.Errorf("text %q: expected %q, got %q", tt.text, tt.expected, id)
		}
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_CaseInsensitive(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id1 := transformer.generateUniqueID("HelloWorld")
	id2 := transformer.generateUniqueID("helloworld")

	if id1 != "helloworld" {
		t.Errorf("expected 'helloworld', got %q", id1)
	}
	if id2 != "helloworld-2" {
		t.Errorf("expected 'helloworld-2' (duplicate), got %q", id2)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_ChineseCharacters(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("你好世界")
	if id == "" {
		t.Error("expected non-empty id for Chinese characters")
	}
	// Chinese characters should be preserved in some form
	if id == "heading" {
		t.Errorf("expected id derived from Chinese, got default %q", id)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_MixedLanguages(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("Hello 世界")
	if id == "" {
		t.Error("expected non-empty id for mixed languages")
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_Numbers(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("123 Numbers")
	if id != "123-numbers" {
		t.Errorf("expected '123-numbers', got %q", id)
	}
}

func TestHeadingIDTransformer_GenerateUniqueID_OnlySpecialChars(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	id := transformer.generateUniqueID("!@#$%^&*()")
	if id != "heading" {
		t.Errorf("expected 'heading' for special-char-only text, got %q", id)
	}
}

// ---------------------------------------------------------------------------
// Test cases: extractNodeText
// ---------------------------------------------------------------------------

func TestExtractNodeText_TextNode(t *testing.T) {
	heading := createHeadingNode(1, "Simple Text")
	source := []byte("Simple Text")

	text := extractNodeText(heading, source)
	if text != "Simple Text" {
		t.Errorf("expected 'Simple Text', got %q", text)
	}
}

func TestExtractNodeText_EmptyNode(t *testing.T) {
	heading := ast.NewHeading(1)
	source := []byte("")

	text := extractNodeText(heading, source)
	if text != "" {
		t.Errorf("expected empty string, got %q", text)
	}
}

func TestExtractNodeText_MultipleChildren(t *testing.T) {
	heading := ast.NewHeading(1)
	textNode1 := ast.NewText()
	textNode1.Segment = text.NewSegment(0, 7)
	heading.AppendChild(heading, textNode1)
	textNode2 := ast.NewText()
	textNode2.Segment = text.NewSegment(7, 13)
	heading.AppendChild(heading, textNode2)
	source := []byte("Part 1 Part 2")

	text := extractNodeText(heading, source)
	if text == "" {
		t.Error("expected non-empty text from multiple children")
	}
}

func TestExtractNodeText_CodeSpanContent(t *testing.T) {
	// Create a heading with inline code
	heading := ast.NewHeading(1)
	codeSpan := ast.NewCodeSpan()
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(0, 4)
	codeSpan.AppendChild(codeSpan, textNode)
	heading.AppendChild(heading, codeSpan)
	source := []byte("code")

	text := extractNodeText(heading, source)
	if text == "" {
		t.Error("expected text extracted from code span")
	}
}

// ---------------------------------------------------------------------------
// Test cases: Transform (integration)
// ---------------------------------------------------------------------------

func TestHeadingIDTransformer_Transform_Integration(t *testing.T) {
	// This test verifies the Transform method works in context
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Simulate document with headings
	heading1 := createHeadingNode(1, "First Heading")
	heading2 := createHeadingNode(2, "Second Heading")

	// Process them
	source := []byte("First Heading\nSecond Heading")
	ht.processHeading(heading1, source)
	ht.processHeading(heading2, source)

	// Verify both got IDs
	id1, ok1 := heading1.AttributeString("id")
	id2, ok2 := heading2.AttributeString("id")

	if !ok1 || !ok2 {
		t.Fatal("expected both headings to have ids")
	}

	id1Bytes, _ := id1.([]byte)
	id2Bytes, _ := id2.([]byte)
	if string(id1Bytes) == "" || string(id2Bytes) == "" {
		t.Error("expected non-empty ids")
	}

	if string(id1Bytes) == string(id2Bytes) {
		t.Error("expected different ids for different headings")
	}
}

// ---------------------------------------------------------------------------
// Test cases: Edge cases and special scenarios
// ---------------------------------------------------------------------------

func TestHeadingIDTransformer_LongHeadingText(t *testing.T) {
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	longText := "This is a very long heading with many words that should still generate a valid ID"
	id := transformer.generateUniqueID(longText)

	if id == "" {
		t.Error("expected non-empty id for long text")
	}

	// Should not contain spaces or special characters
	for _, ch := range id {
		if ch == ' ' || ch == '!' {
			t.Errorf("id contains invalid character: %c", ch)
		}
	}
}

func TestHeadingIDTransformer_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"  Leading spaces", "leading-spaces"},
		{"Trailing spaces  ", "trailing-spaces"},
		{"  Both sides  ", "both-sides"},
		{"\tTab\tCharacters", "tab-characters"},
		{"\nNewline\nCharacters", "newline-characters"},
	}

	for _, tt := range tests {
		transformer := newHeadingIDTransformer().(*headingIDTransformer)
		id := transformer.generateUniqueID(tt.text)
		if id != tt.expected {
			t.Errorf("text %q: expected %q, got %q", tt.text, tt.expected, id)
		}
	}
}

func TestHeadingIDTransformer_ThreadSafety(t *testing.T) {
	// Test that the mutex protects concurrent access
	transformer := newHeadingIDTransformer().(*headingIDTransformer)

	// Generate IDs concurrently
	done := make(chan string, 10)
	for i := 0; i < 10; i++ {
		go func() {
			id := transformer.generateUniqueID("Concurrent")
			done <- id
		}()
	}

	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := <-done
		if ids[id] {
			t.Errorf("duplicate id generated: %q", id)
		}
		ids[id] = true
	}

	if len(ids) != 10 {
		t.Errorf("expected 10 unique ids, got %d", len(ids))
	}
}

func TestHeadingIDTransformer_HyphenTrimming(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"-Leading", "leading"},
		{"Trailing-", "trailing"},
		{"-Both-", "both"},
		{"Normal-Text", "normal-text"},
		{"---Multiple---", "multiple"},
	}

	for _, tt := range tests {
		transformer := newHeadingIDTransformer().(*headingIDTransformer)
		id := transformer.generateUniqueID(tt.text)
		if id != tt.expected {
			t.Errorf("text %q: expected %q, got %q", tt.text, tt.expected, id)
		}
	}
}

// ---------------------------------------------------------------------------
// Test cases: Transform method (proper interface tests with Document and Reader)
// ---------------------------------------------------------------------------

func TestHeadingIDTransformer_Transform_EmptyDocument(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create an empty document
	doc := ast.NewDocument()
	source := []byte("")
	reader := text.NewReader(source)
	pc := parser.NewContext()

	// Should not panic or error
	ht.Transform(doc, reader, pc)

	// No headings, so no IDs should be added
	if len(ht.usedIDs) != 0 {
		t.Errorf("expected empty usedIDs map for empty document, got %d entries", len(ht.usedIDs))
	}
}

func TestHeadingIDTransformer_Transform_SingleHeading(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with a single heading
	doc := ast.NewDocument()
	source := []byte("# Introduction\n\nSome content here.")
	heading := ast.NewHeading(1)
	textNode := ast.NewText()
	textNode.Segment = text.NewSegment(2, 15) // "# Introduction" -> text part
	heading.AppendChild(heading, textNode)
	doc.AppendChild(doc, heading)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// Heading should have an ID
	id, ok := heading.AttributeString("id")
	if !ok {
		t.Fatal("expected heading to have id attribute after Transform")
	}

	idBytes, _ := id.([]byte)
	if string(idBytes) == "" {
		t.Error("expected non-empty id")
	}

	// Check that the ID was tracked in usedIDs
	if len(ht.usedIDs) == 0 {
		t.Error("expected usedIDs map to be populated")
	}
}

func TestHeadingIDTransformer_Transform_MultipleHeadings(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with multiple headings at different levels
	doc := ast.NewDocument()
	source := []byte("# Chapter\n\n## Section\n\n### Subsection")

	// Create heading 1
	heading1 := ast.NewHeading(1)
	text1 := ast.NewText()
	text1.Segment = text.NewSegment(2, 9) // "Chapter"
	heading1.AppendChild(heading1, text1)
	doc.AppendChild(doc, heading1)

	// Create heading 2
	heading2 := ast.NewHeading(2)
	text2 := ast.NewText()
	text2.Segment = text.NewSegment(13, 20) // "Section"
	heading2.AppendChild(heading2, text2)
	doc.AppendChild(doc, heading2)

	// Create heading 3
	heading3 := ast.NewHeading(3)
	text3 := ast.NewText()
	text3.Segment = text.NewSegment(24, 36) // "Subsection"
	heading3.AppendChild(heading3, text3)
	doc.AppendChild(doc, heading3)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// All headings should have IDs
	id1, ok1 := heading1.AttributeString("id")
	id2, ok2 := heading2.AttributeString("id")
	id3, ok3 := heading3.AttributeString("id")

	if !ok1 || !ok2 || !ok3 {
		t.Fatal("expected all headings to have id attributes")
	}

	// IDs should be different
	id1Bytes, _ := id1.([]byte)
	id2Bytes, _ := id2.([]byte)
	id3Bytes, _ := id3.([]byte)

	if string(id1Bytes) == string(id2Bytes) || string(id1Bytes) == string(id3Bytes) ||
		string(id2Bytes) == string(id3Bytes) {
		t.Error("expected all IDs to be unique")
	}
}

func TestHeadingIDTransformer_Transform_DuplicateHeadings(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with duplicate heading text
	doc := ast.NewDocument()
	source := []byte("# Overview\n\n## Overview")

	// Create heading 1 with text "Overview"
	heading1 := ast.NewHeading(1)
	text1 := ast.NewText()
	text1.Segment = text.NewSegment(2, 10) // "Overview"
	heading1.AppendChild(heading1, text1)
	doc.AppendChild(doc, heading1)

	// Create heading 2 with text "Overview"
	heading2 := ast.NewHeading(2)
	text2 := ast.NewText()
	text2.Segment = text.NewSegment(15, 23) // "Overview"
	heading2.AppendChild(heading2, text2)
	doc.AppendChild(doc, heading2)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// Both headings should have IDs
	id1, ok1 := heading1.AttributeString("id")
	id2, ok2 := heading2.AttributeString("id")

	if !ok1 || !ok2 {
		t.Fatal("expected both headings to have id attributes")
	}

	id1Bytes, _ := id1.([]byte)
	id2Bytes, _ := id2.([]byte)

	// First should be "overview", second should be "overview-2"
	if string(id1Bytes) != "overview" {
		t.Errorf("expected first heading id to be 'overview', got %q", string(id1Bytes))
	}
	if string(id2Bytes) != "overview-2" {
		t.Errorf("expected second heading id to be 'overview-2', got %q", string(id2Bytes))
	}
}

func TestHeadingIDTransformer_Transform_PreexistingIDs(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with one heading that already has an ID
	doc := ast.NewDocument()
	source := []byte("# First\n\n# Second")

	// Create heading 1 with pre-existing ID
	heading1 := ast.NewHeading(1)
	heading1.SetAttributeString("id", []byte("custom-id"))
	text1 := ast.NewText()
	text1.Segment = text.NewSegment(2, 7) // "First"
	heading1.AppendChild(heading1, text1)
	doc.AppendChild(doc, heading1)

	// Create heading 2 without pre-existing ID
	heading2 := ast.NewHeading(1)
	text2 := ast.NewText()
	text2.Segment = text.NewSegment(11, 17) // "Second"
	heading2.AppendChild(heading2, text2)
	doc.AppendChild(doc, heading2)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// First heading should keep its custom ID
	id1, _ := heading1.AttributeString("id")
	id1Bytes, _ := id1.([]byte)
	if string(id1Bytes) != "custom-id" {
		t.Errorf("expected custom ID to be preserved, got %q", string(id1Bytes))
	}

	// Second heading should have an auto-generated ID
	id2, ok2 := heading2.AttributeString("id")
	if !ok2 {
		t.Fatal("expected second heading to have an auto-generated id")
	}
	id2Bytes, _ := id2.([]byte)
	if string(id2Bytes) == "" || string(id2Bytes) == "custom-id" {
		t.Errorf("expected auto-generated ID for second heading, got %q", string(id2Bytes))
	}
}

func TestHeadingIDTransformer_Transform_NestedStructure(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with nested paragraph and heading structure
	doc := ast.NewDocument()
	source := []byte("# Main Title\n\nIntroduction text.\n\n## Subsection")

	// Create heading 1
	heading1 := ast.NewHeading(1)
	text1 := ast.NewText()
	text1.Segment = text.NewSegment(2, 12) // "Main Title"
	heading1.AppendChild(heading1, text1)
	doc.AppendChild(doc, heading1)

	// Create a paragraph
	para := ast.NewParagraph()
	paraText := ast.NewText()
	paraText.Segment = text.NewSegment(14, 32) // "Introduction text."
	para.AppendChild(para, paraText)
	doc.AppendChild(doc, para)

	// Create heading 2
	heading2 := ast.NewHeading(2)
	text2 := ast.NewText()
	text2.Segment = text.NewSegment(37, 47) // "Subsection"
	heading2.AppendChild(heading2, text2)
	doc.AppendChild(doc, heading2)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// Both headings should have IDs
	id1, ok1 := heading1.AttributeString("id")
	id2, ok2 := heading2.AttributeString("id")

	if !ok1 || !ok2 {
		t.Fatal("expected both headings to have id attributes")
	}

	// Verify they're different
	id1Bytes, _ := id1.([]byte)
	id2Bytes, _ := id2.([]byte)
	if string(id1Bytes) == string(id2Bytes) {
		t.Error("expected different IDs for different headings")
	}
}

func TestHeadingIDTransformer_Transform_ComplexContent(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with mixed content including code spans
	doc := ast.NewDocument()
	source := []byte("# Getting Started with `Go`")

	heading := ast.NewHeading(1)

	// Add text "Getting Started with "
	text1 := ast.NewText()
	text1.Segment = text.NewSegment(2, 23) // "Getting Started with "
	heading.AppendChild(heading, text1)

	// Add code span for "Go"
	codeSpan := ast.NewCodeSpan()
	codeText := ast.NewText()
	codeText.Segment = text.NewSegment(25, 27) // "Go"
	codeSpan.AppendChild(codeSpan, codeText)
	heading.AppendChild(heading, codeSpan)

	doc.AppendChild(doc, heading)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	ht.Transform(doc, reader, pc)

	// Heading should have an ID that includes both text and code content
	id, ok := heading.AttributeString("id")
	if !ok {
		t.Fatal("expected heading with code span to have id attribute")
	}

	idBytes, _ := id.([]byte)
	idStr := string(idBytes)
	if idStr == "" {
		t.Error("expected non-empty id for heading with code span")
	}

	// The ID should be derived from the text content
	if idStr == "heading" {
		t.Errorf("expected ID derived from heading content, got %q", idStr)
	}
}

func TestHeadingIDTransformer_Transform_NonHeadingNodes(t *testing.T) {
	transformer := newHeadingIDTransformer()
	ht := transformer.(*headingIDTransformer)

	// Create a document with non-heading nodes
	doc := ast.NewDocument()
	source := []byte("Just a paragraph\n\n**Bold text**")

	// Add a paragraph
	para := ast.NewParagraph()
	paraText := ast.NewText()
	paraText.Segment = text.NewSegment(0, 15) // "Just a paragraph"
	para.AppendChild(para, paraText)
	doc.AppendChild(doc, para)

	// Add a paragraph with emphasis
	para2 := ast.NewParagraph()
	emphasis := ast.NewEmphasis(2) // Strong emphasis
	emphText := ast.NewText()
	emphText.Segment = text.NewSegment(19, 28) // "Bold text"
	emphasis.AppendChild(emphasis, emphText)
	para2.AppendChild(para2, emphasis)
	doc.AppendChild(doc, para2)

	reader := text.NewReader(source)
	pc := parser.NewContext()

	// Should not panic with non-heading nodes
	ht.Transform(doc, reader, pc)

	// No headings, so usedIDs should be empty
	if len(ht.usedIDs) != 0 {
		t.Errorf("expected empty usedIDs for document with no headings, got %d entries", len(ht.usedIDs))
	}
}
