# Lifecycle Hooks in Detail

Seven phase names exist in the protocol. **Five of them fire during
`mdpress build`, one fires during `mdpress serve`, and two are declared but
never dispatched.** Only one phase can actually change the output.

This page says exactly what each phase gives you, so you can pick the right one
instead of discovering the answer by trial and error. The wire format is in the
[Plugin API Reference](./api.md).

## What Fires, And What It Can Do

| Phase | Fires in `build` | Fires in `serve` | `content` payload | Returned `content` used? |
| --- | --- | --- | --- | --- |
| `before_build` | once | no | empty | no |
| `after_parse` | once per chapter | once per chapter, per rebuild | that chapter's rendered HTML | **yes** |
| `before_render` | once | no | the cover HTML | no |
| `after_render` | once | no | the table-of-contents HTML | no |
| `after_build` | once | no | empty | no |
| `before_serve` | never | never | — | — |
| `after_serve` | never | never | — | — |

Two consequences worth internalising before you write anything:

1. **`after_parse` is the only phase that can modify the book.** In every other
   phase the value you return in `content` is read and then thrown away. If you
   want to change what a reader sees, do it in `after_parse`.
2. **`before_serve` and `after_serve` never run.** The names are defined in the
   protocol and accepted by `--mdpress-hooks`, but no code path dispatches
   them. A plugin that only subscribes to those two will be loaded, will show
   up as valid in `mdpress doctor`, and will never be executed.

Everything else is still useful as a *side-effect* hook: the process runs, so it
can write files, call an API, or fail loudly. It just cannot hand content back.

## Build Sequence

```
mdpress build
  │
  ├─ load book.yaml, theme, plugins   ← plugins are probed (executed!) here
  │
  ├─ [before_build]                   content: ""
  │
  ├─ parse every chapter to HTML
  │    └─ [after_parse]  per chapter  content: chapter HTML  → replaceable
  │
  ├─ build cover + TOC
  ├─ [before_render]                  content: cover HTML
  ├─ assemble the single-page HTML
  ├─ [after_render]                   content: TOC HTML
  │
  ├─ write every requested format (html, pdf, epub, site, typst…)
  │
  └─ [after_build]                    content: ""
```

Note that `before_render` / `after_render` / `after_build` fire **once per
build**, not once per output format — a `--format html,pdf,epub` run still
dispatches each of them exactly once, and there is no way to tell from the
request which formats were requested.

## Serve Sequence

```
mdpress serve
  │
  ├─ load book.yaml, theme, plugins   ← plugins are probed (executed!) here
  ├─ initial render
  │    └─ [after_parse]  per chapter
  │
  └─ on every file change
       └─ [after_parse]  per chapter
```

`mdpress serve` re-runs the chapter pipeline only, so `after_parse` is the sole
hook it dispatches. A plugin that does its work in `after_build` will appear to
"not run in dev mode" — that is why.

## `before_build`

Fires once, after `book.yaml` and the theme are loaded, before any chapter is
parsed.

Request: `phase` is `before_build`; `content`, `chapter_file`, `output_path` and
`output_format` are empty; `config` holds your settings.

Good for: preflight checks (is an API key present? is a converter installed?),
clearing a scratch directory your plugin owns, recording a build start time.

Not good for: anything that needs to change the build. There is no
configuration write-back — mdPress has already loaded its config, and your
`content` is ignored.

```bash
#!/bin/sh
# preflight — refuse to build without an API token.
case "$1" in
  --mdpress-info)  echo '{"version":"1.0.0","description":"Checks build prerequisites."}'; exit 0 ;;
  --mdpress-hooks) echo '["before_build"]'; exit 0 ;;
esac

cat > /dev/null   # drain the request; we do not need it

if [ -z "$DOCS_API_TOKEN" ]; then
  echo '{"error":"DOCS_API_TOKEN is not set"}'
else
  echo '{}'
fi
```

Reporting an `error` produces a warning. It does **not** abort the build — see
[Failure Handling](./api.md#failure-handling).

## `after_parse`

Fires once per chapter, right after that chapter's Markdown has been converted
to HTML and before headings, figures and tables are registered for
cross-references. Anything you inject is numbered like hand-written content.

Request: `content` is the chapter's HTML, `chapter_file` is its source path, and
`chapter_index` is its zero-based position in the flattened chapter list.

This is the phase for content work: injecting banners, rewriting elements,
counting words, validating structure.

```python
#!/usr/bin/env python3
"""reading-time — prepend an estimated reading time to every chapter."""
import json
import re
import sys

TAGS = re.compile(r"<[^>]+>")

if len(sys.argv) > 1:
    if sys.argv[1] == "--mdpress-info":
        json.dump({"version": "1.0.0",
                   "description": "Prepends an estimated reading time."}, sys.stdout)
        sys.exit(0)
    if sys.argv[1] == "--mdpress-hooks":
        json.dump(["after_parse"], sys.stdout)
        sys.exit(0)

req = json.load(sys.stdin)

wpm = req.get("config", {}).get("words_per_minute", 200)
words = len(TAGS.sub(" ", req.get("content", "")).split())
minutes = max(1, round(words / wpm))

banner = f'<p class="reading-time">{minutes} min read</p>\n'
json.dump({"content": banner + req.get("content", "")}, sys.stdout)
```

Remember that `content` is **HTML, not Markdown** — the conversion has already
happened. A plugin that searches for `## ` or `[text](link)` will find nothing.

Returning `"content": ""` (or omitting the key) leaves the chapter untouched,
which is what a read-only plugin should do.

### Stopping the chain

Set `"stop": true` to skip every plugin declared after yours for this phase:

```json
{"content": "<p>…</p>", "stop": true}
```

The chapter keeps whatever content you returned; later plugins simply do not
run for it. `stop` has no effect on other phases or other chapters.

## `before_render`

Fires once, after all chapters are parsed, just before the single-page HTML
document is assembled.

Request: `content` holds the **cover HTML** — not a chapter, and not the whole
document. `chapter_file`, `output_path` and `output_format` are empty.

The cover is passed as a representative payload for inspection. Returning a
modified cover has no effect; the assembled document uses the cover mdPress
already built.

Good for: side effects that must happen after all content exists but before
output files are written.

## `after_render`

Fires once, after the single-page HTML document has been assembled and before
the formats are written.

Request: `content` holds the **table-of-contents HTML**. Same story as
`before_render`: it is there for inspection, and the return value is discarded.

Good for: extracting the document outline, checking that every chapter made it
into the TOC.

## `after_build`

Fires once, after every requested format has been written to disk.

Request: `content`, `chapter_file`, `output_path` and `output_format` are all
empty. **This phase does not tell you where the output went.** If your plugin
needs the path, read `output.filename` from `book.yaml`, or pass the path
through your own `config:` block:

```yaml
plugins:
  - name: publish
    path: ./plugins/publish
    config:
      artifact: dist/book.pdf
```

Good for: uploading artifacts, sending a notification, generating a report next
to a path you already know.

```python
#!/usr/bin/env python3
"""notify — copy the built artifact somewhere once the build is done."""
import json
import shutil
import sys

if len(sys.argv) > 1:
    if sys.argv[1] == "--mdpress-info":
        json.dump({"version": "1.0.0",
                   "description": "Copies the built artifact to a drop directory."}, sys.stdout)
        sys.exit(0)
    if sys.argv[1] == "--mdpress-hooks":
        json.dump(["after_build"], sys.stdout)
        sys.exit(0)

req = json.load(sys.stdin)
cfg = req.get("config", {})

try:
    shutil.copy(cfg["artifact"], cfg["drop_dir"])
except Exception as exc:                       # surfaced as a build warning
    json.dump({"error": f"publish failed: {exc}"}, sys.stdout)
else:
    json.dump({}, sys.stdout)
```

## `before_serve` And `after_serve`

Declared in the protocol, accepted by `--mdpress-hooks`, **never dispatched**.
No part of `mdpress serve` calls them today.

They are listed here so you do not spend an afternoon debugging a plugin that
was never going to run. If you need to react to serve, use `after_parse` — it
fires on the initial render and on every rebuild — and keep in mind that it
fires per chapter, so "once per rebuild" work needs its own guard.

## Choosing A Phase

| You want to… | Use |
| --- | --- |
| Change what readers see | `after_parse` |
| Validate content and warn | `after_parse` |
| Check prerequisites before work starts | `before_build` |
| Inspect the cover or TOC | `before_render` / `after_render` |
| Publish, notify, or archive when the build is done | `after_build` |
| React to a live-reload rebuild | `after_parse` |

Subscribe only to the phases you use. A plugin whose `--mdpress-hooks` call
fails is subscribed to all seven, which means one process spawn per chapter
plus four more per build for no reason.

See the [Plugin API Reference](./api.md) for the exact request and response
schema, and [Building a Plugin](./building-a-plugin.md) for a full walkthrough.
