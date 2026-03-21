# mdPress 产品委员会讨论：下一步计划

> 日期：2026-03-20（更新版）
> 版本基准：v0.4.3（最新 git tag）
> 性质：内部参考文档，不纳入 git 提交

---

## 一、当前状态快照

### 版本与发布历史

```
v0.2.0  已发布  多格式输出（HTML/ePub/site）、GitHub 源、SUMMARY.md、live preview、doctor 命令
v0.3.0  已发布  ePub 输出、插件系统、KaTeX、Mermaid、GitBook 迁移工具
v0.3.1  已发布  CJK PDF 字体嵌入、TOC 深度控制、智能文件名、regexp 性能
v0.4.0  已发布  Typst 后端、并行构建、PDF 水印/边距/书签/页脚、PlantUML、构建缓存
v0.4.1  已发布  Bug 修复、测试覆盖扩展、CI 升级（Go 1.25）
v0.4.2  已发布  并行 check、修复慢测试 DNS 超时、gofmt 全量格式化
v0.4.3  已发布  docs 补全、glossary 性能优化、测试覆盖继续扩展
v0.5.0  ROADMAP.md 标记为已发布，但 git tag 不存在 ← ⚠️ 状态不一致
v1.0.0  计划中（目标 2027-Q1）
```

### 构建与测试健康度（截至 2026-03-20）

| 项目 | 状态 | 说明 |
|------|------|------|
| `go build ./...` | ✅ 成功 | 无编译错误 |
| `go test ./...` | ❌ **4 个包失败** | 较上次**退步**（上次全绿） |
| CI（main 分支） | ❌ **连续 5 次失败** | 包括最新 commit |
| Go 文件总数 | 144 个 | 代码规模稳步增长 |
| Stars / Forks | 3 / 0 | 社区影响力极低 |
| Open Issues | 0 | 尚无外部用户反馈 |

### 当前失败测试清单

| 包 | 失败测试 | 根因诊断 |
|----|---------|---------|
| `cmd` | `TestThemesCmd_SubcommandRegistration` | 测试检查 `Use == "show"`，但实际 Use 为 `"show <theme-name>"`——测试写法有误 |
| `cmd` | `TestExecuteThemesPreview/empty_output_path` | 传空字符串时函数尝试写入目录路径，预期行为不一致 |
| `internal/config` | `TestDiscoverWithBookJSON` | SUMMARY.md 章节发现返回 0，预期 1——实现或测试 fixture 有问题 |
| `internal/plugin` | `TestLoadPlugins_ResolvePath` | 测试期望相对路径解析到不存在的文件 `plugins/myplugin`，fixture 未创建 |
| `internal/plugin` | `TestLoadPlugins_InitFailure` | 同上，依赖环境 fixture 未准备 |
| `tests/golden` | `TestGoldenHTML` | HTML 渲染器已改动（新增三栏布局等），但 golden 文件未更新 |

**结论：最近连续 10 个"增加测试覆盖"的 commit 引入了失败的测试，导致 CI 全红。这是本次讨论最紧迫的问题。**

### HTML 渲染器现状

`internal/renderer/html_standalone.go` 已经实现了相当完善的单页 HTML：
- GitBook 风格三栏布局（全局侧边栏 / 内容区 / 页内 TOC）
- 深色 / 浅色 / 系统主题切换
- 全文搜索（⌘K）
- 代码块复制按钮 + 语言标签
- Callout 框
- 上一章 / 下一章导航
- Mermaid 图表
- KaTeX 数学公式
- 图片灯箱

### PDF CJK 字体现状

v0.3.1 通过注入 `@font-face`（`file://` URL）解决了 Chromium 后端的 CJK 字体嵌入问题。
Typst 后端理论上原生支持 CJK，但实际测试覆盖率低（`internal/pdf` 55.2%，`internal/typst` 76.5%）。

---

## 二、四角色讨论

---

### 产品经理（PM）

#### 最大优势

mdPress 功能已经相当完整：多格式（PDF/HTML/ePub/site）、双后端（Chromium/Typst）、插件系统、PlantUML、KaTeX、Mermaid、主题系统、多语言支持、GitBook 迁移……这些功能组合对于"写书/写技术文档"场景覆盖得很好。Go 语言实现意味着单二进制、跨平台、安装简单——这是对标 GitBook/mdBook 的核心差异点。

#### 最大短板

**当前最大问题是：CI 连续红了 5 次，而项目的 Stars 只有 3。** 这两件事密切相关：潜在用户看到 CI 红叉会立刻关掉页面。我们花了很多精力"增加测试覆盖"，但反而让 CI 变得更差——这是典型的执行策略失误。

用户反馈：**目前 Issues 为零**，说明社区规模太小，几乎没有真实用户。我们需要先让项目被发现，才能收到用户反馈。

#### PM 的 Top 3

1. **立即修复测试/CI 红色状态**（P0，今天就做）：连续 5 次 CI 失败是项目可信度的最大威胁
2. **README 增加 GIF 演示 + 一键安装命令**（P0）：让第一眼看到项目的人能在 30 秒内理解"这是什么、能做什么、怎么安装"
3. **HTML 输出质量提升为主打卖点**（P1）：现有 HTML 渲染器功能丰富，但 visual polish 还不够，需要能截图展示的"惊艳感"

---

### 架构师（Arch）

#### 最大优势

双后端架构（Chromium/Typst）是一个正确的长期决策：Chromium 确保最佳兼容性，Typst 提供零依赖路径。插件系统生命周期设计合理。渲染管道（source → config → markdown → renderer → output）职责清晰，扩展点明确。

#### 最大短板

**测试体系与架构意图脱节**：当前测试失败的根因不是"没测试"，而是"测试写了但是写错了"——Use 字段匹配逻辑错误、fixture 文件未创建、golden 文件未同步更新。这暴露出开发流程问题：没有"先跑测试再提交"的习惯（pre-commit hook 或本地测试强制要求）。

**ROADMAP.md 与 git tag 状态不一致**：ROADMAP 显示 v0.5.0 已发布，但 git 中最新 tag 是 v0.4.3，这意味着 ROADMAP 是"理想状态"而非"现实状态"——这会误导自己和贡献者。

#### HTML 输出优雅度：架构视角

当前 `html_standalone.go` 已将所有 CSS/JS 内联进单个文件，结构合理。要达到"优雅"需要解决：
1. **CSS 变量体系**：所有颜色/字体通过 CSS Custom Properties 统一管理，主题切换不需要 class 替换
2. **响应式断点**：移动端 < 768px 时侧边栏收起为抽屉式
3. **打印样式**：`@media print` 隐藏导航，保持排版质量
4. **无障碍访问**：`role`、`aria-*` 属性，键盘导航

#### PDF CJK 最终方案：架构视角

当前 `@font-face file://` 方案有环境依赖（需要系统安装 CJK 字体）。更彻底的方案：
- **嵌入字体子集**：构建时扫描所有 CJK 字符，用 Python `fonttools` 或 Go `sfnt` 提取最小字体子集，base64 嵌入 HTML
- **Typst 路径**：Typst 原生支持 CJK，可将"无 CJK 系统字体"场景推给 Typst 后端

#### Arch 的 Top 3

1. **修复所有失败测试**（P0）：这是架构健康的基础，且大部分修复是 1-5 行的测试代码修正
2. **引入 pre-commit 测试强制**：`make pre-commit` 脚本或 git hook，阻止 CI 红色 commit 进入 main
3. **ROADMAP/版本状态同步**：建立"版本状态"管理规范，ROADMAP 只记录已发布的 tag，避免现实与文档的分裂

---

### 程序员（Dev）

#### 最大优势

代码质量在持续改善：`errors.Join`、gofmt 规范、regexp 预编译、并发优化——这些都是专业 Go 代码该有的样子。`html_standalone.go` 已经相当现代，实现了搜索、暗色模式、MathJax/KaTeX，这不是一个简单工具该有的功能密度。

#### 最大短板

**测试是假绿灯**：最近 10 个 commit 全部是"增加测试"，但是 CI 全红——说明测试写完没有本地跑一次就 push 了。这比没有测试更危险：给人一种"有保护"的错觉。

具体需要修复的问题：

1. `TestThemesCmd_SubcommandRegistration`（`cmd/themes_test.go:36`）：
   ```go
   // 错误写法：
   if cmd.Use == sc.cmd {  // sc.cmd = "show"，但 Use = "show <theme-name>"
   // 修复：
   if strings.HasPrefix(cmd.Use, sc.cmd) {
   ```

2. `TestExecuteThemesPreview/empty_output_path`（`cmd/themes_test.go`）：
   函数在空路径时应该回退到默认路径，或者测试期望应该改为"预期报错"

3. `TestLoadPlugins_ResolvePath`（`internal/plugin/loader_test.go:248`）：
   测试依赖 `plugins/myplugin` 文件存在，需要在测试中创建临时 fixture 或 mock

4. `TestGoldenHTML`（`tests/golden/golden_test.go`）：
   运行 `go test ./tests/golden/... -update` 更新 golden 文件即可

5. `TestDiscoverWithBookJSON`（`internal/config/discover_test.go:75`）：
   需要检查 SUMMARY.md fixture 内容是否正确，或 discovery 逻辑是否有 bug

#### HTML 输出优雅度：Dev 视角

`html_standalone.go` 功能已经很全，差的是**细节打磨**：
- 字体：使用 Google Fonts（Inter/Source Code Pro）或嵌入 WOFF2 子集
- 代码高亮：当前用 Chroma，需确认暗色主题下高亮 token 颜色方案
- 过渡动画：侧边栏展开/收起应有 CSS transition
- 首屏加载：HTML 内联 CSS 可能很大，考虑 `<style>` critical CSS + 懒加载

#### PDF CJK 最终方案：Dev 视角

当前 `internal/pdf/pdf.go` 中的字体注入逻辑依赖系统路径。更稳健的实现：
```go
// 扩展候选路径（已有基础，继续完善）
var cjkFontCandidates = []string{
    // macOS
    "/System/Library/Fonts/STHeiti Medium.ttc",
    "/Library/Fonts/Songti.ttc",
    // Linux（Docker 环境）
    "/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
    "/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc",
    // Windows
    "C:/Windows/Fonts/msyh.ttc",
}
```
终极方案：`make fonts` 步骤在 CI/release 构建时将 Noto CJK 字体嵌入为 Go embed 资源。

#### Dev 的 Top 3

1. **修复 6 个失败测试**（P0，1-2 小时工作量，全是测试代码改动，不动产品代码）
2. **HTML 输出：增加字体 + 过渡动画 + 打印 CSS**（P1，能极大提升演示效果）
3. **Makefile 增加 `make test-local` 强制前置检查**，防止 push 破 CI

---

### 测试（QA）

#### 最大优势

测试基础设施已经搭建得不错：golden test 框架、integration test、单元测试都有。`internal/cover`（100%）、`internal/toc`（98.3%）、`internal/theme`（95.1%）、`internal/crossref`（94.4%）这些核心包已经非常健康。

#### 最大短板

**"数量"和"质量"的混淆**：最近的提交专注于增加覆盖率数字，但写出了会失败的测试——这说明测试没有被实际执行验证。高覆盖率但 CI 红色，比低覆盖率但 CI 绿色更糟糕，因为它消耗了开发时间却没有带来保障。

**现有失败的根因分析**：
- **测试逻辑错误**（`themes_test`）：Use 字段包含参数占位符，测试写法太严格
- **缺失 fixture**（`loader_test`）：需要创建临时测试文件
- **golden 文件过期**（`golden_test`）：产品代码更新后没有同步更新 golden
- **实现 bug**（`config/discover_test`）：SUMMARY.md 发现逻辑可能有实际 bug，需要深查

#### HTML 输出测试：QA 视角

当前 `TestGoldenHTML` 是最好的 HTML 质量保障机制。但 golden 文件目前是 845 行的完整 HTML——这太脆弱了，任何 CSS 空格变化都会导致测试失败。建议：
- **结构性断言**：用 `golang.org/x/net/html` 解析，断言 DOM 结构（标题数量、章节 ID、代码块语言）
- **快照分离**：样式变更用 `--update` 重新生成；内容结构用断言保护

#### QA 的 Top 3

1. **立刻修复所有失败测试，CI 回绿**（P0）：没有绿色 CI 的覆盖率数字没有意义
2. **引入 HTML 结构性断言**取代脆弱的全文 golden 比对（P1）
3. **建立测试规范文档**：每个测试必须本地运行通过才能 push，PR checklist 中加入"测试在本地绿色"

---

## 三、关键议题讨论

### Q1：项目当前最大优势

**共识**：功能密度超出了项目知名度。一个单二进制、支持 PDF/HTML/ePub、有主题系统、有插件架构、有双 PDF 后端的 Go 工具，在同类中是相当稀缺的组合。这是真正的竞争力，但还没有被市场感知。

---

### Q2：最大短板 + 用户反馈

**共识**：
- **CI 连续红色**是最大的可见性问题（外部用户会看到）
- **Stars 3 / Issues 0** 说明还没有真实用户基础——反馈不是"差"，而是"静默"
- 无法改进没有的用户反馈；当前阶段的优先任务是让项目被发现

**分歧**：
- PM 认为应该先做市场推广
- Dev/QA 认为必须先把 CI 修绿，否则推广是在推损害口碑

**结论**：Dev/QA 正确——先绿后推。

---

### Q3：v0.5.0 应该包含什么

**PM 立场**：v0.5.0 要有用户能感知的功能，PlantUML 本地渲染对企业用户有价值，HTML 输出优化对所有用户可见。

**Arch 立场**：v0.5.0 必须先清理技术债务（CI 绿色、测试修复），才能有信心做新功能。

**Dev 立场**：HTML 输出打磨是性价比最高的投入——改 CSS 成本低、视觉效果显著、可以截图宣传。

**QA 立场**：没有绿色 CI 的 v0.5.0 不应该发布。测试质量规范必须先建立。

**最终共识**（见第四节）

---

### Q4：如何提升项目影响力

**PM 提议（完整清单）**：

1. **README 改造**：
   - 加入 demo GIF（`mdpress build` 的 30 秒录屏）
   - 加入 badge 区（build status / go version / license / stars）
   - 加入"vs GitBook / mdBook"对比表
   - 中文 README 单独维护（`README_zh.md` 已有，确保同步）

2. **内容营销**：
   - 用 mdPress 本身构建一本"mdPress 用户手册"，输出 HTML 在线展示
   - 提交到 awesome-go、awesome-markdown 等列表
   - 在 V2EX、少数派、掘金发布中文介绍文章

3. **社区建设**：
   - 设置 GitHub Discussions 作为社区问答入口
   - 写 CONTRIBUTING.md 中文版（已有，检查质量）
   - 设计 "good first issue" 标签的 issue 引导新贡献者

4. **CI 绿色是前提**：以上都无效，如果 CI 是红的

---

### Q5：HTML 输出如何做到优雅

**现状评估**：`html_standalone.go` 已实现三栏布局、暗色模式、全文搜索、代码复制、Mermaid、KaTeX，功能层面已经优雅。问题在**视觉打磨**。

**Arch + Dev 共识的改进方向**：

#### 字体
```css
/* 优先使用系统字体栈，CJK 专门处理 */
body {
  font-family:
    -apple-system, BlinkMacSystemFont,
    "Segoe UI", "Helvetica Neue",
    "PingFang SC", "Microsoft YaHei",
    "Noto Sans CJK SC",
    sans-serif;
}
code, pre {
  font-family:
    "JetBrains Mono", "Fira Code",
    "Source Code Pro", "Cascadia Code",
    "Noto Sans Mono CJK SC",
    monospace;
}
```

#### 交互动画
```css
/* 侧边栏平滑过渡 */
.sidebar { transition: transform 0.25s ease; }
/* 链接悬停 */
a { transition: color 0.15s ease; }
/* 代码块复制按钮 */
.copy-btn { transition: opacity 0.2s ease, background 0.15s ease; }
```

#### 打印优化
```css
@media print {
  .sidebar, .toc-right, .search-overlay { display: none !important; }
  .content { max-width: 100%; margin: 0; padding: 0; }
  a[href]:after { content: " (" attr(href) ")"; font-size: 0.8em; }
}
```

#### 响应式移动端
```css
@media (max-width: 768px) {
  .layout { grid-template-columns: 1fr; }
  .sidebar { position: fixed; transform: translateX(-100%); }
  .sidebar.open { transform: translateX(0); }
  .toc-right { display: none; }
}
```

**PM 要求**：这些改动实施后，必须有**截图**加入 README，作为视觉展示材料。

---

### Q6：PDF 中文字体问题的最终解决方案

**历史**：v0.3.1 通过 `@font-face file://` 解决了基础问题，但依赖系统字体，Docker/CI 环境中可能无 CJK 字体。

**四角色共识的最终方案**：**三层降级策略**

```
第一层：嵌入式 Noto CJK 字体（go:embed）
  → 在 release 构建时 embed Noto Sans CJK SC 子集（WOFF2，约 3-5 MB）
  → 优先使用，完全不依赖系统字体
  → 实现：scripts/embed-fonts.sh 生成字体子集 → internal/assets/fonts/

第二层：系统字体自动发现（现有逻辑扩展）
  → 扩展 CJK 候选路径（macOS / Linux / Windows / Docker）
  → 覆盖 Noto CJK 在 Ubuntu/Debian 的标准路径
  → 覆盖 Windows 宋体/微软雅黑路径

第三层：Typst 后端回退
  → 当 Chromium 路径无 CJK 字体时，--backend typst 原生支持 CJK
  → doctor 命令给出明确提示和切换建议
```

**实施优先级**：
- 第二层（扩展候选路径）：**本周可做**，成本极低
- 第三层（doctor 提示）：**v0.5.0 必须有**，用户体验保障
- 第一层（embed 字体）：**v0.6.0**，涉及 release 构建流程改造

---

## 四、优先级排序与共识

### 综合优先级矩阵

| 任务 | 用户价值 | 实现成本 | 紧迫性 | 优先级 |
|------|---------|---------|--------|--------|
| 修复 6 个失败测试，CI 回绿 | 高（可信度） | 极低（1-2h） | 🔴 P0 | **立刻做** |
| 修复 ROADMAP 版本状态不一致 | 中 | 极低（5min） | 🟠 P1 | 同上 |
| HTML 输出视觉打磨（字体/动画/响应式/打印） | 高（演示效果） | 低（CSS 改动） | 🟠 P1 | v0.5.0 |
| README GIF 演示 + badge + 对比表 | 高（发现率） | 低 | 🟠 P1 | v0.5.0 |
| PDF CJK 第二层（扩展候选路径） | 高（企业用户） | 极低 | 🟠 P1 | v0.5.0 |
| doctor 命令字体提示（CJK 第三层） | 高（用户体验） | 低 | 🟠 P1 | v0.5.0 |
| PlantUML 本地渲染 | 中（企业离线） | 中（3-5天） | 🟡 P2 | v0.5.0 |
| HTML 结构性断言（替代全文 golden） | 中（测试健壮性） | 中 | 🟡 P2 | v0.5.0 |
| pre-commit 测试强制脚本 | 中（防止 CI 红） | 低 | 🟡 P2 | v0.5.0 |
| 内容营销（掘金/V2EX/awesome-go） | 高（Stars） | 低（时间投入） | 🟡 P2 | 持续 |
| PDF CJK 第一层（embed 字体） | 高（零依赖） | 高（release 改造） | 🟢 P3 | v0.6.0 |
| 双后端 golden test 框架 | 中（回归保护） | 高 | 🟢 P3 | v0.6.0 |
| 覆盖率冲刺至 90% | 高（v1.0 目标） | 高 | 🟢 P3 | v1.0.0 |

---

## 五、v0.5.0 具体计划

**主题**：CI 绿色 + HTML 优雅 + CJK 字体最终方案（第二/三层）+ PlantUML 本地化

**验收标准**：
- `go test ./...` 全部通过，CI main 分支绿色
- HTML 输出有截图在 README 展示
- `mdpress doctor` 对 CJK 字体缺失给出明确提示和 Typst 切换建议
- ROADMAP.md 与 git tag 状态保持一致

### 功能清单

#### 🔴 Must Have（阻塞发布）

- [ ] 修复 `TestThemesCmd_SubcommandRegistration`：`Use` 字段比对改为 `strings.HasPrefix`
- [ ] 修复 `TestExecuteThemesPreview/empty_output_path`：明确空路径的预期行为并对齐测试
- [ ] 修复 `TestLoadPlugins_ResolvePath` / `TestLoadPlugins_InitFailure`：补充 fixture 或改用 mock
- [ ] 更新 `TestGoldenHTML` golden 文件（`go test ./tests/golden/... -update`）
- [ ] 排查 `TestDiscoverWithBookJSON`：是实现 bug 还是 fixture 问题，并修复
- [ ] 确认 CI 在 main 分支连续绿色

#### 🟠 Should Have（v0.5.0 核心特性）

- [ ] HTML 输出视觉打磨：
  - [ ] CJK 系统字体栈
  - [ ] 代码字体（JetBrains Mono / Fira Code 优先）
  - [ ] 侧边栏/按钮过渡动画（CSS transition）
  - [ ] 响应式移动端（< 768px 抽屉式侧边栏）
  - [ ] `@media print` 打印样式
- [ ] PDF CJK 字体候选路径扩展（Noto/Ubuntu/Windows 完整列表）
- [ ] `mdpress doctor` 增加 CJK 字体诊断，无字体时提示 `--backend typst`
- [ ] PlantUML 本地渲染（`renderLocal()` 实现，支持 CLI / JAR 两种模式）
- [ ] README 改造：演示截图 / badge / 对比表

#### 🟡 Nice to Have

- [ ] `Makefile` 增加 `make pre-commit`（本地跑完整测试套件）
- [ ] HTML 结构性断言替代部分 golden 全文比对
- [ ] GitHub Discussions 启用

---

## 六、各角色分歧与最终决定

| 议题 | PM | Arch | Dev | QA | 最终决定 |
|------|----|----|-----|-----|---------|
| v0.5.0 先修 CI 还是先做功能 | 先做功能（有截止） | 先修 CI（架构健康） | 先修 CI（1-2h 能搞定） | 先修 CI（必须） | **先修 CI，当天完成，不影响功能开发** |
| HTML golden 测试策略 | 无意见 | 结构性断言 | 更新 golden 文件即可 | 两者都要 | **v0.5.0 先更新 golden 文件，v0.6.0 迁移到结构性断言** |
| PDF CJK 第一层（embed 字体）是否进 v0.5.0 | 想要（零依赖卖点） | 太重（release 流程） | 评估后再说 | 不影响测试 | **推迟到 v0.6.0，v0.5.0 做第二/三层** |
| PlantUML 本地渲染 vs HTML 优化哪个先 | HTML 优化（可见） | 两者都要 | HTML 优化（性价比高） | 无意见 | **并行推进；HTML 优化优先（可单独上线）** |
| 版本号：下一个是 v0.4.4 还是 v0.5.0 | v0.5.0（功能版本） | v0.4.4（补丁先行） | v0.5.0（一步到位） | v0.4.4 先修复 | **先发 v0.4.4 修复 CI/测试，再发 v0.5.0 功能** |

---

## 七、行动项（当前 Sprint）

| # | 任务 | 负责 | 目标版本 | 预估工时 |
|---|------|------|---------|---------|
| 1 | 修复 `themes_test` Use 字段比对 | Dev | v0.4.4 | 30min |
| 2 | 修复 `loader_test` fixture 问题 | Dev | v0.4.4 | 1h |
| 3 | 排查 `config/discover_test` 失败根因 | Dev+Arch | v0.4.4 | 1-2h |
| 4 | 更新 golden HTML 文件 | Dev | v0.4.4 | 15min |
| 5 | 确认 CI 绿色，发布 v0.4.4 tag | Dev | v0.4.4 | — |
| 6 | HTML 输出 CSS 打磨 | Dev | v0.5.0 | 1-2天 |
| 7 | CJK 字体候选路径扩展 | Dev | v0.5.0 | 2h |
| 8 | doctor 命令字体诊断提示 | Dev | v0.5.0 | 3h |
| 9 | PlantUML `renderLocal()` 实现 | Dev | v0.5.0 | 3-5天 |
| 10 | README 改造（截图/badge/对比表） | PM+Dev | v0.5.0 | 1天 |
| 11 | 内容营销（掘金/V2EX 文章） | PM | 持续 | — |

---

*本文档为产品委员会内部讨论记录，不纳入 git 版本管理，仅供本地参考。*
*上次更新：2026-03-19 → 本次更新：2026-03-20*
