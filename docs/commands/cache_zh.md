# `mdpress cache`

[English](cache.md)

## 用途

查看或清空 mdPress 在多次构建之间保留的缓存。

mdPress 会把解析后的章节（以及其他构建中间产物）缓存在一个运行时缓存目录下，使未改动的章节
不必重新渲染。两周未使用的条目会被自动清理，因此这个命令用于立刻回收空间，或强制一次完全
冷启动的重建。

## 语法

```bash
mdpress cache info
mdpress cache clear
```

## 子命令

| 子命令 | 说明 |
| --- | --- |
| `info` | 查看缓存位置、条目数和占用大小 |
| `clear` | 删除全部缓存条目 |

## 输出示例

```text
  mdpress Cache
  ──────────────────────────────────────────────────

  Location: /tmp/mdpress-cache
  Entries:  1042
  Size:     18.3 MB

    chrome-runtime            0 entries         0 B
    images                    4 entries     38.0 KB
    parsed-chapters        1038 entries     18.3 MB

  Run 'mdpress cache clear' to reclaim this space.
```

## 缓存位置

| 设置 | 效果 |
| --- | --- |
| 默认 | 依赖操作系统的临时目录（例如 `$TMPDIR/mdpress-cache`） |
| `--cache-dir <path>` | 本次命令使用该目录 |
| `MDPRESS_CACHE_DIR` | 同上，以环境变量形式给出 —— 在 CI 中很有用，缓存需要放在任务能够恢复的位置 |
| `--no-cache` | 让单次命令绕过缓存，不删除任何东西 |

## 注意事项

- `cache clear` 不会碰你的项目，它只删除缓存条目。
- 任何时候删除缓存都是安全的 —— 下一次构建只是重新渲染每个章节而已。
- 如果某个渲染修复看起来没有生效，请先用 `--no-cache`：它能在不丢掉整个缓存的前提下判断是不是缓存的问题。
