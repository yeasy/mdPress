package crossref

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TestNewResolver tests creating a new resolver
func TestNewResolver(t *testing.T) {
	r := NewResolver()
	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	refs := r.GetAllReferences()
	if len(refs) != 0 {
		t.Errorf("new resolver should have no references, got %d", len(refs))
	}
}

// TestRegisterFigure tests registering figure references
func TestRegisterFigure(t *testing.T) {
	r := NewResolver()

	n1 := r.RegisterFigure("fig1", "第一张图")
	if n1 != 1 {
		t.Errorf("first figure number should be 1, got %d", n1)
	}

	n2 := r.RegisterFigure("fig2", "第二张图")
	if n2 != 2 {
		t.Errorf("second figure number should be 2, got %d", n2)
	}

	// Re-registering the same ID should return the same number
	n1again := r.RegisterFigure("fig1", "重复注册")
	if n1again != 1 {
		t.Errorf("re-registration should return original number 1, got %d", n1again)
	}
}

// TestRegisterTable tests registering table references
func TestRegisterTable(t *testing.T) {
	r := NewResolver()

	n1 := r.RegisterTable("tab1", "表格一")
	n2 := r.RegisterTable("tab2", "表格二")

	if n1 != 1 || n2 != 2 {
		t.Errorf("table numbering error: got %d, %d, want 1, 2", n1, n2)
	}
}

// TestRegisterSection tests registering section references
func TestRegisterSection(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("background", "背景", 2)
	r.RegisterSection("details", "细节", 2)

	ref, err := r.Resolve("intro")
	if err != nil {
		t.Fatalf("failed to resolve 'intro': %v", err)
	}
	if ref.Type != TypeSection {
		t.Errorf("wrong reference type: got %v, want %v", ref.Type, TypeSection)
	}
}

// TestResolve tests reference resolution
func TestResolve(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig_arch", "架构图")
	r.RegisterTable("tab_compare", "对比表")
	r.RegisterSection("ch1", "第一章", 1)

	tests := []struct {
		id       string
		wantType ReferenceType
		wantErr  bool
	}{
		{"fig_arch", TypeFigure, false},
		{"tab_compare", TypeTable, false},
		{"ch1", TypeSection, false},
		{"nonexistent", "", true},
	}

	for _, tt := range tests {
		ref, err := r.Resolve(tt.id)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Resolve(%q) should return an error", tt.id)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q) returned unexpected error: %v", tt.id, err)
			continue
		}
		if ref.Type != tt.wantType {
			t.Errorf("Resolve(%q).Type = %v, want %v", tt.id, ref.Type, tt.wantType)
		}
	}
}

// TestProcessHTML tests reference replacement in HTML
func TestProcessHTML(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig1", "示例图")
	r.RegisterTable("tab1", "示例表")

	input := "如 {{ref:fig1}} 所示，详见 {{ref:tab1}}。"
	result := r.ProcessHTML(input)

	if strings.Contains(result, "{{ref:fig1}}") {
		t.Error("figure reference was not replaced")
	}
	if strings.Contains(result, "{{ref:tab1}}") {
		t.Error("table reference was not replaced")
	}
	if !strings.Contains(result, "图1") {
		t.Error("result should contain '图1'")
	}
	if !strings.Contains(result, "表1") {
		t.Error("result should contain '表1'")
	}
	if !strings.Contains(result, `href="#fig1"`) {
		t.Error("result should contain fig1 anchor link")
	}
}

// TestProcessHTMLUnknownRef tests that unknown references are preserved
func TestProcessHTMLUnknownRef(t *testing.T) {
	r := NewResolver()
	input := "参见 {{ref:unknown_id}} 了解更多。"
	result := r.ProcessHTML(input)

	if !strings.Contains(result, "{{ref:unknown_id}}") {
		t.Error("unknown reference should preserve original placeholder")
	}
}

// TestProcessHTMLNoRefs tests HTML without references
func TestProcessHTMLNoRefs(t *testing.T) {
	r := NewResolver()
	input := "<p>这是一段普通文本，没有引用。</p>"
	result := r.ProcessHTML(input)
	if result != input {
		t.Errorf("HTML without references should not be modified: got %q", result)
	}
}

// TestAddCaptions tests adding figure and table captions
func TestAddCaptions(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig_demo", "演示图")
	r.RegisterTable("tab_demo", "演示表")

	// Figure caption
	figHTML := `<figure id="fig_demo"><img src="demo.png"></figure>`
	result := r.AddCaptions(figHTML)
	if !strings.Contains(result, "figcaption") {
		t.Error("should add figcaption to figure")
	}
	if !strings.Contains(result, "图1") {
		t.Error("figcaption should contain '图1'")
	}

	// Table caption
	tabHTML := `<table id="tab_demo"><tr><td>data</td></tr></table>`
	result = r.AddCaptions(tabHTML)
	if !strings.Contains(result, "caption") {
		t.Error("should add caption to table")
	}
	if !strings.Contains(result, "表1") {
		t.Error("caption should contain '表1'")
	}
}

// TestAddCaptionsNoDuplicate tests that captions are not added when already present
func TestAddCaptionsNoDuplicate(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig1", "图一")

	html := `<figure id="fig1"><img src="a.png"><figcaption>已有标题</figcaption></figure>`
	result := r.AddCaptions(html)

	count := strings.Count(result, "figcaption")
	if count != 2 { // opening tag + closing tag
		t.Errorf("existing figcaption should be kept as-is, actual tag count %d", count)
	}
	if strings.Contains(result, "图1") {
		t.Error("should not add numbered caption when figcaption already exists")
	}
}

// TestGetAllReferences tests retrieving all references
func TestGetAllReferences(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("f1", "图1")
	r.RegisterFigure("f2", "图2")
	r.RegisterTable("t1", "表1")
	r.RegisterSection("s1", "节1", 1)

	refs := r.GetAllReferences()
	if len(refs) != 4 {
		t.Errorf("total reference count error: got %d, want 4", len(refs))
	}
}

// TestReset tests reset functionality
func TestReset(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("f1", "图1")
	r.RegisterTable("t1", "表1")
	r.RegisterSection("s1", "节1", 1)

	r.Reset()

	refs := r.GetAllReferences()
	if len(refs) != 0 {
		t.Errorf("should have no references after reset: got %d", len(refs))
	}

	// After reset, numbering should restart from 1
	n := r.RegisterFigure("f_new", "新图")
	if n != 1 {
		t.Errorf("numbering should restart from 1 after reset: got %d", n)
	}
}

// TestConcurrentAccess tests thread safety of concurrent access
func TestConcurrentAccess(t *testing.T) {
	r := NewResolver()
	done := make(chan bool, 100)

	for i := 0; i < 50; i++ {
		go func(n int) {
			r.RegisterFigure("fig_concurrent", "并发图")
			r.RegisterTable("tab_concurrent", "并发表")
			_, _ = r.Resolve("fig_concurrent")
			r.ProcessHTML("{{ref:fig_concurrent}}")
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestEscapeHTML tests HTML escaping
func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"<script>", "&lt;script&gt;"},
		{`"quoted"`, "&quot;quoted&quot;"},
		{"a&b", "a&amp;b"},
		{"it's", "it&#39;s"},
	}

	for _, tt := range tests {
		got := utils.EscapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestSectionNumbering tests hierarchical section numbering
func TestSectionNumbering(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("ch1", "第一章", 1)
	r.RegisterSection("sec1_1", "第1.1节", 2)
	r.RegisterSection("sec1_2", "第1.2节", 2)
	r.RegisterSection("ch2", "第二章", 1)
	r.RegisterSection("sec2_1", "第2.1节", 2)

	ref, _ := r.Resolve("ch2")
	if ref.Number != 2 {
		t.Errorf("chapter 2 number should be 2, got %d", ref.Number)
	}

	ref, _ = r.Resolve("sec2_1")
	if ref.Number != 1 {
		t.Errorf("section 2.1 sibling number should be 1, got %d", ref.Number)
	}
}

// TestProcessHTMLSectionRef tests that section references generate links in the section 1.2 format
func TestProcessHTMLSectionRef(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("background", "背景", 2)
	r.RegisterSection("details", "细节", 2)

	// Test section reference processing
	input := "详见 {{ref:details}} 部分。"
	result := r.ProcessHTML(input)

	// Should not contain the original placeholder
	if strings.Contains(result, "{{ref:details}}") {
		t.Error("section reference placeholder should be replaced")
	}

	// Should contain section symbol and number
	if !strings.Contains(result, "§") {
		t.Error("section reference should contain the section symbol")
	}

	// Should contain hierarchical number
	if !strings.Contains(result, "1.2") {
		t.Error("section reference should contain hierarchical number 1.2")
	}

	// Should be formatted as a link
	if !strings.Contains(result, `href="#details"`) {
		t.Error("section reference should contain anchor link")
	}

	if !strings.Contains(result, `class="ref-section"`) {
		t.Error("section reference should contain correct CSS class")
	}
}

// TestRegisterSectionDeepNesting tests 4-level deep hierarchical section numbering
func TestRegisterSectionDeepNesting(t *testing.T) {
	r := NewResolver()

	// Register 4-level nested structure
	r.RegisterSection("ch1", "第一章", 1)
	r.RegisterSection("sec1_1", "第1.1节", 2)
	r.RegisterSection("subsec1_1_1", "第1.1.1小节", 3)
	r.RegisterSection("detail1_1_1_1", "第1.1.1.1条", 4)

	r.RegisterSection("ch2", "第二章", 1)
	r.RegisterSection("sec2_1", "第2.1节", 2)
	r.RegisterSection("subsec2_1_1", "第2.1.1小节", 3)
	r.RegisterSection("detail2_1_1_1", "第2.1.1.1条", 4)

	tests := []struct {
		id      string
		wantNum string
	}{
		{"ch1", "1"},
		{"sec1_1", "1.1"},
		{"subsec1_1_1", "1.1.1"},
		{"detail1_1_1_1", "1.1.1.1"},
		{"ch2", "2"},
		{"sec2_1", "2.1"},
		{"subsec2_1_1", "2.1.1"},
		{"detail2_1_1_1", "2.1.1.1"},
	}

	for _, tt := range tests {
		ref, err := r.Resolve(tt.id)
		if err != nil {
			t.Errorf("Resolve(%q) failed: %v", tt.id, err)
			continue
		}

		if ref.NumberStr != tt.wantNum {
			t.Errorf("NumberStr for %q = %q, want %q", tt.id, ref.NumberStr, tt.wantNum)
		}

		if ref.Type != TypeSection {
			t.Errorf("Type for %q = %v, want %v", tt.id, ref.Type, TypeSection)
		}
	}
}

// TestRegisterDuplicateSection tests idempotency of registering the same section ID twice
func TestRegisterDuplicateSection(t *testing.T) {
	r := NewResolver()

	// First registration
	r.RegisterSection("intro", "简介", 1)
	ref1, _ := r.Resolve("intro")

	// Second registration with same ID (should be ignored)
	r.RegisterSection("intro", "修改后的简介", 1)
	ref2, _ := r.Resolve("intro")

	// ID and number should remain unchanged
	if ref1.ID != ref2.ID {
		t.Error("duplicate registration should keep the same ID")
	}

	if ref1.Number != ref2.Number {
		t.Errorf("duplicate registration should keep the same number: got %d, then %d", ref1.Number, ref2.Number)
	}

	// Title should retain the original value (not updated)
	if ref2.Title != "简介" {
		t.Errorf("duplicate registration should not update title: got %q, want %q", ref2.Title, "简介")
	}

	// Total count should only increment once
	r.RegisterSection("ch2", "第二章", 1)
	ref3, _ := r.Resolve("ch2")
	if ref3.Number != 2 {
		t.Errorf("new section number should be 2, got %d", ref3.Number)
	}
}

// TestAddCaptionsUnregistered tests adding captions for unregistered figure/table IDs
func TestAddCaptionsUnregistered(t *testing.T) {
	r := NewResolver()

	// Only register fig1, not fig2 or tab1
	r.RegisterFigure("fig1", "已注册的图")

	// HTML with both registered and unregistered IDs
	html := `
	<figure id="fig1"><img src="a.png"></figure>
	<figure id="fig_unreg"><img src="b.png"></figure>
	<table id="tab_unreg"><tr><td>data</td></tr></table>
	`

	result := r.AddCaptions(html)

	// Registered ones should be processed
	if !strings.Contains(result, "figcaption") || !strings.Contains(result, "图1") {
		t.Error("registered figure should have caption added")
	}

	// Unregistered ones should remain unchanged
	if !strings.Contains(result, `id="fig_unreg"`) {
		t.Error("unregistered figure ID should be preserved")
	}

	if !strings.Contains(result, `id="tab_unreg"`) {
		t.Error("unregistered table ID should be preserved")
	}

	// Unregistered elements should not be modified
	origCount := strings.Count(html, `<figure id="fig_unreg">`)
	resultCount := strings.Count(result, `id="fig_unreg"`)
	if origCount != resultCount {
		t.Error("unregistered figures/tables should not be modified")
	}
}

// TestResolveSearchOrder tests resolution priority (figure > table > section)
func TestResolveSearchOrder(t *testing.T) {
	r := NewResolver()

	// Register different reference types with the same ID
	id := "item"

	// Though unlikely in practice, we test Resolve search order

	// Case 1: registered as figure only
	r.Reset()
	r.RegisterFigure(id, "这是一张图")
	ref, err := r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("should find figure reference")
	}

	// Case 2: registered as table and figure; figure should be found (higher priority)
	r.Reset()
	r.RegisterTable(id, "这是一张表")
	r.RegisterFigure(id, "这是一张图")
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("when both figure and table exist, figure should be returned first")
	}

	// Case 3: registered as section
	r.Reset()
	r.RegisterSection(id, "这是一章", 1)
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeSection {
		t.Error("should find section reference")
	}

	// Case 4: all three types (unrealistic but tests priority); figure should be returned
	r.Reset()
	r.RegisterSection(id, "章节", 1)
	r.RegisterTable(id, "表")
	r.RegisterFigure(id, "图")
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("when all three types exist, figure should be returned first")
	}
}

// TestProcessHTMLMultipleRefs tests processing multiple different reference types in a single HTML string
func TestProcessHTMLMultipleRefs(t *testing.T) {
	r := NewResolver()

	// Register different reference types
	r.RegisterFigure("fig1", "架构图")
	r.RegisterFigure("fig2", "流程图")
	r.RegisterTable("tab1", "性能对比")
	r.RegisterTable("tab2", "功能列表")
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("method", "方法", 2)

	// HTML with multiple mixed-type references
	input := `
	<p>如 {{ref:fig1}} 所示，系统架构如下。根据 {{ref:tab1}}，性能指标如下。</p>
	<p>详见 {{ref:intro}} 和 {{ref:method}} 获取更多信息。</p>
	<p>{{ref:fig2}} 展示了流程，{{ref:tab2}} 列出了所有功能。</p>
	`

	result := r.ProcessHTML(input)

	// Verify all placeholders were replaced
	if strings.Contains(result, "{{ref:") {
		t.Error("all reference placeholders should be replaced")
	}

	// Verify each type was replaced correctly
	if !strings.Contains(result, "图1") || !strings.Contains(result, "图2") {
		t.Error("both figure references should be processed")
	}

	if !strings.Contains(result, "表1") || !strings.Contains(result, "表2") {
		t.Error("both table references should be processed")
	}

	// Sections should display hierarchical numbering
	if !strings.Contains(result, "§1") || !strings.Contains(result, "§1.1") {
		t.Error("both section references should be processed")
	}

	// Verify link structure
	if !strings.Contains(result, `href="#fig1"`) || !strings.Contains(result, `href="#fig2"`) {
		t.Error("figure references should contain correct anchors")
	}

	if !strings.Contains(result, `href="#tab1"`) || !strings.Contains(result, `href="#tab2"`) {
		t.Error("table references should contain correct anchors")
	}

	if !strings.Contains(result, `href="#intro"`) || !strings.Contains(result, `href="#method"`) {
		t.Error("section references should contain correct anchors")
	}

	// Verify CSS classes are correct
	if !strings.Contains(result, `class="ref-figure"`) {
		t.Error("figure references should contain ref-figure class")
	}

	if !strings.Contains(result, `class="ref-table"`) {
		t.Error("table references should contain ref-table class")
	}

	if !strings.Contains(result, `class="ref-section"`) {
		t.Error("section references should contain ref-section class")
	}
}
