# Plugin Overview

mdPress plugins are external executables declared in `book.yaml`. They are not in-process Go plugins.

## Configure A Plugin

```yaml
plugins:
  - name: word-count
    path: ./examples/plugins/word-count
    config:
      warn_threshold: 500
```

- `name` is the plugin identifier.
- `path` points to the executable relative to `book.yaml`.
- `config` is passed through as JSON.

Plugins run in declaration order.

## Protocol

mdPress probes each executable with `--mdpress-info` and `--mdpress-hooks` when those flags are available. During hook execution it sends JSON on stdin and reads JSON from stdout. Anything written to stderr is captured in the logs.

If the helper flags are missing, mdPress falls back to version `0.1.0` and subscribes the plugin to all phases.

## Hook Phases

| Phase | When It Runs |
| --- | --- |
| `before_build` | After config load, before chapter processing. |
| `after_parse` | After a chapter has been rendered to HTML. |
| `before_render` | Before final HTML assembly. |
| `after_render` | After the HTML document has been assembled. |
| `after_build` | After all output files are written. |
| `before_serve` | Before the live preview server starts. |
| `after_serve` | When the live preview server shuts down. |

## Hook Data

Each hook receives a `HookContext` with:

- the current config
- the active phase
- the current content payload
- chapter index and source file
- output path and format when relevant
- a shared `Metadata` map for passing state between phases

If a plugin returns non-empty `content`, mdPress replaces the current payload. If it returns `stop: true`, later plugins in the same phase are skipped.
