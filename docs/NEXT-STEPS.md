# mdPress 产品委员会讨论：下一步计划

> 日期：2026-03-19
> 版本基准：v0.4.3（最新发布）
> 性质：内部参考文档，不纳入 git 提交

---

## 一、当前状态快照

### 版本历程

```
v0.2.0  已发布  多格式输出（HTML/ePub/site）、GitHub 源、SUMMARY.md、live preview、doctor 命令
v0.3.0  已发布  ePub 输出、插件系统
v0.3.1  已发布  CJK PDF 字体嵌入、TOC 深度控制、智能文件名
v0.4.0  已发布  Typst 后端、并行构建、PDF 水印/自定义边距/书签/页脚、PlantUML 支持、构建缓存
v0.4.1  已发布  Bug 修复、测试覆盖扩展、CI 升级（Go 1.25 / actions v5/v6）
v0.4.2  已发布  并行 check、修复慢测试 DNS 超时、gofmt 全量格式化
v0.4.3  已发布  docs 补全、CI 修复（移除 Docker Hub）、glossary 性能优化（预编译正则）
v1.0.0  计划中  目标 2027-Q1，稳定 API、90% 覆盖率、完整文档
```

### 构建与测试健康度（截至 v0.4.3）

| 项目 | 状态 |
|------|------|
| `go build ./...` | ✅ 成功，无警告 |
| `go test ./...` | ✅ 全部 21 个包通过 |
| 总覆盖率 | **62.3%**（目标 v1.0.0 需达 90%） |
| CI（main 分支） | ✅ 成功 |
| Release CI（tag push） | ❌ Docker job 失败（见下文） |

### 各包覆盖率

| 包 | 覆盖率 | 评级 |
|----|--------|------|
| `internal/cover` | 100.0% | ✅ |
| `internal/toc` | 98.3% | ✅ |
| `internal/theme` | 95.1% | ✅ |
| `internal/crossref` | 94.4% | ✅ |
| `internal/variables` | 92.9% | ✅ |
| `internal/glossary` | 92.4% | ✅ |
| `internal/i18n` | 92.0% | ✅ |
| `internal/renderer` | 90.3% | ✅ |
| `internal/linkrewrite` | 92.6% | ✅ |
| `internal/output` | 81.9% | 🟡 |
| `internal/typst` | 76.5% | 🟡 |
| `internal/plugin` | 73.8% | 🟡 |
| `internal/markdown` | 70.5% | 🟡 |
| `internal/server` | 60.5% | 🟠 |
| `internal/pdf` | 55.2% | 🟠 |
| `internal/plantuml` | 54.8% | 🟠 |
| `internal/config` | 50.6% | 🟠 |
| `cmd` | 47.7% | 🔴 |
| `internal/source` | 41.2% | 🔴 |
| `pkg/utils` | 64.9% | 🟡 |

---

## 二、已知问题列表

### 🔴 P0 — 阻塞性问题

1. **Release CI Docker job 持续失败**
   - 现象：每次 tag push 触发的 Release 工作流中，docker job 报 "Log in to Docker Hub" 失败
   - 根因：CI 日志显示 job 步骤为 "Log in to Docker Hub"，但 workflow 代码已改为只推 GHCR，实际原因疑为 secrets 缺失或步骤名称残留导致误判
   - 影响：每个正式 release tag 都带红叉，对外形象差，可能让用户误以为构建失败
   - 建议：检查 release.yml 中 docker job 依赖的 secrets，或确认步骤名称是否与旧 Docker Hub 配置冲突

2. **Node.js 20 actions 弃用警告**
   - 现象：CI 中 `actions/checkout@v4`、`actions/setup-go@v5`、`goreleaser/goreleaser-action@v6` 均使用 Node.js 20，GitHub 将于 2026-06-02 强制切换到 Node.js 24
   - 影响：届时 CI 可能中断
   - 建议：升级到 `@v5`/`@v6`/`@v7` 等支持 Node.js 24 的版本

### 🟠 P1 — 功能缺失

3. **PlantUML 本地渲染未实现**
   - 位置：`internal/plantuml/plantuml.go:141`
   - 现状：`renderLocal()` 直接返回 error，代码注释为 `// TODO: Implement local PlantUML rendering using os/exec`
   - 影响：离线环境或企业内网无法使用 PlantUML；`plantuml` plugin 的 plugin.go 全部函数覆盖率为 0%，表明插件集成路径也未被测试

4. **pdf.Generate / pdf.GenerateFromFile 覆盖率极低**
   - `Generate`：26.7%，`GenerateFromFile`：8.0%
   - 这是核心 PDF 生成路径，几乎没有自动化测试保护
   - 高风险：PDF 回归难以被 CI 发现

5. **cmd 包覆盖率仅 47.7%，source 包 41.2%**
   - `cmd/build_run.go:rendererHeadingsToSiteHeadings`：0%
   - `cmd/chapter_pipeline.go:pdfChapterImageOptions`：0%
   - 大量 CLI 主流程路径无测试

### 🟡 P2 — 质量改善

6. **Typst 后端仍处于 alpha 状态**
   - v0.4.x 修复了多个 Typst bug（heading off-by-one、list indent、input sanitization）
   - 覆盖率 76.5%，但核心转换逻辑边界条件未充分测试
   - 与 Chromium 后端的功能对齐度未量化

7. **总覆盖率 62.3%，距 v1.0.0 目标 90% 差距 27.7 个百分点**
   - 需要系统性补充测试，而非零散补丁

---

## 三、四角色讨论

### 产品经理（PM）

**v0.4.x 交付了什么？**

v0.4.0 是里程碑版本，引入了 Typst 零依赖 PDF 后端，实现了并行构建和构建缓存，并显著增强了 PDF 输出质量（水印、自定义边距、书签、品牌页脚）。v0.4.1~v0.4.3 是质量修复迭代，专注于稳定性、CI 升级和性能优化。

**当前最大问题（PM 视角）**

- Release CI 红叉损害项目可信度，新用户看到 "failure" 会产生顾虑
- PlantUML 仅支持在线服务器，企业用户（内网环境）无法使用
- 版本节奏快（v0.4.0→v0.4.3 在同一天），但缺少版本说明页面（GitHub Release Notes 是否完整？）

**PM 对下个版本的 Top 3 建议**

1. **修复 Release CI**（P0，用户信任问题，1-2 天工作量）
2. **升级 GitHub Actions 到 Node.js 24**（P0，强制截止日期 2026-06-02）
3. **PlantUML 本地渲染**（P1，企业用户痛点，解锁离线场景）

---

### 架构师（Arch）

**v0.4.x 交付了什么？**

v0.4.0 引入了双后端架构（Chromium / Typst），这是一个重要的架构决策。并行构建和 hash-based 缓存为大型书籍的性能奠定了基础。Plugin 系统和 PlantUML 集成验证了插件钩子机制的可行性。

**当前最大问题（Arch 视角）**

- **PlantUML plugin 集成路径覆盖率为 0%**：`plugin.go` 的 `Init`、`Execute`、`Cleanup` 全部未测试，这意味着插件生命周期的正确性没有任何保障
- **双后端缺乏对比测试**：Typst 和 Chromium 后端对同一输入是否产生语义等价的输出？没有 golden test 机制
- **`--backend` 标志的路由逻辑**是否覆盖所有边界情况（backend 不可用时的 fallback、错误消息的清晰度）？

**Arch 对下个版本的 Top 3 建议**

1. **PlantUML 本地渲染（os/exec）**：使用本地 `plantuml.jar` 或 `plantuml` CLI，与 server 模式保持接口一致，通过 config 选择模式
2. **双后端 golden test 框架**：对同一 Markdown 输入，分别用 Chromium 和 Typst 生成，验证结构等价性（章节数、标题层级、图像数量等元数据层面）
3. **plantuml plugin 集成测试**：补充 `plugin.go` 的 Init/Execute/Cleanup 测试，确保插件系统的生命周期管理正确

---

### 程序员（Dev）

**v0.4.x 交付了什么？**

v0.4.1~v0.4.3 改善了代码健康度：`errors.Join` 替代手工错误拼接，gofmt 全量格式化，glossary 正则预编译，并发 check 优化。这些都是好的习惯，但主线功能 PlantUML 本地渲染被搁置了。

**当前最大问题（Dev 视角）**

- `renderLocal()` 是一个空壳，返回 error。实现它需要：找到本地 PlantUML 可执行文件（doctor 命令可以帮助）、通过 stdin/stdout 传递数据、处理进程生命周期
- `pdf.Generate` 和 `pdf.GenerateFromFile` 覆盖率极低（26.7% / 8.0%），这两个函数实际上依赖 Chromium，测试它们需要 integration test 环境，但至少应该有 mock 路径测试错误处理
- CI actions 版本过时，升级有明确 deadline（2026-06-02）

**Dev 对下个版本的 Top 3 建议**

1. **实现 `renderLocal()`**（plantuml.go:141），支持 `plantuml` CLI 和 `plantuml.jar` 两种模式，通过 `PLANTUML_BIN` 或 config 配置路径
2. **补充 `cmd` 和 `source` 包测试**（覆盖率分别为 47.7% / 41.2%），重点是 `rendererHeadingsToSiteHeadings` 和 `pdfChapterImageOptions` 这类零覆盖的函数
3. **GitHub Actions 升级**：`checkout@v5`、`setup-go@v6`，避免 2026-06-02 强制切换导致 CI 中断

---

### 测试（QA）

**v0.4.x 交付了什么？**

v0.4.1 在测试方面投入显著：新增了 root flag、typst generator option、quickstart、image processing、unclosed code block、isFenceClose 等多组测试。这是正确方向，但测试分布不均衡。

**当前最大问题（QA 视角）**

- **整体覆盖率 62.3%**，距 v1.0.0 目标 90% 还差 27.7 个百分点，且缺口集中在最重要的路径上
- **`cmd` 包 47.7%**：CLI 是用户入口，任何 CLI 回归都会直接影响用户，却是覆盖率最低的区域之一
- **`internal/source` 41.2%**：source 包负责从文件系统、GitHub 等加载源文件，是数据进入管道的第一关，测试不足风险高
- **`pdf` 和 `plantuml` 插件路径覆盖率低**，且这些路径依赖外部服务，需要 mock 策略

**QA 对下个版本的 Top 3 建议**

1. **制定测试补充路线图**：针对 `cmd`、`source`、`pdf`、`plantuml` 按季度设定覆盖率里程碑，而非一次性冲刺
2. **为 pdf.Generate 添加 mock-based 测试**：mock Chromium 客户端接口，测试错误处理路径（超时、浏览器不可用、HTML 解析失败）
3. **plantuml plugin 生命周期测试**：`Init` / `Execute` / `Cleanup` 全部为 0%，必须在下个版本补齐

---

## 四、优先级排序与共识

### 综合评分矩阵

| 任务 | 用户价值 | 实现成本 | 紧迫性 | 综合优先级 |
|------|---------|---------|--------|----------|
| 修复 Release CI（Docker job 红叉） | 高（信任） | 低（1-2天） | 高 | **P0** |
| 升级 GitHub Actions（Node.js 24） | 中（CI 稳定） | 低（配置修改） | 高（截止 2026-06-02） | **P0** |
| PlantUML 本地渲染 | 高（企业用户） | 中（3-5天） | 中 | **P1** |
| plantuml plugin 集成测试 | 中（代码质量） | 中 | 中 | **P1** |
| cmd / source 包测试覆盖 | 中（回归保护） | 中（持续投入） | 中 | **P1** |
| pdf.Generate mock 测试 | 中（回归保护） | 中（需要 mock 设计） | 低 | **P2** |
| 双后端 golden test 框架 | 高（长期） | 高（需要设计） | 低 | **P2** |
| 整体覆盖率冲刺至 90% | 高（v1.0 目标） | 高（持续数月） | 低（2027-Q1） | **P3** |

---

### 最终共识：版本规划建议

#### v0.4.4（补丁版本，目标 1-2 周内）

**主题：CI 稳定 + 工具链升级**

- [ ] 修复 Release 工作流 Docker job 失败问题（检查 secrets 或步骤名称）
- [ ] 升级 `actions/checkout`、`actions/setup-go`、`goreleaser/goreleaser-action` 到支持 Node.js 24 的版本
- [ ] 补充 plantuml plugin.go（Init/Execute/Cleanup/EnableIfNeeded）基础测试

**验收标准**：Release CI 全部绿色；GitHub Actions 无 Node.js 20 弃用警告

---

#### v0.5.0（功能版本，目标 4-6 周内）

**主题：PlantUML 本地渲染 + 测试体系建设**

- [ ] 实现 `renderLocal()`：支持 `plantuml` CLI（PATH 查找）和 `plantuml.jar`（通过 `PLANTUML_JAR` 或 config 指定）
- [ ] 更新 `doctor` 命令，检测本地 PlantUML 可用性
- [ ] 补充 `cmd` 包测试至 65%+（重点：`rendererHeadingsToSiteHeadings`、`pdfChapterImageOptions`）
- [ ] 补充 `internal/source` 包测试至 60%+
- [ ] 为 `pdf.Generate` 设计 mock-based 错误路径测试
- [ ] 双后端基础 golden test：对标准 Markdown fixture，验证输出元数据等价

**验收标准**：总覆盖率 ≥ 68%；离线 PlantUML 渲染可用；plantuml plugin 覆盖率 ≥ 60%

---

#### v0.6.0 及更远（面向 v1.0.0 铺路）

- 覆盖率持续提升，目标 75%+
- `mdpress doctor` 增强：检测 Chrome、Typst、PlantUML、字体、网络
- `mdpress upgrade` 命令（自更新）
- 考虑补充平台 CI matrix（macOS + Linux + Windows）
- API 稳定性审查，为 v1.0.0 冻结做准备

---

## 五、行动项（当前 sprint）

| # | 任务 | 负责角色 | 目标版本 |
|---|------|---------|---------|
| 1 | 排查并修复 Release CI docker job 失败 | Dev + Arch | v0.4.4 |
| 2 | 升级 GitHub Actions 到 Node.js 24 兼容版本 | Dev | v0.4.4 |
| 3 | 补充 plantuml plugin.go 基础测试 | QA + Dev | v0.4.4 |
| 4 | 实现 PlantUML 本地渲染（os/exec） | Dev | v0.5.0 |
| 5 | 更新 doctor 命令检测 PlantUML | Dev | v0.5.0 |
| 6 | cmd 包测试覆盖专项提升 | QA | v0.5.0 |
| 7 | source 包测试覆盖专项提升 | QA | v0.5.0 |
| 8 | pdf.Generate mock 测试设计 | Arch + QA | v0.5.0 |
| 9 | 双后端 golden test 框架设计 | Arch | v0.5.0 |

---

*本文档为产品委员会内部讨论记录，不纳入 git 版本管理，仅供本地参考。*
