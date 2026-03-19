package markdown

import (
	"fmt"
	"os"
	"sync"
)

// ProcessingResult 文档处理结果
type ProcessingResult struct {
	FilePath string
	HTML     string
	Headings []HeadingInfo
	Error    error
}

// DocumentProcessor 批量文档处理器，支持并发
type DocumentProcessor struct {
	parser         *Parser
	maxConcurrency int
	cache          map[string]*ProcessingResult
	cacheMu        sync.RWMutex
}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor(maxConcurrency int) *DocumentProcessor {
	return &DocumentProcessor{
		parser:         NewParser(),
		maxConcurrency: maxConcurrency,
		cache:          make(map[string]*ProcessingResult),
	}
}

// ProcessFile 处理单个 Markdown 文件
func (dp *DocumentProcessor) ProcessFile(filePath string) *ProcessingResult {
	dp.cacheMu.RLock()
	if result, exists := dp.cache[filePath]; exists {
		dp.cacheMu.RUnlock()
		return result
	}
	dp.cacheMu.RUnlock()

	content, err := os.ReadFile(filePath)
	if err != nil {
		return &ProcessingResult{
			FilePath: filePath,
			Error:    fmt.Errorf("读取文件失败: %w", err),
		}
	}

	html, headings, err := dp.parser.Parse(content)
	if err != nil {
		return &ProcessingResult{
			FilePath: filePath,
			Error:    fmt.Errorf("解析失败: %w", err),
		}
	}

	result := &ProcessingResult{
		FilePath: filePath,
		HTML:     html,
		Headings: headings,
	}

	dp.cacheMu.Lock()
	dp.cache[filePath] = result
	dp.cacheMu.Unlock()

	return result
}

// ProcessFiles 并发处理多个文件
func (dp *DocumentProcessor) ProcessFiles(filePaths []string) []*ProcessingResult {
	results := make([]*ProcessingResult, len(filePaths))

	if dp.maxConcurrency <= 0 {
		var wg sync.WaitGroup
		for i, path := range filePaths {
			wg.Add(1)
			go func(idx int, filePath string) {
				defer wg.Done()
				results[idx] = dp.ProcessFile(filePath)
			}(i, path)
		}
		wg.Wait()
	} else {
		semaphore := make(chan struct{}, dp.maxConcurrency)
		var wg sync.WaitGroup
		for i, path := range filePaths {
			wg.Add(1)
			go func(idx int, filePath string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				results[idx] = dp.ProcessFile(filePath)
			}(i, path)
		}
		wg.Wait()
	}

	return results
}

// ClearCache 清空缓存
func (dp *DocumentProcessor) ClearCache() {
	dp.cacheMu.Lock()
	defer dp.cacheMu.Unlock()
	dp.cache = make(map[string]*ProcessingResult)
}
