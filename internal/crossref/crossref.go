// Package crossref 提供交叉引用和自动编号功能。
// 支持对图表、表格和章节的编号和引用，可以在 HTML 中替换占位符为实际的编号。
package crossref

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/yeasy/mdpress/pkg/utils"
)

// Package-level compiled regexps to avoid recompilation per call.
var (
	refPlaceholderRegexp = regexp.MustCompile(`\{\{ref:([a-zA-Z0-9_\-]+)\}\}`)
	figureCaptionRegexp  = regexp.MustCompile(`<figure\s+id="([^"]+)"([^>]*)>(.*?)</figure>`)
	tableCaptionRegexp   = regexp.MustCompile(`<table\s+id="([^"]+)"([^>]*)>(.*?)</table>`)
)

// ReferenceType 定义引用类型的常量
type ReferenceType string

const (
	TypeFigure  ReferenceType = "figure"  // 图片
	TypeTable   ReferenceType = "table"   // 表格
	TypeSection ReferenceType = "section" // 章节
)

// Reference 表示一个被追踪的引用对象
type Reference struct {
	Type      ReferenceType // 引用类型（图、表或章节）
	ID        string        // 唯一标识符
	Number    int           // 自动分配的编号
	Title     string        // 标题或描述
	Level     int           // 对于章节，表示标题级别；其他类型默认为 0
	NumberStr string        // 分层编号字符串，如 "1.2.3"（仅章节类型使用）
}

// Resolver 管理所有的交叉引用和自动编号
type Resolver struct {
	mu            sync.RWMutex          // 并发访问锁
	figures       map[string]*Reference // 图片的ID到引用的映射
	tables        map[string]*Reference // 表格的ID到引用的映射
	sections      map[string]*Reference // 章节的ID到引用的映射
	figCount      int                   // 图片计数器
	tabCount      int                   // 表格计数器
	sectionCounts map[int]int           // 按级别的章节计数器
}

// NewResolver 创建一个新的交叉引用解析器实例
func NewResolver() *Resolver {
	return &Resolver{
		figures:       make(map[string]*Reference),
		tables:        make(map[string]*Reference),
		sections:      make(map[string]*Reference),
		sectionCounts: make(map[int]int),
	}
}

// RegisterFigure 注册一个图片并返回自动分配的编号
// 参数 id 是图片的唯一标识符（通常用于生成 HTML 锚点）
// 参数 title 是图片的标题或说明
// 返回值是自动分配的图片编号
func (r *Resolver) RegisterFigure(id, title string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if ref, exists := r.figures[id]; exists {
		return ref.Number
	}

	r.figCount++
	ref := &Reference{
		Type:   TypeFigure,
		ID:     id,
		Number: r.figCount,
		Title:  title,
		Level:  0,
	}

	r.figures[id] = ref
	return r.figCount
}

// RegisterTable 注册一个表格并返回自动分配的编号
// 参数 id 是表格的唯一标识符
// 参数 title 是表格的标题或说明
// 返回值是自动分配的表格编号
func (r *Resolver) RegisterTable(id, title string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if ref, exists := r.tables[id]; exists {
		return ref.Number
	}

	r.tabCount++
	ref := &Reference{
		Type:   TypeTable,
		ID:     id,
		Number: r.tabCount,
		Title:  title,
		Level:  0,
	}

	r.tables[id] = ref
	return r.tabCount
}

// RegisterSection 注册一个章节
// 参数 id 是章节的唯一标识符
// 参数 title 是章节标题
// 参数 level 是标题级别（1-6），用于生成分层编号
//
// 对于分层编号的示例：
// Level 1: 1. 2. 3. ...
// Level 2: 1.1. 1.2. 2.1. ...
// Level 3: 1.1.1. 1.1.2. ...
func (r *Resolver) RegisterSection(id, title string, level int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在
	if _, exists := r.sections[id]; exists {
		return
	}

	// 重置更深层级的计数器
	for lv := level + 1; lv <= 6; lv++ {
		delete(r.sectionCounts, lv)
	}

	// 增加当前级别的计数
	r.sectionCounts[level]++

	// 构建分层编号
	var numbers []string
	for lv := 1; lv <= level; lv++ {
		if count, ok := r.sectionCounts[lv]; ok {
			numbers = append(numbers, strconv.Itoa(count))
		} else {
			numbers = append(numbers, "0")
		}
	}

	// 生成编号字符串（如 "1.2.3"）用于显示
	numberStr := strings.Join(numbers, ".")

	ref := &Reference{
		Type:      TypeSection,
		ID:        id,
		Number:    r.sectionCounts[level],
		Title:     title,
		Level:     level,
		NumberStr: numberStr,
	}

	r.sections[id] = ref
}

// Resolve 根据 ID 查找引用信息
// 参数 id 是引用的唯一标识符
// 返回值是找到的 Reference 指针，如果未找到返回 error
func (r *Resolver) Resolve(id string) (*Reference, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 按优先级查找（图 > 表 > 章）
	if ref, ok := r.figures[id]; ok {
		return ref, nil
	}
	if ref, ok := r.tables[id]; ok {
		return ref, nil
	}
	if ref, ok := r.sections[id]; ok {
		return ref, nil
	}

	return nil, fmt.Errorf("reference not found: %s", id)
}

// ProcessHTML 处理 HTML 内容，替换 {{ref:id}} 占位符为实际的引用
// 支持的占位符格式：
// - {{ref:fig_1}} 替换为 "图1"
// - {{ref:table_1}} 替换为 "表1"
// - {{ref:section_intro}} 替换为 "1.2.3"（章节编号）
//
// 示例：
// 输入: "如图 {{ref:fig_demo}} 所示，..."
// 输出: "如图 图1 所示，..."
func (r *Resolver) ProcessHTML(html string) string {
	return refPlaceholderRegexp.ReplaceAllStringFunc(html, func(match string) string {
		// 提取 ID
		parts := refPlaceholderRegexp.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		id := parts[1]
		ref, err := r.Resolve(id)
		if err != nil {
			// 如果找不到引用，返回原占位符
			return match
		}

		// 根据类型生成引用文本
		switch ref.Type {
		case TypeFigure:
			return fmt.Sprintf(`<a href="#%s" class="ref-figure">图%d</a>`, utils.EscapeAttr(ref.ID), ref.Number)
		case TypeTable:
			return fmt.Sprintf(`<a href="#%s" class="ref-table">表%d</a>`, utils.EscapeAttr(ref.ID), ref.Number)
		case TypeSection:
			label := fmt.Sprintf("第%d节", ref.Number)
			if ref.NumberStr != "" {
				label = "§" + ref.NumberStr
			}
			return fmt.Sprintf(`<a href="#%s" class="ref-section">%s</a>`, utils.EscapeAttr(ref.ID), label)
		default:
			return match
		}
	})
}

// AddCaptions 为图表添加编号标题
// 处理形如 <figure id="fig_1"><img ...></figure> 的 HTML
// 为其添加 <figcaption>图1: 标题</figcaption>
//
// 示例：
// 输入: <figure id="fig_demo"><img src="demo.png"></figure>
// 输出: <figure id="fig_demo"><img src="demo.png"><figcaption>图1: 演示图</figcaption></figure>
func (r *Resolver) AddCaptions(html string) string {
	// Copy both maps in a single atomic snapshot
	r.mu.RLock()
	figuresCopy := make(map[string]*Reference, len(r.figures))
	for id, ref := range r.figures {
		figuresCopy[id] = ref
	}
	tablesCopy := make(map[string]*Reference, len(r.tables))
	for id, ref := range r.tables {
		tablesCopy[id] = ref
	}
	r.mu.RUnlock()

	// 为图片添加标题
	html = r.addFigureCaptions(html, figuresCopy)

	// 为表格添加标题
	html = r.addTableCaptions(html, tablesCopy)

	return html
}

// addFigureCaptions 为 figure 元素添加标题
func (r *Resolver) addFigureCaptions(html string, figures map[string]*Reference) string {
	return figureCaptionRegexp.ReplaceAllStringFunc(html, func(match string) string {
		parts := figureCaptionRegexp.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		id := parts[1]
		attrs := parts[2]
		content := parts[3]

		// 查找该 ID 对应的引用
		ref, ok := figures[id]
		if !ok {
			return match
		}

		// 检查是否已有 figcaption（避免重复）
		if strings.Contains(content, "<figcaption") {
			return match
		}

		// 构建新的 figure 元素，包含标题
		caption := fmt.Sprintf(`<figcaption>图%d: %s</figcaption>`,
			ref.Number, utils.EscapeHTML(ref.Title))

		return fmt.Sprintf(`<figure id="%s"%s>%s%s</figure>`,
			utils.EscapeAttr(id), attrs, content, caption)
	})
}

// addTableCaptions 为 table 元素添加标题
func (r *Resolver) addTableCaptions(html string, tables map[string]*Reference) string {
	return tableCaptionRegexp.ReplaceAllStringFunc(html, func(match string) string {
		parts := tableCaptionRegexp.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}

		id := parts[1]
		attrs := parts[2]
		content := parts[3]

		// 查找该 ID 对应的引用
		ref, ok := tables[id]
		if !ok {
			return match
		}

		// 检查是否已有 caption（避免重复）
		if strings.Contains(content, "<caption") {
			return match
		}

		// 构建新的 table 元素，在开头添加 caption
		caption := fmt.Sprintf(`<caption>表%d: %s</caption>`,
			ref.Number, utils.EscapeHTML(ref.Title))

		return fmt.Sprintf(`<table id="%s"%s>%s%s</table>`,
			utils.EscapeAttr(id), attrs, caption, content)
	})
}

// GetAllReferences 返回所有已注册的引用（用于调试或构建参考列表）
// 返回的映射包含所有类型的引用
func (r *Resolver) GetAllReferences() map[string]*Reference {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Reference)

	// 复制所有引用
	for id, ref := range r.figures {
		result[id] = ref
	}
	for id, ref := range r.tables {
		result[id] = ref
	}
	for id, ref := range r.sections {
		result[id] = ref
	}

	return result
}

// Reset 清空所有引用信息，重新初始化解析器
// 用于处理多个独立的文档时
func (r *Resolver) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.figures = make(map[string]*Reference)
	r.tables = make(map[string]*Reference)
	r.sections = make(map[string]*Reference)
	r.figCount = 0
	r.tabCount = 0
	r.sectionCounts = make(map[int]int)
}
