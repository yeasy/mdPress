package pdf

import (
	"testing"
	"time"
)

// TestNewGenerator 测试创建生成器
func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Fatal("NewGenerator 返回 nil")
	}
	// 验证默认值
	if g.timeout != defaultTimeout {
		t.Errorf("默认超时错误: got %v, want %v", g.timeout, defaultTimeout)
	}
	if g.pageWidth != defaultPageWidth {
		t.Errorf("默认页宽错误: got %f, want %f", g.pageWidth, defaultPageWidth)
	}
	if g.pageHeight != defaultPageHeight {
		t.Errorf("默认页高错误: got %f, want %f", g.pageHeight, defaultPageHeight)
	}
	if !g.printBackground {
		t.Error("默认应打印背景")
	}
}

// TestNewGeneratorWithOptions 测试带选项的创建
func TestNewGeneratorWithOptions(t *testing.T) {
	g := NewGenerator(
		WithTimeout(30*time.Second),
		WithPageSize(148, 210),
		WithMargins(10, 10, 15, 15),
		WithPrintBackground(false),
		WithHeaderFooter(true),
	)

	if g.timeout != 30*time.Second {
		t.Errorf("超时设置错误: got %v", g.timeout)
	}
	if g.pageWidth != 148 {
		t.Errorf("页宽设置错误: got %f", g.pageWidth)
	}
	if g.pageHeight != 210 {
		t.Errorf("页高设置错误: got %f", g.pageHeight)
	}
	if g.marginLeft != 10 {
		t.Errorf("左边距设置错误: got %f", g.marginLeft)
	}
	if g.marginTop != 15 {
		t.Errorf("上边距设置错误: got %f", g.marginTop)
	}
	if g.printBackground {
		t.Error("printBackground 应为 false")
	}
	if !g.displayHeaderFooter {
		t.Error("displayHeaderFooter 应为 true")
	}
}

// TestGenerateEmptyContent 测试空内容
func TestGenerateEmptyContent(t *testing.T) {
	g := NewGenerator()
	err := g.Generate("", "output.pdf")
	if err == nil {
		t.Error("空 HTML 内容应返回错误")
	}
}

// TestGenerateEmptyOutput 测试空输出路径
func TestGenerateEmptyOutput(t *testing.T) {
	g := NewGenerator()
	err := g.Generate("<html></html>", "")
	if err == nil {
		t.Error("空输出路径应返回错误")
	}
}

// TestGenerateFromNonExistentFile 测试不存在的 HTML 文件
func TestGenerateFromNonExistentFile(t *testing.T) {
	g := NewGenerator()
	err := g.GenerateFromFile("/nonexistent/file.html", "output.pdf")
	if err == nil {
		t.Error("不存在的文件应返回错误")
	}
}

// TestWithTimeoutOption 测试超时选项
func TestWithTimeoutOption(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{"10秒", 10 * time.Second},
		{"1分钟", time.Minute},
		{"5分钟", 5 * time.Minute},
	}

	for _, tt := range tests {
		g := NewGenerator(WithTimeout(tt.timeout))
		if g.timeout != tt.timeout {
			t.Errorf("%s: 超时设置错误: got %v, want %v", tt.name, g.timeout, tt.timeout)
		}
	}
}

// TestWithPageSizeOption 测试页面尺寸选项
func TestWithPageSizeOption(t *testing.T) {
	tests := []struct {
		name          string
		width, height float64
	}{
		{"A4", 210, 297},
		{"A5", 148, 210},
		{"Letter", 216, 279},
		{"B5", 176, 250},
	}

	for _, tt := range tests {
		g := NewGenerator(WithPageSize(tt.width, tt.height))
		if g.pageWidth != tt.width {
			t.Errorf("%s: 页宽错误: got %f, want %f", tt.name, g.pageWidth, tt.width)
		}
		if g.pageHeight != tt.height {
			t.Errorf("%s: 页高错误: got %f, want %f", tt.name, g.pageHeight, tt.height)
		}
	}
}

// TestWithMarginsOption 测试边距选项
func TestWithMarginsOption(t *testing.T) {
	g := NewGenerator(WithMargins(5, 10, 15, 20))
	if g.marginLeft != 5 {
		t.Errorf("左边距错误: got %f", g.marginLeft)
	}
	if g.marginRight != 10 {
		t.Errorf("右边距错误: got %f", g.marginRight)
	}
	if g.marginTop != 15 {
		t.Errorf("上边距错误: got %f", g.marginTop)
	}
	if g.marginBottom != 20 {
		t.Errorf("下边距错误: got %f", g.marginBottom)
	}
}

// TestChromiumCheck 测试 Chromium 检查
// 注意：此测试在 CI 环境中可能失败（无 Chrome）
func TestChromiumCheck(t *testing.T) {
	g := NewGenerator()
	err := g.checkChromiumAvailable()
	// 不断言具体结果，因为取决于环境
	_ = err
}

// TestMultipleOptionsChaining tests chaining multiple options together.
func TestMultipleOptionsChaining(t *testing.T) {
	g := NewGenerator(
		WithTimeout(90*time.Second),
		WithPageSize(210, 297),
		WithMargins(20, 20, 25, 25),
		WithPrintBackground(true),
		WithHeaderFooter(false),
	)

	if g.timeout != 90*time.Second {
		t.Error("chained options: timeout wrong")
	}
	if g.pageWidth != 210 || g.pageHeight != 297 {
		t.Error("chained options: page size wrong")
	}
	if g.marginLeft != 20 || g.marginTop != 25 {
		t.Error("chained options: margin wrong")
	}
	if !g.printBackground {
		t.Error("chained options: printBackground should be true")
	}
	if g.displayHeaderFooter {
		t.Error("chained options: displayHeaderFooter should be false")
	}
}

// TestDocumentOutlineEnabledByDefault verifies outline is on by default.
func TestDocumentOutlineEnabledByDefault(t *testing.T) {
	g := NewGenerator()
	if !g.generateDocumentOutline {
		t.Error("generateDocumentOutline should be true by default")
	}
	if !g.generateTaggedPDF {
		t.Error("generateTaggedPDF should be true by default")
	}
}

// TestWithDocumentOutlineOption tests toggling the document outline.
func TestWithDocumentOutlineOption(t *testing.T) {
	g := NewGenerator(WithDocumentOutline(false))
	if g.generateDocumentOutline {
		t.Error("generateDocumentOutline should be false after WithDocumentOutline(false)")
	}

	g2 := NewGenerator(WithDocumentOutline(true))
	if !g2.generateDocumentOutline {
		t.Error("generateDocumentOutline should be true after WithDocumentOutline(true)")
	}
}

// TestWithTaggedPDFOption tests toggling tagged PDF generation.
func TestWithTaggedPDFOption(t *testing.T) {
	g := NewGenerator(WithTaggedPDF(false))
	if g.generateTaggedPDF {
		t.Error("generateTaggedPDF should be false after WithTaggedPDF(false)")
	}
}
