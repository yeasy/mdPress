// Package markdown 提供 Markdown 解析和 HTML 转换功能。
// 基于 goldmark 库，支持 GFM 扩展、代码高亮、脚注等特性。
//
// 核心类型：
//   - Parser: Markdown 解析器，调用 Parse() 返回 HTML 和标题列表
//   - HeadingInfo: 标题信息（级别、文本、ID），用于目录生成
//
// 使用示例：
//
//	p := markdown.NewParser(markdown.WithCodeTheme("monokai"))
//	html, headings, err := p.Parse(source)
package markdown
