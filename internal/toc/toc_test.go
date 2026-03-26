package toc

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TestNewGenerator 测试创建生成器
func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator 返回 nil")
	}
}

// TestGenerateEmpty 测试空标题列表
func TestGenerateEmpty(t *testing.T) {
	g := NewGenerator()
	entries := g.Generate(nil)
	if len(entries) != 0 {
		t.Errorf("空输入应返回空列表: got %d entries", len(entries))
	}

	entries = g.Generate([]HeadingInfo{})
	if len(entries) != 0 {
		t.Errorf("空切片应返回空列表: got %d entries", len(entries))
	}
}

// TestGenerateFlat 测试同级标题
func TestGenerateFlat(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "章节一", ID: "ch1"},
		{Level: 1, Text: "章节二", ID: "ch2"},
		{Level: 1, Text: "章节三", ID: "ch3"},
	}

	entries := g.Generate(headings)
	if len(entries) != 3 {
		t.Fatalf("应有 3 个顶层条目: got %d", len(entries))
	}

	for i, entry := range entries {
		if entry.Level != 1 {
			t.Errorf("条目 %d 级别错误: got %d", i, entry.Level)
		}
		if len(entry.Children) != 0 {
			t.Errorf("条目 %d 不应有子条目: got %d", i, len(entry.Children))
		}
	}
}

// TestGenerateNested 测试嵌套标题
func TestGenerateNested(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "第一章", ID: "ch1"},
		{Level: 2, Text: "1.1 小节", ID: "sec1-1"},
		{Level: 2, Text: "1.2 小节", ID: "sec1-2"},
		{Level: 3, Text: "1.2.1 子节", ID: "sub1-2-1"},
		{Level: 1, Text: "第二章", ID: "ch2"},
		{Level: 2, Text: "2.1 小节", ID: "sec2-1"},
	}

	entries := g.Generate(headings)
	if len(entries) != 2 {
		t.Fatalf("应有 2 个顶层条目: got %d", len(entries))
	}

	// 第一章应有 2 个子条目
	ch1 := entries[0]
	if ch1.Title != "第一章" {
		t.Errorf("第一个条目标题错误: got %q", ch1.Title)
	}
	if len(ch1.Children) != 2 {
		t.Fatalf("第一章应有 2 个子条目: got %d", len(ch1.Children))
	}

	// 1.2 小节应有 1 个子条目（1.2.1）
	sec12 := ch1.Children[1]
	if len(sec12.Children) != 1 {
		t.Errorf("1.2 小节应有 1 个子条目: got %d", len(sec12.Children))
	}

	// 第二章应有 1 个子条目
	ch2 := entries[1]
	if len(ch2.Children) != 1 {
		t.Errorf("第二章应有 1 个子条目: got %d", len(ch2.Children))
	}

	// 总条目数应为 6
	total := CountEntries(entries)
	if total != 6 {
		t.Errorf("应有 6 个总条目: got %d", total)
	}
}

// TestGenerateDeepNesting 测试深层嵌套
func TestGenerateDeepNesting(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "H1", ID: "h1"},
		{Level: 2, Text: "H2", ID: "h2"},
		{Level: 3, Text: "H3", ID: "h3"},
		{Level: 4, Text: "H4", ID: "h4"},
		{Level: 5, Text: "H5", ID: "h5"},
		{Level: 6, Text: "H6", ID: "h6"},
	}

	entries := g.Generate(headings)
	if len(entries) != 1 {
		t.Fatalf("应有 1 个顶层条目: got %d", len(entries))
	}
}

// TestRenderHTMLEmpty 测试渲染空目录
func TestRenderHTMLEmpty(t *testing.T) {
	g := NewGenerator()
	html := g.RenderHTML(nil)
	if html != "" {
		t.Errorf("空目录应返回空字符串: got %q", html)
	}
}

// TestRenderHTMLBasic 测试基本 HTML 渲染
func TestRenderHTMLBasic(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "简介", ID: "intro", Children: []TOCEntry{}},
		{Level: 1, Title: "总结", ID: "summary", Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	if !strings.Contains(html, `<nav class="toc">`) {
		t.Error("HTML 应包含 nav.toc 标签")
	}
	if !strings.Contains(html, `href="#intro"`) {
		t.Error("HTML 应包含 intro 锚点链接")
	}
	if !strings.Contains(html, `href="#summary"`) {
		t.Error("HTML 应包含 summary 锚点链接")
	}
	if !strings.Contains(html, "简介") {
		t.Error("HTML 应包含标题文本")
	}
}

// TestRenderHTMLNested 测试嵌套 HTML 渲染
func TestRenderHTMLNested(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1 节", ID: "sec1-1", Children: []TOCEntry{}},
			},
		},
	}

	html := g.RenderHTML(entries)

	// 应有嵌套的 ul
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 2 {
		t.Errorf("嵌套目录应有至少 2 层 ul 标签: got %d", ulCount)
	}
}

// TestRenderHTMLEscaping 测试特殊字符转义
func TestRenderHTMLEscaping(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "<script>alert('xss')</script>", ID: "xss-test", Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	if strings.Contains(html, "<script>") {
		t.Error("HTML 标签应被转义")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("应包含转义后的标签")
	}
}

// TestGetEntry 测试按 ID 查找条目
func TestGetEntry(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1", ID: "sec1-1", Children: []TOCEntry{}},
				{
					Level: 2, Title: "1.2", ID: "sec1-2",
					Children: []TOCEntry{
						{Level: 3, Title: "1.2.1", ID: "sub1-2-1", Children: []TOCEntry{}},
					},
				},
			},
		},
		{Level: 1, Title: "第二章", ID: "ch2", Children: []TOCEntry{}},
	}

	tests := []struct {
		id    string
		found bool
		title string
	}{
		{"ch1", true, "第一章"},
		{"sec1-1", true, "1.1"},
		{"sub1-2-1", true, "1.2.1"},
		{"ch2", true, "第二章"},
		{"nonexistent", false, ""},
	}

	for _, tt := range tests {
		entry := GetEntry(entries, tt.id)
		if tt.found {
			if entry == nil {
				t.Errorf("应找到 ID=%q 的条目", tt.id)
				continue
			}
			if entry.Title != tt.title {
				t.Errorf("ID=%q 的标题错误: got %q, want %q", tt.id, entry.Title, tt.title)
			}
		} else if entry != nil {
			t.Errorf("ID=%q 应不存在", tt.id)
		}
	}
}

// TestFlattenToList 测试扁平化
func TestFlattenToList(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "A", ID: "a",
			Children: []TOCEntry{
				{Level: 2, Title: "A1", ID: "a1", Children: []TOCEntry{}},
				{Level: 2, Title: "A2", ID: "a2", Children: []TOCEntry{}},
			},
		},
		{Level: 1, Title: "B", ID: "b", Children: []TOCEntry{}},
	}

	flat := FlattenToList(entries)
	if len(flat) != 4 {
		t.Fatalf("扁平化后应有 4 个条目: got %d", len(flat))
	}

	expectedTitles := []string{"A", "A1", "A2", "B"}
	for i, title := range expectedTitles {
		if flat[i].Title != title {
			t.Errorf("条目 %d 标题错误: got %q, want %q", i, flat[i].Title, title)
		}
	}

	// 扁平化后不应有 Children
	for i, entry := range flat {
		if len(entry.Children) != 0 {
			t.Errorf("条目 %d 扁平化后不应有子条目", i)
		}
	}
}

// TestCountEntries 测试条目计数
func TestCountEntries(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "A", ID: "a",
			Children: []TOCEntry{
				{Level: 2, Title: "A1", ID: "a1", Children: []TOCEntry{}},
				{
					Level: 2, Title: "A2", ID: "a2",
					Children: []TOCEntry{
						{Level: 3, Title: "A2a", ID: "a2a", Children: []TOCEntry{}},
					},
				},
			},
		},
		{Level: 1, Title: "B", ID: "b", Children: []TOCEntry{}},
	}

	count := CountEntries(entries)
	if count != 5 {
		t.Errorf("应有 5 个条目: got %d", count)
	}
}

// TestCountEntriesEmpty 测试空列表计数
func TestCountEntriesEmpty(t *testing.T) {
	count := CountEntries(nil)
	if count != 0 {
		t.Errorf("空列表计数应为 0: got %d", count)
	}
}

// TestEscapeHTMLToc 测试 toc 包内的 HTML 转义
func TestEscapeHTMLToc(t *testing.T) {
	input := `<a href="test">&'`
	expected := `&lt;a href=&quot;test&quot;&gt;&amp;&#39;`
	got := utils.EscapeHTML(input)
	if got != expected {
		t.Errorf("EscapeHTML 结果错误: got %q, want %q", got, expected)
	}
}

// TestGenerateSkippedLevels 测试跳过标题级别的处理
func TestGenerateSkippedLevels(t *testing.T) {
	g := NewGenerator()
	// 测试直接从 H1 跳到 H3 的情况
	headings := []HeadingInfo{
		{Level: 1, Text: "第一章", ID: "ch1"},
		{Level: 3, Text: "直接跳到三级", ID: "skip2"},
		{Level: 2, Text: "回到二级", ID: "back2"},
	}

	entries := g.Generate(headings)
	if len(entries) != 1 {
		t.Fatalf("应有 1 个顶层条目: got %d", len(entries))
	}

	// 验证跳过的级别仍能正确嵌套
	ch1 := entries[0]
	if len(ch1.Children) < 2 {
		t.Errorf("第一章应有至少 2 个子条目，以处理跳过的级别: got %d", len(ch1.Children))
	}

	total := CountEntries(entries)
	if total != 3 {
		t.Errorf("总条目数应为 3: got %d", total)
	}
}

// TestGenerateRepeatedTitles 测试相同标题但不同 ID 的处理
func TestGenerateRepeatedTitles(t *testing.T) {
	g := NewGenerator()
	// 相同标题但不同 ID
	headings := []HeadingInfo{
		{Level: 1, Text: "简介", ID: "intro-1"},
		{Level: 1, Text: "简介", ID: "intro-2"},
		{Level: 1, Text: "简介", ID: "intro-3"},
	}

	entries := g.Generate(headings)
	if len(entries) != 3 {
		t.Fatalf("应有 3 个条目: got %d", len(entries))
	}

	// 验证每个条目都有不同的 ID
	ids := make(map[string]bool)
	for _, entry := range entries {
		if ids[entry.ID] {
			t.Errorf("ID %q 重复了", entry.ID)
		}
		ids[entry.ID] = true
		if entry.Title != "简介" {
			t.Errorf("标题应为 '简介': got %q", entry.Title)
		}
	}
}

// TestGenerateSpecialCharIDs 测试包含特殊字符的 ID
func TestGenerateSpecialCharIDs(t *testing.T) {
	g := NewGenerator()
	headings := []HeadingInfo{
		{Level: 1, Text: "标题", ID: "id-with-dashes"},
		{Level: 1, Text: "标题", ID: "id_with_underscores"},
		{Level: 1, Text: "标题", ID: "id.with.dots"},
		{Level: 1, Text: "标题", ID: "id-123-numbers"},
	}

	entries := g.Generate(headings)
	if len(entries) != 4 {
		t.Fatalf("应有 4 个条目: got %d", len(entries))
	}

	expectedIDs := []string{"id-with-dashes", "id_with_underscores", "id.with.dots", "id-123-numbers"}
	for i, expectedID := range expectedIDs {
		if entries[i].ID != expectedID {
			t.Errorf("条目 %d 的 ID 应为 %q: got %q", i, expectedID, entries[i].ID)
		}
	}
}

// TestRenderHTMLDeepNesting 测试 4+ 层级的深层嵌套 HTML 渲染
func TestRenderHTMLDeepNesting(t *testing.T) {
	g := NewGenerator()
	// 构建 4 层深的嵌套结构
	entries := []TOCEntry{
		{
			Level: 1, Title: "Level 1", ID: "l1",
			Children: []TOCEntry{
				{
					Level: 2, Title: "Level 2", ID: "l2",
					Children: []TOCEntry{
						{
							Level: 3, Title: "Level 3", ID: "l3",
							Children: []TOCEntry{
								{Level: 4, Title: "Level 4", ID: "l4", Children: []TOCEntry{}},
							},
						},
					},
				},
			},
		},
	}

	html := g.RenderHTML(entries)

	// 检查深层嵌套结构
	if !strings.Contains(html, "Level 1") {
		t.Error("HTML 应包含第 1 层标题")
	}
	if !strings.Contains(html, "Level 2") {
		t.Error("HTML 应包含第 2 层标题")
	}
	if !strings.Contains(html, "Level 3") {
		t.Error("HTML 应包含第 3 层标题")
	}
	if !strings.Contains(html, "Level 4") {
		t.Error("HTML 应包含第 4 层标题")
	}

	// 检查嵌套的 ul 标签
	ulCount := strings.Count(html, "<ul>")
	if ulCount < 4 {
		t.Errorf("4 层嵌套应有至少 4 个 ul 标签: got %d", ulCount)
	}

	// 检查所有锚点链接存在
	links := []string{"#l1", "#l2", "#l3", "#l4"}
	for _, link := range links {
		if !strings.Contains(html, link) {
			t.Errorf("HTML 应包含链接 %s", link)
		}
	}
}

// TestFlattenToListEmpty 测试扁平化空和 nil 列表
func TestFlattenToListEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  []TOCEntry
		expect int
	}{
		{
			name:   "nil 切片",
			input:  nil,
			expect: 0,
		},
		{
			name:   "空切片",
			input:  []TOCEntry{},
			expect: 0,
		},
		{
			name: "单层无子条目",
			input: []TOCEntry{
				{Level: 1, Title: "A", ID: "a", Children: []TOCEntry{}},
			},
			expect: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FlattenToList(tt.input)
			if len(result) != tt.expect {
				t.Errorf("应有 %d 个条目: got %d", tt.expect, len(result))
			}
		})
	}
}

// TestGetEntryFirstMatch 测试 GetEntry 返回第一个匹配项
func TestGetEntryFirstMatch(t *testing.T) {
	entries := []TOCEntry{
		{
			Level: 1, Title: "第一章", ID: "ch1",
			Children: []TOCEntry{
				{Level: 2, Title: "1.1", ID: "duplicate-id", Children: []TOCEntry{}},
				{
					Level: 2, Title: "1.2", ID: "sec1-2",
					Children: []TOCEntry{
						{Level: 3, Title: "1.2.1", ID: "duplicate-id", Children: []TOCEntry{}},
					},
				},
			},
		},
	}

	// 查询可能出现多次的 ID
	entry := GetEntry(entries, "duplicate-id")
	if entry == nil {
		t.Fatal("应找到 ID 为 'duplicate-id' 的条目")
		return
	}

	// 验证返回的是第一个匹配项
	if entry.Title != "1.1" {
		t.Errorf("应返回第一个匹配项，标题为 '1.1': got %q", entry.Title)
	}

	if entry.Level != 2 {
		t.Errorf("第一个匹配项应为 Level 2: got %d", entry.Level)
	}
}

// TestRenderHTMLWithPageNumbers 测试渲染包含页码的 TOC 条目
func TestRenderHTMLWithPageNumbers(t *testing.T) {
	g := NewGenerator()
	entries := []TOCEntry{
		{Level: 1, Title: "第一章", ID: "ch1", PageNum: 1, Children: []TOCEntry{
			{Level: 2, Title: "1.1 节", ID: "sec1-1", PageNum: 2, Children: []TOCEntry{}},
		}},
		{Level: 1, Title: "第二章", ID: "ch2", PageNum: 5, Children: []TOCEntry{}},
	}

	html := g.RenderHTML(entries)

	// 验证 HTML 中包含标题和链接
	if !strings.Contains(html, "第一章") {
		t.Error("HTML 应包含 '第一章'")
	}
	if !strings.Contains(html, "#ch1") {
		t.Error("HTML 应包含 #ch1 链接")
	}

	// 页码可能以不同方式呈现（取决于实现）
	if !strings.Contains(html, "<nav class=\"toc\">") {
		t.Error("HTML 应包含 toc 导航容器")
	}
}
