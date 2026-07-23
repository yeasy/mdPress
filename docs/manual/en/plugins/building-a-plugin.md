# Building a Plugin: Step-by-Step Tutorial

We are going to build `reading-time`: a plugin that prepends
*"3 min read"* to every chapter. It is small enough to read in one sitting and
exercises every part of the protocol — the two discovery flags, per-chapter
content rewriting, and plugin configuration from `book.yaml`.

Python is used here because it is short. **Any executable works**: a shell
script, a Go binary, a compiled Rust program. mdPress only cares that it runs
and speaks JSON on stdin/stdout.

## Step 1: A Project To Build

```bash
mkdir -p demo/chapters demo/plugins
cd demo
```

`book.yaml`:

```yaml
book:
  title: "Reading Time Demo"
  author: "You"

chapters:
  - title: "Introduction"
    file: chapters/01-intro.md
```

`chapters/01-intro.md`:

```markdown
# Introduction

Some words to count in this chapter.
```

Check that a plain build works before adding a plugin:

```bash
mdpress build --format html
```

> Chapter entries are mappings with `title:` and `file:` keys. A bare string
> (`- chapters/01-intro.md`) is a YAML type error, not a shortcut.

## Step 2: The Discovery Flags

A plugin must answer two flags before it is asked to do anything. Create
`plugins/reading-time` with just that much:

```python
#!/usr/bin/env python3
"""reading-time: prepends an estimated reading time to every chapter."""
import json
import sys


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] == "--mdpress-info":
            json.dump({"version": "1.0.0",
                       "description": "Prepends an estimated reading time to each chapter."},
                      sys.stdout)
            return
        if sys.argv[1] == "--mdpress-hooks":
            json.dump(["after_parse"], sys.stdout)
            return


main()
```

Make it executable — mdPress refuses a plugin without the execute bit:

```bash
chmod +x plugins/reading-time
```

Test the two flags by hand:

```console
$ ./plugins/reading-time --mdpress-info
{"version": "1.0.0", "description": "Prepends an estimated reading time to each chapter."}
$ ./plugins/reading-time --mdpress-hooks
["after_parse"]
```

`--mdpress-hooks` is how a plugin subscribes to phases. Answering
`["after_parse"]` means mdPress runs this program once per chapter and never
for the other phases. A plugin that does **not** answer this flag is subscribed
to all seven phases, which is almost never what you want.

## Step 3: Handle The Hook

When a hook fires, mdPress runs the same program with **no arguments**, writes
one JSON request to stdin, and reads one JSON response from stdout. Add the
body:

```python
#!/usr/bin/env python3
"""reading-time: prepends an estimated reading time to every chapter."""
import json
import re
import sys

TAGS = re.compile(r"<[^>]+>")


def main():
    if len(sys.argv) > 1:
        if sys.argv[1] == "--mdpress-info":
            json.dump({"version": "1.0.0",
                       "description": "Prepends an estimated reading time to each chapter."},
                      sys.stdout)
            return
        if sys.argv[1] == "--mdpress-hooks":
            json.dump(["after_parse"], sys.stdout)
            return

    req = json.load(sys.stdin)

    wpm = req.get("config", {}).get("words_per_minute", 200)
    words = len(TAGS.sub(" ", req.get("content", "")).split())
    minutes = max(1, round(words / wpm))

    banner = f'<p class="reading-time">{minutes} min read</p>\n'
    json.dump({"content": banner + req.get("content", "")}, sys.stdout)


main()
```

Three things to notice:

- **`content` is HTML.** By `after_parse` the Markdown has already been
  converted, which is why the tag-stripping regex is there and why the banner
  is a `<p>` element rather than a Markdown paragraph.
- **`config` comes straight from `book.yaml`.** Numbers arrive as numbers;
  always supply a default, because the key may be absent.
- **The response replaces the chapter.** Returning `{"content": ""}` or `{}`
  means "leave it alone" — that is how a read-only plugin reports success.

## Step 4: Test The Plugin Without mdPress

The protocol is plain JSON on stdin, so a plugin is testable from a shell:

```console
$ echo '{"phase":"after_parse","content":"<p>hello world</p>","chapter_index":0,
  "chapter_file":"chapters/01-intro.md","config":{"words_per_minute":200},"metadata":{}}' \
  | ./plugins/reading-time
{"content": "<p class=\"reading-time\">1 min read</p>\n<p>hello world</p>"}
```

Do this before wiring the plugin into a build. A plugin failure is only ever a
warning in mdPress, so a broken plugin is easy to miss in build output.

## Step 5: Register It In `book.yaml`

```yaml
book:
  title: "Reading Time Demo"
  author: "You"

chapters:
  - title: "Introduction"
    file: chapters/01-intro.md

plugins:
  - name: reading-time
    path: ./plugins/reading-time
    config:
      words_per_minute: 200
```

`path` must be relative to `book.yaml` and must stay inside the project
directory. Absolute paths are rejected.

## Step 6: Verify With `doctor`

```console
$ mdpress doctor --verbose
  ✓ Plugin "reading-time" responds (version 1.0.0, 1 hook(s))
  ✓ All 1 plugin(s) are valid
```

`doctor` actually runs the discovery flags, so this line proves the handshake
works. Without `--verbose` you only get the summary. A plugin that answers
neither flag is reported as:

```
⚠ Plugin "reading-time" does not speak the mdpress plugin protocol
  (no valid --mdpress-info or --mdpress-hooks response)
```

## Step 7: Build

```console
$ mdpress build --format html
  ✅ Build completed (elapsed 123ms)
  ✓ Generated html  → .../Reading-Time-Demo.html

$ grep -o '<p class="reading-time">[^<]*</p>' Reading-Time-Demo.html
<p class="reading-time">1 min read</p>
```

`mdpress serve` runs the same hook on the initial render and again after every
file change, so the banner stays current while you edit.

## Debugging

### The build says nothing about my plugin

That is the normal case — a successful plugin is silent. To confirm it ran at
all, have it write a file:

```python
with open("/tmp/reading-time.log", "a") as fh:
    fh.write(req.get("chapter_file", "?") + "\n")
```

**Do not use stderr for this.** mdPress only surfaces stderr when the plugin
exits non-zero; on a successful run it is discarded.

### My plugin runs but nothing changes

Check three things in order:

1. **The phase.** `after_parse` is the only phase whose returned `content` is
   used. Anything returned from `before_build`, `before_render`, `after_render`
   or `after_build` is discarded. See [Lifecycle Hooks](./lifecycle-hooks.md).
2. **The field name.** The response key is `content`. `modified_content`,
   `output`, `body` and so on are ignored, and an ignored response reads as
   "no change".
3. **Stray stdout.** The response must be the *only* thing on stdout. A stray
   `print()` makes the response unparseable, which is reported as
   `failed to parse plugin response`.

### Only some of my plugins run

Plugin loading is all-or-nothing. One bad entry disables every plugin for that
build:

```
WARN plugin loading failed (continuing without plugins): failed to load
     plugin "missing": plugin executable not found at ".../does-not-exist"
```

Also check for `"stop": true` in an earlier plugin — it skips every plugin
declared after it for that phase.

### My plugin needs to know the output path

It cannot. `output_path` and `output_format` are present in the request but are
always empty. Pass what you need through your own `config:` block instead.

## Distributing A Plugin

There is no plugin registry and no install command. Distribution is "ship the
executable and the `book.yaml` snippet":

```
reading-time/
├── README.md          # what it does, plus the book.yaml block to copy
├── LICENSE
└── reading-time       # the executable
```

If your plugin is compiled, ship the source and a build line — users on a
different OS or architecture need to build it themselves:

```bash
go build -o plugins/word-count ./examples/plugins/word-count
```

Say plainly in your README that installing the plugin means letting mdPress
execute it during every build and serve.

## Next Steps

- [Plugin API Reference](./api.md) — the exact request/response schema and the
  trust model.
- [Lifecycle Hooks](./lifecycle-hooks.md) — what each phase can and cannot do.
- [`examples/plugins/word-count`](https://github.com/yeasy/mdpress/tree/main/examples/plugins/word-count)
  — the same protocol in Go.
