# 贡献指南

[English](CONTRIBUTING.md)

感谢你对 mdPress 的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告 Bug

如果你发现了 Bug，请在 [GitHub Issues](https://github.com/yeasy/mdpress/issues) 中提交，包含以下信息：

- 操作系统和版本
- Go 版本（`go version`）
- Chrome/Chromium 版本
- mdPress 版本（`mdpress --version`）
- 复现步骤（最好附带最小可复现的 `book.yaml` 和 Markdown 文件）
- 期望行为与实际行为
- 相关日志（使用 `--verbose` 获取详细日志）

### 功能建议

欢迎提交功能建议，请在 Issue 中描述：

- 你想解决的问题
- 建议的解决方案
- 可能的替代方案
- 该功能适合放在哪个版本（参见 [路线图](docs/ROADMAP_zh.md)）

### 提交代码

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feature/my-feature`
3. 安装 pre-commit hook：`make hooks`
4. 编写代码并添加测试
5. 提交更改：`git commit -m "feat: 添加新功能"`
   pre-commit hook 会自动运行 gofmt、go vet、golangci-lint、编译检查和快速测试。
6. 推送分支：`git push origin feature/my-feature`
7. 创建 Pull Request，描述你的变更内容和动机

## 开发环境搭建

### 前置要求

- Go 1.26 或更高版本
- Chrome 或 Chromium 浏览器（用于 PDF 生成相关的测试）
- GNU Make
- （可选）[golangci-lint](https://golangci-lint.run/) 用于代码检查

### 搭建步骤

```bash
# 1. 克隆仓库
git clone https://github.com/yeasy/mdpress.git
cd mdpress

# 2. 编译
make build

# 3. 运行测试
make test

# 4. 运行示例，验证构建结果
make example
```

### 常用开发命令

```bash
make build      # 编译二进制到 bin/mdpress
make test       # 运行所有测试（含竞态检测）
make check      # 格式化 + 静态检查 + 编译 + 快速测试（提交前检查）
make lint       # 代码静态检查（go vet + golangci-lint）
make fmt        # 格式化代码（gofmt）
make coverage   # 生成测试覆盖率报告（coverage.html）
make clean      # 清理构建产物
make example    # 使用 examples/ 构建示例 PDF
```

## 代码规范

### 代码风格

- 遵循 Go 标准代码风格，使用 `gofmt` 格式化
- 行宽不超过 120 字符（建议）
- 所有导出的函数、类型和方法必须有文档注释
- 注释统一使用英文，保持一致性

### 项目结构约定

| 目录 | 说明 | 注意事项 |
|------|------|----------|
| `cmd/` | CLI 命令定义（cobra） | 命令文件以功能命名，不含业务逻辑 |
| `internal/` | 内部包，不对外暴露 | 每个子包职责单一，通过接口解耦 |
| `pkg/utils/` | 通用工具函数 | 仅放置无业务依赖的纯工具函数 |
| `tests/` | 集成测试和端到端测试 | 单元测试放在对应包的 `_test.go` 中 |
| `themes/` | 主题 YAML 配置文件 | 新增主题需同时更新 `internal/theme/builtin.go` |
| `examples/` | 示例项目文件 | 用于文档展示和 `make example` |

### 错误处理

- 使用 `fmt.Errorf("xxx: %w", err)` 包装错误，保留调用链
- 尽量返回有意义的错误信息，帮助用户定位问题
- 避免使用 `panic`，除非是程序初始化阶段不可恢复的错误

### 日志

- 使用标准库 `log/slog` 进行日志输出
- `--verbose` 模式下输出 Debug 级别日志
- 正常模式下只输出 Info 及以上级别

## 提交信息规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

常用 type：

| type | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档变更 |
| `style` | 代码格式（不影响逻辑） |
| `refactor` | 重构 |
| `test` | 测试相关 |
| `chore` | 构建/工具链变更 |
| `perf` | 性能优化 |

示例：

```
feat(config): 添加 SUMMARY.md 自动发现支持

当 book.yaml 中未定义 chapters 时，自动在同目录下查找
SUMMARY.md 并解析章节结构。

Closes #42
```

## 测试要求

### 单元测试

- 新增功能必须有对应的单元测试
- 测试文件放在对应包目录下，以 `_test.go` 结尾
- 使用 Go 标准测试框架（`testing` 包）
- 测试函数命名：`TestXxx_描述`（如 `TestParseConfig_EmptyChapters`）

### 测试覆盖率

- 核心包（config、markdown、renderer、toc、crossref）目标覆盖率 ≥ 80%
- 新增代码的测试覆盖率不应低于现有水平
- 运行 `make coverage` 查看覆盖率报告

### 集成测试

- 涉及多个模块协作的功能，需要在 `tests/` 目录下添加集成测试
- 集成测试应使用 `tests/testdata/` 下的测试数据

### 端到端测试

- 涉及 CLI 命令和文件 I/O 的功能，需要添加端到端测试
- PDF 生成相关的测试可使用 build tag 标记（如 `//go:build e2e`），避免 CI 环境无 Chromium 时失败

### Golden 测试（快照测试）

Golden 测试是基于快照的回归测试，用于验证 HTML 生成输出是否符合预期。它们能够捕捉到对渲染输出的意外改变。

- **运行 Golden 测试**：`go test ./tests/golden/...`
- **更新 Golden 文件**：`go test ./tests/golden/... -update`
- **何时更新**：仅在对 HTML 输出进行了刻意变更（样式、结构、功能）后再更新 Golden 文件。Golden 文件存储在 `tests/golden/testdata/golden/` 中
- **工作原理**：测试将 Markdown 输入渲染为 HTML，对易变字段（如日期）进行规范化处理，然后与之前保存的 Golden 文件进行对比。首次运行时，Golden 文件会被创建，测试会跳过以便进行人工审查。

### 真实样本回归测试

- 涉及 GitBook 兼容、章节链接、目录解析、多语言等能力的改动，除了合成测试外，应至少用一个真实书籍样本做回归
- 建议优先使用 `docker_practice`（深层 SUMMARY.md、图片、章节互链）和 `learning_pickleball`（LANGS.md、双语目录行为）这两类样本
- 如果行为仍有边界或限制，请在 README 或 ROADMAP 中明确写出，不要只让测试或代码隐含表达

## PR 审查流程

1. 创建 PR 后，CI 会自动运行测试和 lint
2. 至少需要一位维护者的 Code Review 通过
3. 所有 CI 检查必须通过
4. PR 描述应包含：变更内容、动机、测试方法
5. 如有 Breaking Change，需在 PR 描述中明确说明

## 文档贡献

- 项目文档使用 Markdown 编写
- `README.md` 和 `README_zh.md` 分别维护英文与中文说明
- 新增功能需同步更新两个 README 文件中的功能列表和 CLI 命令说明
- 架构变更需同步更新 `docs/ARCHITECTURE.md` 和 `docs/ARCHITECTURE_zh.md`

## 许可证

向 mdPress 贡献代码，即表示你同意你的贡献将按 [MIT 许可证](LICENSE) 发布。
