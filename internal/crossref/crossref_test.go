package crossref

import (
	"strings"
	"testing"

	"github.com/yeasy/mdpress/pkg/utils"
)

// TestNewResolver 测试创建新解析器
func TestNewResolver(t *testing.T) {
	r := NewResolver()
	if r == nil {
		t.Fatal("NewResolver 返回 nil")
	}
	refs := r.GetAllReferences()
	if len(refs) != 0 {
		t.Errorf("新解析器应无引用，got %d", len(refs))
	}
}

// TestRegisterFigure 测试注册图片引用
func TestRegisterFigure(t *testing.T) {
	r := NewResolver()

	n1 := r.RegisterFigure("fig1", "第一张图")
	if n1 != 1 {
		t.Errorf("第一张图编号应为 1，got %d", n1)
	}

	n2 := r.RegisterFigure("fig2", "第二张图")
	if n2 != 2 {
		t.Errorf("第二张图编号应为 2，got %d", n2)
	}

	// 重复注册同一 ID 应返回相同编号
	n1again := r.RegisterFigure("fig1", "重复注册")
	if n1again != 1 {
		t.Errorf("重复注册应返回原编号 1，got %d", n1again)
	}
}

// TestRegisterTable 测试注册表格引用
func TestRegisterTable(t *testing.T) {
	r := NewResolver()

	n1 := r.RegisterTable("tab1", "表格一")
	n2 := r.RegisterTable("tab2", "表格二")

	if n1 != 1 || n2 != 2 {
		t.Errorf("表格编号错误: got %d, %d, want 1, 2", n1, n2)
	}
}

// TestRegisterSection 测试注册章节引用
func TestRegisterSection(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("background", "背景", 2)
	r.RegisterSection("details", "细节", 2)

	ref, err := r.Resolve("intro")
	if err != nil {
		t.Fatalf("查找 'intro' 失败: %v", err)
	}
	if ref.Type != TypeSection {
		t.Errorf("引用类型错误: got %v, want %v", ref.Type, TypeSection)
	}
}

// TestResolve 测试引用查找
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
				t.Errorf("Resolve(%q) 应返回错误", tt.id)
			}
			continue
		}
		if err != nil {
			t.Errorf("Resolve(%q) 返回意外错误: %v", tt.id, err)
			continue
		}
		if ref.Type != tt.wantType {
			t.Errorf("Resolve(%q).Type = %v, want %v", tt.id, ref.Type, tt.wantType)
		}
	}
}

// TestProcessHTML 测试 HTML 中的引用替换
func TestProcessHTML(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig1", "示例图")
	r.RegisterTable("tab1", "示例表")

	input := "如 {{ref:fig1}} 所示，详见 {{ref:tab1}}。"
	result := r.ProcessHTML(input)

	if strings.Contains(result, "{{ref:fig1}}") {
		t.Error("图片引用未被替换")
	}
	if strings.Contains(result, "{{ref:tab1}}") {
		t.Error("表格引用未被替换")
	}
	if !strings.Contains(result, "图1") {
		t.Error("结果中应包含 '图1'")
	}
	if !strings.Contains(result, "表1") {
		t.Error("结果中应包含 '表1'")
	}
	if !strings.Contains(result, `href="#fig1"`) {
		t.Error("结果中应包含 fig1 锚点链接")
	}
}

// TestProcessHTMLUnknownRef 测试未知引用保留原文
func TestProcessHTMLUnknownRef(t *testing.T) {
	r := NewResolver()
	input := "参见 {{ref:unknown_id}} 了解更多。"
	result := r.ProcessHTML(input)

	if !strings.Contains(result, "{{ref:unknown_id}}") {
		t.Error("未知引用应保留原占位符")
	}
}

// TestProcessHTMLNoRefs 测试无引用的 HTML
func TestProcessHTMLNoRefs(t *testing.T) {
	r := NewResolver()
	input := "<p>这是一段普通文本，没有引用。</p>"
	result := r.ProcessHTML(input)
	if result != input {
		t.Errorf("无引用的 HTML 不应被修改: got %q", result)
	}
}

// TestAddCaptions 测试图表标题添加
func TestAddCaptions(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig_demo", "演示图")
	r.RegisterTable("tab_demo", "演示表")

	// 图片标题
	figHTML := `<figure id="fig_demo"><img src="demo.png"></figure>`
	result := r.AddCaptions(figHTML)
	if !strings.Contains(result, "figcaption") {
		t.Error("应为 figure 添加 figcaption")
	}
	if !strings.Contains(result, "图1") {
		t.Error("figcaption 中应包含 '图1'")
	}

	// 表格标题
	tabHTML := `<table id="tab_demo"><tr><td>data</td></tr></table>`
	result = r.AddCaptions(tabHTML)
	if !strings.Contains(result, "caption") {
		t.Error("应为 table 添加 caption")
	}
	if !strings.Contains(result, "表1") {
		t.Error("caption 中应包含 '表1'")
	}
}

// TestAddCaptionsNoDuplicate 测试不重复添加标题
func TestAddCaptionsNoDuplicate(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("fig1", "图一")

	html := `<figure id="fig1"><img src="a.png"><figcaption>已有标题</figcaption></figure>`
	result := r.AddCaptions(html)

	count := strings.Count(result, "figcaption")
	if count != 2 { // 开标签 + 闭标签
		t.Errorf("已有 figcaption 时应只保留原有一个 figcaption，实际标签计数 %d", count)
	}
	if strings.Contains(result, "图1") {
		t.Error("已有 figcaption 时不应再添加编号标题")
	}
}

// TestGetAllReferences 测试获取所有引用
func TestGetAllReferences(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("f1", "图1")
	r.RegisterFigure("f2", "图2")
	r.RegisterTable("t1", "表1")
	r.RegisterSection("s1", "节1", 1)

	refs := r.GetAllReferences()
	if len(refs) != 4 {
		t.Errorf("总引用数错误: got %d, want 4", len(refs))
	}
}

// TestReset 测试重置
func TestReset(t *testing.T) {
	r := NewResolver()
	r.RegisterFigure("f1", "图1")
	r.RegisterTable("t1", "表1")
	r.RegisterSection("s1", "节1", 1)

	r.Reset()

	refs := r.GetAllReferences()
	if len(refs) != 0 {
		t.Errorf("重置后应无引用: got %d", len(refs))
	}

	// 重置后编号应从头开始
	n := r.RegisterFigure("f_new", "新图")
	if n != 1 {
		t.Errorf("重置后编号应从 1 开始: got %d", n)
	}
}

// TestConcurrentAccess 测试并发访问安全性
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

// TestEscapeHTML 测试 HTML 转义
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

// TestSectionNumbering 测试章节分层编号
func TestSectionNumbering(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("ch1", "第一章", 1)
	r.RegisterSection("sec1_1", "第1.1节", 2)
	r.RegisterSection("sec1_2", "第1.2节", 2)
	r.RegisterSection("ch2", "第二章", 1)
	r.RegisterSection("sec2_1", "第2.1节", 2)

	ref, _ := r.Resolve("ch2")
	if ref.Number != 2 {
		t.Errorf("第二章编号应为 2, got %d", ref.Number)
	}

	ref, _ = r.Resolve("sec2_1")
	if ref.Number != 1 {
		t.Errorf("第2.1节同级编号应为 1, got %d", ref.Number)
	}
}

// TestProcessHTMLSectionRef 测试章节引用在 HTML 中生成 §1.2 格式的链接
func TestProcessHTMLSectionRef(t *testing.T) {
	r := NewResolver()
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("background", "背景", 2)
	r.RegisterSection("details", "细节", 2)

	// 测试章节引用的处理
	input := "详见 {{ref:details}} 部分。"
	result := r.ProcessHTML(input)

	// 不应该包含原占位符
	if strings.Contains(result, "{{ref:details}}") {
		t.Error("章节引用占位符应被替换")
	}

	// 应该包含 § 符号和编号
	if !strings.Contains(result, "§") {
		t.Error("章节引用应包含 § 符号")
	}

	// 应该包含分层编号
	if !strings.Contains(result, "1.2") {
		t.Error("章节引用应包含分层编号 1.2")
	}

	// 应该是链接格式
	if !strings.Contains(result, `href="#details"`) {
		t.Error("章节引用应包含锚点链接")
	}

	if !strings.Contains(result, `class="ref-section"`) {
		t.Error("章节引用应包含正确的 CSS 类")
	}
}

// TestRegisterSectionDeepNesting 测试 4 级深度的章节分层编号
func TestRegisterSectionDeepNesting(t *testing.T) {
	r := NewResolver()

	// 注册 4 级嵌套结构
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

// TestRegisterDuplicateSection 测试重复注册同一章节 ID 的幂等性
func TestRegisterDuplicateSection(t *testing.T) {
	r := NewResolver()

	// 第一次注册
	r.RegisterSection("intro", "简介", 1)
	ref1, _ := r.Resolve("intro")

	// 第二次注册相同 ID（应被忽略）
	r.RegisterSection("intro", "修改后的简介", 1)
	ref2, _ := r.Resolve("intro")

	// ID 和编号应该保持不变
	if ref1.ID != ref2.ID {
		t.Error("重复注册应保持相同的 ID")
	}

	if ref1.Number != ref2.Number {
		t.Errorf("重复注册应保持相同的编号: got %d, then %d", ref1.Number, ref2.Number)
	}

	// 标题应该保持原值（未被更新）
	if ref2.Title != "简介" {
		t.Errorf("重复注册不应更新标题: got %q, want %q", ref2.Title, "简介")
	}

	// 总计数应该只增加一次
	r.RegisterSection("ch2", "第二章", 1)
	ref3, _ := r.Resolve("ch2")
	if ref3.Number != 2 {
		t.Errorf("新章节编号应为 2, got %d", ref3.Number)
	}
}

// TestAddCaptionsUnregistered 测试为未注册的图表 ID 添加标题
func TestAddCaptionsUnregistered(t *testing.T) {
	r := NewResolver()

	// 只注册 fig1，不注册 fig2 和 tab1
	r.RegisterFigure("fig1", "已注册的图")

	// 包含已注册和未注册 ID 的 HTML
	html := `
	<figure id="fig1"><img src="a.png"></figure>
	<figure id="fig_unreg"><img src="b.png"></figure>
	<table id="tab_unreg"><tr><td>data</td></tr></table>
	`

	result := r.AddCaptions(html)

	// 已注册的应被处理
	if !strings.Contains(result, "figcaption") || !strings.Contains(result, "图1") {
		t.Error("已注册的图应添加标题")
	}

	// 未注册的应保持原样
	if !strings.Contains(result, `id="fig_unreg"`) {
		t.Error("未注册的图 ID 应保留")
	}

	if !strings.Contains(result, `id="tab_unreg"`) {
		t.Error("未注册的表 ID 应保留")
	}

	// 未注册元素不应被修改
	origCount := strings.Count(html, `<figure id="fig_unreg">`)
	resultCount := strings.Count(result, `id="fig_unreg"`)
	if origCount != resultCount {
		t.Error("未注册的图表不应被修改")
	}
}

// TestResolveSearchOrder 测试查找优先级（图 > 表 > 章节）
func TestResolveSearchOrder(t *testing.T) {
	r := NewResolver()

	// 使用相同 ID 注册不同类型的引用
	id := "item"

	// 虽然实际使用中这不太可能，但我们测试 Resolve 的搜索顺序

	// 情况1：只注册为图
	r.Reset()
	r.RegisterFigure(id, "这是一张图")
	ref, err := r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("应该找到图引用")
	}

	// 情况2：注册为表和图，应找到图（优先级更高）
	r.Reset()
	r.RegisterTable(id, "这是一张表")
	r.RegisterFigure(id, "这是一张图")
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("当有图和表时，应优先返回图")
	}

	// 情况3：注册为章节
	r.Reset()
	r.RegisterSection(id, "这是一章", 1)
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeSection {
		t.Error("应该找到章节引用")
	}

	// 情况4：三种都有（虽然不现实），应优先返回图
	r.Reset()
	r.RegisterSection(id, "章节", 1)
	r.RegisterTable(id, "表")
	r.RegisterFigure(id, "图")
	ref, err = r.Resolve(id)
	if err != nil || ref.Type != TypeFigure {
		t.Error("三种都有时应优先返回图")
	}
}

// TestProcessHTMLMultipleRefs 测试单个 HTML 字符串中处理多个不同类型的引用
func TestProcessHTMLMultipleRefs(t *testing.T) {
	r := NewResolver()

	// 注册不同类型的引用
	r.RegisterFigure("fig1", "架构图")
	r.RegisterFigure("fig2", "流程图")
	r.RegisterTable("tab1", "性能对比")
	r.RegisterTable("tab2", "功能列表")
	r.RegisterSection("intro", "简介", 1)
	r.RegisterSection("method", "方法", 2)

	// 包含多个混合类型引用的 HTML
	input := `
	<p>如 {{ref:fig1}} 所示，系统架构如下。根据 {{ref:tab1}}，性能指标如下。</p>
	<p>详见 {{ref:intro}} 和 {{ref:method}} 获取更多信息。</p>
	<p>{{ref:fig2}} 展示了流程，{{ref:tab2}} 列出了所有功能。</p>
	`

	result := r.ProcessHTML(input)

	// 验证所有占位符都被替换
	if strings.Contains(result, "{{ref:") {
		t.Error("所有引用占位符应被替换")
	}

	// 验证每种类型都被正确替换
	if !strings.Contains(result, "图1") || !strings.Contains(result, "图2") {
		t.Error("两个图的引用都应被处理")
	}

	if !strings.Contains(result, "表1") || !strings.Contains(result, "表2") {
		t.Error("两个表的引用都应被处理")
	}

	// 章节应显示为分层编号
	if !strings.Contains(result, "§1") || !strings.Contains(result, "§1.1") {
		t.Error("两个章节的引用都应被处理")
	}

	// 验证链接结构
	if !strings.Contains(result, `href="#fig1"`) || !strings.Contains(result, `href="#fig2"`) {
		t.Error("图的引用应包含正确的锚点")
	}

	if !strings.Contains(result, `href="#tab1"`) || !strings.Contains(result, `href="#tab2"`) {
		t.Error("表的引用应包含正确的锚点")
	}

	if !strings.Contains(result, `href="#intro"`) || !strings.Contains(result, `href="#method"`) {
		t.Error("章节的引用应包含正确的锚点")
	}

	// 验证 CSS 类都正确
	if !strings.Contains(result, `class="ref-figure"`) {
		t.Error("图的引用应包含 ref-figure 类")
	}

	if !strings.Contains(result, `class="ref-table"`) {
		t.Error("表的引用应包含 ref-table 类")
	}

	if !strings.Contains(result, `class="ref-section"`) {
		t.Error("章节的引用应包含 ref-section 类")
	}
}
