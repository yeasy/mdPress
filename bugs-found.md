# mdPress Bug 检测报告

> 日期：2026-03-19
> 审查范围：全部 Go 源文件（63 个源文件 + 51 个测试文件）

---

## 🔴 高优先级 Bug（可导致程序崩溃或数据问题）

### Bug #1: server.go 关闭时 debounceTimer 泄漏导致竞态条件

**文件：** `internal/server/server.go` 第 489-569 行（fsnotify）和第 580-614 行（polling）

**问题描述：** `watchFilesWithFsnotify()` 和 `watchFilesPolling()` 中的 debounceTimer 在函数退出时没有被停止。当 `ctx.Done()` 触发函数返回后，已经创建的 `time.AfterFunc` 回调仍然会执行，访问 `s.BuildFunc`、`s.notifyClients()` 等服务器状态。

**重现场景：**
1. 用户修改文件触发 debounce（创建 500ms 定时器）
2. 在 500ms 内按 Ctrl+C 关闭服务器
3. ctx 被取消，`watchFilesWithFsnotify` 返回
4. 定时器回调仍然执行，调用 `s.BuildFunc()` 和 `s.notifyClients()`
5. 此时服务器可能正在关闭，HTTP server 已 Shutdown，WebSocket 连接已关闭

**影响：** 关闭服务器时可能出现 panic（写入已关闭的连接）、日志中出现混乱错误、或在极端情况下产生数据竞争。

**修复方案：**
```go
// watchFilesWithFsnotify 和 watchFilesPolling 都需要添加：
defer func() {
    debounceMu.Lock()  // fsnotify 版本需要
    if debounceTimer != nil {
        debounceTimer.Stop()
    }
    debounceMu.Unlock()
}()
```

---

### Bug #2: quickstart.go `filepath.Glob("*")` 无法匹配隐藏文件

**文件：** `cmd/quickstart.go` 第 44-50 行

**问题代码：**
```go
if utils.FileExists(absDir) {
    entries, err := filepath.Glob(filepath.Join(absDir, "*"))
    if err == nil && len(entries) > 0 {
        return fmt.Errorf("directory %s already exists and is not empty; choose a new directory name", dir)
    }
}
```

**问题描述：** Go 的 `filepath.Glob("*")` 不匹配以 `.` 开头的隐藏文件/目录。如果目标目录只包含隐藏文件（如 `.git`、`.DS_Store`），`Glob` 返回空切片，代码认为目录"为空"并继续创建示例项目，可能覆盖已有的 Git 仓库结构。

**重现场景：**
```bash
mkdir my-book && cd my-book && git init
# 此时目录包含 .git/
mdpress quickstart my-book  # 不会报错，直接写入文件
```

**影响：** 可能在已初始化的 Git 仓库中意外创建示例项目文件。

**修复方案：**
```go
if utils.FileExists(absDir) {
    entries, err := os.ReadDir(absDir)
    if err != nil {
        return fmt.Errorf("failed to read directory: %w", err)
    }
    if len(entries) > 0 {
        return fmt.Errorf("directory %s already exists and is not empty; choose a new directory name", dir)
    }
}
```

---

### Bug #3: quickstart.go `filepath.Glob` 错误被静默忽略

**文件：** `cmd/quickstart.go` 第 46-49 行

**问题描述：** 条件 `if err == nil && len(entries) > 0` 当 `filepath.Glob` 返回错误时（如权限问题），代码静默跳过非空目录检查，直接进入文件创建流程。

**影响：** 权限受限的目录可能被意外写入（或写入失败后报出更隐晦的错误）。

**修复方案：** 与 Bug #2 同一个修复一并解决。

---

## 🟡 中优先级 Bug（可能导致不正确的行为）

### Bug #4: build_run.go 多语言构建中的语言代码覆盖逻辑有误

**文件：** `cmd/build_run.go` 第 62-65 行

**问题代码：**
```go
if guessed := guessLanguageCode(lang.Dir); guessed != "" {
    if langCfg.Book.Language == "" || (langCfg.Book.Language == "zh-CN" && guessed != "zh-CN") {
        langCfg.Book.Language = guessed
    }
}
```

**问题描述：** 第二个条件 `langCfg.Book.Language == "zh-CN" && guessed != "zh-CN"` 的意图不明确。它表示"如果当前语言是 zh-CN 且猜测的语言不是 zh-CN，则覆盖"。这可能是为了处理 `config.Load()` 中默认值为 `"zh-CN"` 的情况，但逻辑有以下问题：

- 如果用户在 book.yaml 中明确设置了 `language: zh-CN`，目录名却是 `en/`，那么 guessed 是 `"en"`，代码会**错误地覆盖**用户的显式配置
- 条件应该区分"默认值 zh-CN"和"用户显式设置的 zh-CN"

**影响：** 多语言项目中，如果某个语言子目录的 book.yaml 设置了 `language: zh-CN`，但目录名暗示是其他语言，会被错误覆盖。

**修复建议：** 在 config 中添加字段标记语言是否由用户显式设置，或只在 language 为空时覆盖：
```go
if guessed := guessLanguageCode(lang.Dir); guessed != "" {
    if langCfg.Book.Language == "" {
        langCfg.Book.Language = guessed
    }
}
```

---

### Bug #5: image.go `prefetchRemoteImages` 中 Logger 在并发 goroutine 中未安全捕获

**文件：** `pkg/utils/image.go` 第 335-376 行

**问题代码：**
```go
go func(src string) {
    // ...
    if options.Logger != nil {
        options.Logger.Warn(...)
    }
    // ...
}(src)
```

**问题描述：** `options` 是值类型（struct），传入 goroutine 时通过闭包捕获。由于 `ImageProcessingOptions` 包含 `*slog.Logger` 指针字段，多个 goroutine 共享同一个 Logger 指针。虽然 `slog.Logger` 本身是并发安全的，但闭包捕获的是 `options`（值拷贝），其中 Logger 指针指向同一个对象。

**实际风险评估：** 由于 `slog.Logger` 是并发安全的，且 `options` 在 goroutine 启动后不会被外部修改，这个问题的实际风险很低。但如果 Logger 接口的具体实现不是并发安全的，就会出现竞态。

**修复方案（防御性编程）：**
```go
logger := options.Logger  // 在 goroutine 启动前捕获
for src := range unique {
    go func(src string) {
        // 使用 logger 而非 options.Logger
    }(src)
}
```

---

### Bug #6: epub.go `writeZipFile` 和 `writeZipBinaryFile` 写入错误未包装上下文

**文件：** `internal/output/epub.go` 第 446-461 行

**问题代码：**
```go
func writeZipFile(w *zip.Writer, name, content string) error {
    fw, err := w.Create(name)
    if err != nil {
        return fmt.Errorf("failed to create %s: %w", name, err)
    }
    _, err = fw.Write([]byte(content))
    return err  // ← 缺少上下文包装
}
```

**问题描述：** `Create` 的错误有上下文包装，但 `Write` 的错误直接返回裸错误。当 ePub 生成过程中写入失败时，错误信息不包含是哪个文件写入失败，难以调试。

**影响：** 不会导致崩溃，但会导致 ePub 生成失败时的错误信息不可读。

**修复方案：**
```go
_, err = fw.Write([]byte(content))
if err != nil {
    return fmt.Errorf("failed to write %s: %w", name, err)
}
return nil
```

---

## 🟢 低优先级 Bug（极端边界情况）

### Bug #7: serve.go 第 219/232 行 `os.RemoveAll(backupDir)` 错误静默丢弃

**文件：** `cmd/serve.go` 第 219 行和第 232 行

**问题描述：** 原子目录交换后，备份目录删除失败时错误被 `_ = os.RemoveAll(backupDir)` 丢弃。如果连续触发多次重建，备份目录可能累积占用磁盘空间。

**影响：** 开发环境中磁盘空间可能缓慢泄漏。

**修复方案：** 添加 `logger.Debug` 级别日志记录。

---

### Bug #8: chapter_cache.go 缓存写入非原子操作

**文件：** `cmd/chapter_cache.go` 第 49-58 行

**问题描述：** `storeParsedChapterCache` 直接使用 `os.WriteFile` 写入最终路径。如果在写入过程中程序崩溃或断电，缓存文件可能处于损坏状态。虽然 `loadParsedChapterCache` 对 JSON 反序列化失败做了降级处理（返回 cache miss），但损坏文件会一直留在磁盘上直到内容变更。

**影响：** 极端情况下缓存文件损坏，但不会导致构建失败（降级为重新解析）。

**修复方案：** 使用 tmpFile + Rename 原子写入模式（与 image.go 一致）。

---

## 汇总

| # | 严重程度 | 文件 | 简述 |
| --- | --- | --- | --- |
| **#1** | 🔴 高 | server.go | debounceTimer 泄漏导致关闭时竞态 |
| **#2** | 🔴 高 | quickstart.go | Glob 不匹配隐藏文件，可能覆盖 .git |
| **#3** | 🔴 高 | quickstart.go | Glob 错误被静默忽略 |
| **#4** | 🟡 中 | build_run.go | 多语言构建中语言代码覆盖逻辑有误 |
| **#5** | 🟡 中 | image.go | Logger 在并发 goroutine 中未安全捕获（实际风险低） |
| **#6** | 🟡 中 | epub.go | writeZipFile 写入错误缺少上下文 |
| **#7** | 🟢 低 | serve.go | RemoveAll 错误静默丢弃 |
| **#8** | 🟢 低 | chapter_cache.go | 缓存写入非原子操作 |
