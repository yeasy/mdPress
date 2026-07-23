# Plugin API Reference

An mdPress plugin is **any executable file**. There is no SDK, no shared
library, and no plugin manifest file. mdPress talks to a plugin by running it
and exchanging one JSON document over stdin/stdout.

This page is the normative description of that protocol. It matches
`internal/plugin/external.go`; the reference plugin is
[`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count).

> **Plugins are arbitrary code.** Declaring a plugin in `book.yaml` means
> mdPress executes that file on your machine. Read [Trust Model](#trust-model)
> before building someone else's project.

## Declaring A Plugin

```yaml
plugins:
  - name: reading-time
    path: ./plugins/reading-time
    config:
      words_per_minute: 200
```

| Key | Required | Meaning |
| --- | --- | --- |
| `name` | yes | Identifier used in log messages and `mdpress doctor` output. |
| `path` | yes | Path to the executable, **relative to `book.yaml`**. |
| `config` | no | Arbitrary YAML mapping, handed to the plugin verbatim on every call. |

Those three keys are the whole schema. `command:`, `hooks:` and `enabled:` are
not recognised — `command:` makes the entry fail with *"is missing the required
'path' field"*, and any other extra key produces an *"unknown key in config
file"* warning and is ignored. A plugin selects its own hooks (see
[Capability Discovery](#capability-discovery)); `book.yaml` cannot filter them.

Path rules, enforced when plugins are loaded:

- Absolute paths are rejected: *"absolute paths are not allowed; use a path
  relative to the project directory"*.
- A relative path that resolves outside the project directory (`../mdpress`, or
  a symlink pointing out of the tree) is rejected too.
- Outside Windows the file must carry the execute bit.
- On Windows an extension-less path also tries the suffixes in `PATHEXT`.

If **any** entry fails these checks, mdPress logs one warning and runs the
build with **no plugins at all** — plugin loading is all-or-nothing:

```
WARN plugin loading failed (continuing without plugins): failed to load
     plugin "missing": plugin executable not found at ".../does-not-exist"
```

Plugins run in declaration order.

## Capability Discovery

When a plugin is loaded, mdPress runs it twice with a single flag each time.
Both calls have a **5 second** timeout, and their stdout is capped at 1 MB.

### `--mdpress-info`

```console
$ ./plugins/reading-time --mdpress-info
{"version":"1.0.0","description":"Prepends an estimated reading time to each chapter."}
```

Only `version` and `description` are read, and both are cosmetic — they show up
in `mdpress doctor --verbose`:

```
✓ Plugin "reading-time" responds (version 1.0.0, 1 hook(s))
```

If the call fails, times out, or does not print JSON, mdPress falls back to
version `0.1.0` and an empty description.

### `--mdpress-hooks`

```console
$ ./plugins/reading-time --mdpress-hooks
["after_parse"]
```

A JSON array of phase names. The plugin is only invoked for the phases it
lists. If the call fails or does not print a JSON array, mdPress subscribes the
plugin to **all seven** phases, so it gets executed far more often than it
probably wants.

Answering neither flag is not fatal, but `mdpress doctor` calls it out:

```
⚠ Plugin "doc-protocol" does not speak the mdpress plugin protocol
  (no valid --mdpress-info or --mdpress-hooks response)
```

## Hook Invocation

For each hook, mdPress starts a **fresh process** with no arguments, writes one
JSON object to its stdin, closes stdin, and reads one JSON object from its
stdout. The process is expected to exit when it is done.

Because every invocation is a new process, a plugin **cannot keep state in
memory between hooks**. Persist to a file if you need to accumulate anything
across chapters.

The invocation timeout is **30 seconds**. stdout and stderr are each capped at
10 MB. The plugin inherits the working directory mdPress was started in.

### Request

```json
{
  "phase": "after_parse",
  "content": "<h1 id=\"introduction\">Introduction</h1>\n<p>Hello…</p>",
  "chapter_index": 0,
  "chapter_file": "chapters/01-intro.md",
  "output_path": "",
  "output_format": "",
  "config": { "words_per_minute": 200 },
  "metadata": {}
}
```

All eight keys are always present. What they actually carry:

| Field | Type | Value |
| --- | --- | --- |
| `phase` | string | The phase name, e.g. `after_parse`. |
| `content` | string | The payload for this phase — see [Lifecycle Hooks](./lifecycle-hooks.md). Empty in `before_build` and `after_build`. |
| `chapter_index` | number | Zero-based chapter index. Only meaningful in `after_parse`; `0` everywhere else. |
| `chapter_file` | string | Chapter source path. Only set in `after_parse`. |
| `output_path` | string | **Always empty today.** No call site fills it in. |
| `output_format` | string | **Always empty today.** No call site fills it in. |
| `config` | object | The `config:` mapping from `book.yaml`, or `{}`. |
| `metadata` | object | **Always empty today.** The response has no way to write to it. |

There is no book title, author, theme, chapter list, or output directory in the
request. A plugin that needs project metadata has to read `book.yaml` itself.

### Response

```json
{
  "content": "<p class=\"reading-time\">3 min read</p>\n<h1>…</h1>",
  "stop": false,
  "error": ""
}
```

| Field | Type | Effect |
| --- | --- | --- |
| `content` | string | Replaces the payload — **but only in `after_parse`**. An empty string, or an omitted key, means "leave the content alone". |
| `stop` | bool | Skip the remaining plugins for this phase. |
| `error` | string | Non-empty means the plugin failed; see [Failure Handling](#failure-handling). |

Those three fields are the entire response schema. `status`, `action`,
`modified_content`, `content_type`, `metadata`, `data`, `warnings` and `errors`
are **not part of the protocol**: a response made only of those keys parses
fine, is read as "no change", and the plugin silently does nothing.

Printing nothing at all is legal and also means "no change".

## Failure Handling

A plugin failure is always a **warning**. It never fails the build and never
changes the exit code.

| Situation | What mdPress does |
| --- | --- |
| Non-zero exit | `WARN … plugin exited with error: exit status 1` — stderr is appended to the message. |
| `"error"` set in the response | `WARN … plugin "name" reported error: <text>` |
| Response is not valid JSON | `WARN … failed to parse plugin response` plus the first 200 characters. |
| Timeout (30 s) | The process is killed and the failure is reported as above. |

Content from a failed invocation is discarded; the build continues with the
original content and with the remaining plugins.

**stderr from a successful invocation is discarded.** It is only surfaced when
the process exits non-zero, so stderr is a crash-report channel, not a logging
channel. A plugin that wants a visible progress log should write its own file.

## Trust Model

A plugin is an executable that mdPress runs during `build` and `serve` —
including at load time, for the `--mdpress-info` / `--mdpress-hooks` probes.
Opening a project therefore runs code that the project ships.

- **Local sources always run their plugins.** There is no flag that turns this
  off; if you do not trust a directory, do not build it.
- **Remote sources refuse to run plugins** unless you opt in:

  ```bash
  mdpress build https://github.com/owner/repo --allow-plugins
  ```

  Without the flag you get `Refusing to run N plugin(s) from a remote project;
  pass --allow-plugins to trust and execute them.` and the build proceeds with
  no plugins. `--allow-plugins` exists on both `build` and `serve` and has **no
  effect on local projects** — those are already trusted.

The containment rules in [Declaring A Plugin](#declaring-a-plugin) — no
absolute paths, nothing outside the project directory — limit *which* file
runs, not *what* it may do. A plugin runs with your full user privileges.

## Reference Implementation

[`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count)
is a complete Go plugin that implements both probe flags and the `after_parse`
hook:

```bash
go build -o plugins/word-count ./examples/plugins/word-count
```

```yaml
plugins:
  - name: word-count
    path: ./plugins/word-count
    config:
      warn_threshold: 500
```

See [Building a Plugin](./building-a-plugin.md) for a step-by-step tutorial and
[Lifecycle Hooks](./lifecycle-hooks.md) for what each phase can actually do.
