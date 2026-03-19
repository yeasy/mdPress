// Package plugin 定义 mdpress 的插件系统接口。
// 当前版本（v0.2）仅定义接口和 Hook 点，不提供具体实现。
// 这些接口为未来的插件系统奠定基础，确保核心代码在合适的位置预留了扩展点。
//
// 插件可以在构建流程的各个阶段介入，修改配置、处理内容或添加后处理逻辑。
// 插件通过 Hook 点与构建流程交互，每个 Hook 点对应构建流程中的一个特定阶段。
package plugin

import (
	"context"

	"github.com/yeasy/mdpress/internal/config"
)

// Phase 表示构建流程中的阶段，插件可以在这些阶段注册 Hook。
type Phase string

const (
	// PhaseConfigLoaded 配置加载完成后触发。
	// 插件可以修改配置、注入默认值、添加自定义配置项。
	PhaseConfigLoaded Phase = "config_loaded"

	// PhaseBeforeParse Markdown 解析前触发。
	// 插件可以预处理 Markdown 源码（如自定义语法扩展、文件包含等）。
	PhaseBeforeParse Phase = "before_parse"

	// PhaseAfterParse Markdown 解析后触发。
	// 插件可以修改解析后的 HTML（如自动链接检查、自定义元素注入等）。
	PhaseAfterParse Phase = "after_parse"

	// PhaseBeforeRender HTML 组装前触发。
	// 插件可以修改封面、目录、章节等渲染部件。
	PhaseBeforeRender Phase = "before_render"

	// PhaseAfterRender HTML 组装后触发。
	// 插件可以修改最终的 HTML 文档（如 SEO 优化、水印注入等）。
	PhaseAfterRender Phase = "after_render"

	// PhaseBeforeOutput 输出文件前触发。
	// 插件可以拦截或修改输出流程（如自定义输出路径、格式转换等）。
	PhaseBeforeOutput Phase = "before_output"

	// PhaseAfterOutput 输出文件后触发。
	// 插件可以执行后处理动作（如上传到 CDN、发送通知等）。
	PhaseAfterOutput Phase = "after_output"
)

// HookContext 传递给插件 Hook 函数的上下文信息。
// 包含当前构建阶段的所有可用数据，插件可以读取和修改这些数据。
type HookContext struct {
	// Context 标准 Go context，用于超时和取消控制
	Context context.Context

	// Config 当前的构建配置（可修改）
	Config *config.BookConfig

	// Phase 当前触发的阶段
	Phase Phase

	// Content 当前处理的内容（Markdown 或 HTML，取决于阶段）
	// BeforeParse 阶段为 Markdown 源码，AfterParse/AfterRender 阶段为 HTML
	Content string

	// ChapterIndex 当前处理的章节索引（仅 BeforeParse/AfterParse 阶段有效）
	// -1 表示非章节上下文
	ChapterIndex int

	// ChapterFile 当前处理的章节文件路径（仅 BeforeParse/AfterParse 阶段有效）
	ChapterFile string

	// OutputPath 输出文件路径（仅 BeforeOutput/AfterOutput 阶段有效）
	OutputPath string

	// OutputFormat 输出格式名称（仅 BeforeOutput/AfterOutput 阶段有效）
	OutputFormat string

	// Metadata 插件间共享的键值对元数据
	// 插件可以通过此字段传递数据给后续阶段或其他插件
	Metadata map[string]interface{}
}

// HookResult Hook 函数的返回结果。
// 插件通过返回 HookResult 来影响构建流程。
type HookResult struct {
	// Content 修改后的内容（如果为空，使用原始内容）
	Content string

	// Stop 如果为 true，跳过当前阶段后续的插件
	Stop bool
}

// Plugin 定义插件的核心接口。
// 每个插件必须实现此接口以参与 mdpress 的构建流程。
type Plugin interface {
	// Name 返回插件的唯一名称。
	// 名称应全部小写，使用连字符分隔（如 "link-checker", "seo-optimizer"）。
	Name() string

	// Version 返回插件版本号（语义化版本，如 "1.0.0"）。
	Version() string

	// Description 返回插件的简短描述。
	Description() string

	// Init 初始化插件。
	// 在构建开始时调用一次，插件可以在此进行配置加载、资源准备等初始化工作。
	Init(cfg *config.BookConfig) error

	// Hooks 返回插件关注的 Hook 点列表。
	// 构建流程只会在插件声明关注的阶段调用其 Execute 方法。
	Hooks() []Phase

	// Execute 在指定阶段执行插件逻辑。
	// 传入 HookContext 包含当前阶段的所有上下文信息。
	// 返回 HookResult 表示处理结果，error 表示执行失败。
	Execute(hookCtx *HookContext) (*HookResult, error)

	// Cleanup 清理插件资源。
	// 在构建完成后调用，无论构建是否成功。
	Cleanup() error
}

// Manager 插件管理器，负责插件的注册、排序和调度。
// v0.2 仅定义接口，不提供实现。
type Manager struct {
	plugins []Plugin
}

// NewManager 创建插件管理器。
func NewManager() *Manager {
	return &Manager{
		plugins: make([]Plugin, 0),
	}
}

// Register 注册一个插件。
// 插件按注册顺序执行。
func (m *Manager) Register(p Plugin) {
	m.plugins = append(m.plugins, p)
}

// InitAll 初始化所有已注册的插件。
func (m *Manager) InitAll(cfg *config.BookConfig) error {
	for _, p := range m.plugins {
		if err := p.Init(cfg); err != nil {
			return &PluginError{
				PluginName: p.Name(),
				Phase:      "",
				Err:        err,
			}
		}
	}
	return nil
}

// RunHook 在指定阶段运行所有关注该阶段的插件。
// 返回最终的 HookContext（可能被插件修改）。
func (m *Manager) RunHook(hookCtx *HookContext) error {
	for _, p := range m.plugins {
		if !m.pluginHandlesPhase(p, hookCtx.Phase) {
			continue
		}
		result, err := p.Execute(hookCtx)
		if err != nil {
			return &PluginError{
				PluginName: p.Name(),
				Phase:      string(hookCtx.Phase),
				Err:        err,
			}
		}
		if result != nil {
			if result.Content != "" {
				hookCtx.Content = result.Content
			}
			if result.Stop {
				break
			}
		}
	}
	return nil
}

// CleanupAll 清理所有插件资源。
// 即使某个插件的 Cleanup 失败，也会继续清理其他插件。
func (m *Manager) CleanupAll() error {
	var lastErr error
	for _, p := range m.plugins {
		if err := p.Cleanup(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Plugins 返回所有已注册的插件列表。
func (m *Manager) Plugins() []Plugin {
	return m.plugins
}

// pluginHandlesPhase 检查插件是否关注指定阶段
func (m *Manager) pluginHandlesPhase(p Plugin, phase Phase) bool {
	for _, h := range p.Hooks() {
		if h == phase {
			return true
		}
	}
	return false
}

// PluginError 插件执行错误，包含插件名称和阶段信息。
type PluginError struct {
	PluginName string
	Phase      string
	Err        error
}

// Error 实现 error 接口。
func (e *PluginError) Error() string {
	if e.Phase != "" {
		return "插件 " + e.PluginName + " 在阶段 " + e.Phase + " 执行失败: " + e.Err.Error()
	}
	return "插件 " + e.PluginName + " 初始化失败: " + e.Err.Error()
}

// Unwrap 支持 errors.Is / errors.As。
func (e *PluginError) Unwrap() error {
	return e.Err
}
