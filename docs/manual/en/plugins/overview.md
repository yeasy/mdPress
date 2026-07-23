# Plugin Overview

mdPress plugins are external executables declared in `book.yaml`. They are not in-process Go plugins.

> **Security: plugins are arbitrary code.** Because a plugin is any executable
> named in `book.yaml`, mdPress runs it on your machine during **build** and
> **serve** — including at plugin-probe time (`--mdpress-info` /
> `--mdpress-hooks`). That means `mdpress build <github-url>`, or opening any
> third-party project, can run arbitrary code from that project. Only
> build or serve projects you trust.
>
> As a guard, remote sources (such as a GitHub URL) **refuse** to run a
> project's plugins unless you opt in with `--allow-plugins` (added in
> v0.7.12). Local sources always run their declared plugins.
>
> ```bash
> mdpress build https://github.com/owner/repo --allow-plugins
> ```

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

If any entry fails to load, mdPress warns once and builds with **no plugins at all**.

## Protocol

mdPress probes each executable with `--mdpress-info` and `--mdpress-hooks`. For each hook it then starts a fresh process, sends one JSON object on stdin and reads one JSON object from stdout.

If the helper flags are missing, mdPress falls back to version `0.1.0` and subscribes the plugin to all phases.

stderr is only surfaced when a plugin exits non-zero. On a successful run it is discarded, so it is not a logging channel.

## Hook Phases

| Phase | When It Runs | Can Change Content |
| --- | --- | --- |
| `before_build` | Once, after config load, before chapter processing. | no |
| `after_parse` | Once per chapter, after it is rendered to HTML. Also the only hook `mdpress serve` dispatches. | **yes** |
| `before_render` | Once, before final HTML assembly. Payload is the cover HTML. | no |
| `after_render` | Once, after the HTML document is assembled. Payload is the TOC HTML. | no |
| `after_build` | Once, after all output files are written. | no |
| `before_serve` | Declared in the protocol but **never dispatched**. | — |
| `after_serve` | Declared in the protocol but **never dispatched**. | — |

## Hook Data

Each request carries the phase, the content payload, the chapter index and source file (`after_parse` only), and the plugin's own `config` block. `output_path`, `output_format` and `metadata` are part of the wire format but are always empty today.

A plugin that returns non-empty `content` replaces the payload — but only in `after_parse`; every other phase discards it. Returning `stop: true` skips the later plugins for that phase. Returning a non-empty `error` produces a build warning and never fails the build.
