# `mdpress cache`

[中文说明](cache_zh.md)

## Purpose

Inspect or clear the caches mdPress keeps between builds.

mdPress caches parsed chapters (and other build intermediates) under a runtime cache
directory so unchanged chapters are not re-rendered. Entries unused for two weeks are
pruned automatically, so this command is for reclaiming the space now, or for forcing a
fully cold rebuild.

## Syntax

```bash
mdpress cache info
mdpress cache clear
```

## Subcommands

| Subcommand | Description |
| --- | --- |
| `info` | Show the cache location, entry count, and size |
| `clear` | Delete every cached entry |

## Example Output

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

## Where The Cache Lives

| Setting | Effect |
| --- | --- |
| default | An OS-dependent temporary directory (e.g. `$TMPDIR/mdpress-cache`) |
| `--cache-dir <path>` | Use this directory for the current command |
| `MDPRESS_CACHE_DIR` | Same, as an environment variable — useful in CI, where the cache must live somewhere the job can restore |
| `--no-cache` | Bypass the cache for a single command without deleting anything |

## Notes

- `cache clear` never touches your project; it only deletes cache entries.
- Deleting the cache is safe at any time — the next build simply re-renders every chapter.
- If a rendering fix appears not to have taken effect, prefer `--no-cache` first: it proves
  whether the cache is involved without throwing the whole cache away.
