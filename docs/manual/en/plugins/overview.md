# Plugin System Overview

The mdPress plugin system allows you to extend and customize the build process without modifying core code. Plugins are external executables that hook into the documentation generation pipeline, enabling features like custom content processing, validation, and generation.

## What Plugins Can Do

Plugins can perform a wide variety of tasks throughout the build pipeline:

### Content Processing
- Transform content before rendering (e.g., custom syntax, macros)
- Modify markdown or HTML dynamically
- Inject content or metadata

### Validation
- Check links and references
- Validate code examples
- Verify documentation conventions
- Check metadata completeness

### Generation
- Generate tables of contents
- Create indexes and glossaries
- Generate diagrams from specifications
- Create sample outputs

### Analytics
- Count words and statistics
- Track changes and contributions
- Monitor documentation quality

### Integration
- Publish to external services
- Submit to search engines
- Generate alternate formats
- Post-process output

### Custom Features
- Add domain-specific functionality
- Implement custom build steps
- Extend the templating system

## When to Use Plugins

Use plugins when you need to:

1. **Extend core functionality** without modifying mdPress itself
2. **Integrate with external tools** (linters, generators, APIs)
3. **Automate repetitive tasks** in the build process
4. **Implement custom business logic** specific to your documentation
5. **Share reusable build extensions** across teams or projects

Don't use plugins for:
- Simple styling changes (use [Custom CSS](../themes/custom-css.md))
- Theme customization (use [Built-in Themes](../themes/builtin-themes.md))
- Basic content organization (use markdown and configuration)

## How Plugins Work

mdPress uses an external executable model where plugins are standalone programs that communicate via JSON over stdin/stdout:

```
┌─────────────┐
│  mdPress    │
│  (Go)       │
└──────┬──────┘
       │ Sends JSON
       ▼
┌──────────────────┐
│  Plugin (Any     │
│  Language)       │
│  (Python/Node.js)│
└──────┬───────────┘
       │ Responds with JSON
       ▼
┌─────────────┐
│  mdPress    │
│  continues  │
└─────────────┘
```

### Key Advantages

- **Language Agnostic**: Write plugins in Python, Node.js, Go, Ruby, or any language
- **Secure**: Plugins run as separate processes with no direct access to mdPress internals
- **Loosely Coupled**: Plugins don't depend on mdPress API versions
- **Easy to Test**: Plugins are independent programs you can test in isolation
- **Debuggable**: Standard JSON protocol makes debugging straightforward

## Loading Plugins

Plugins are defined in `book.yaml` under the `plugins` section:

```yaml
plugins:
  - name: word-counter
    command: ./plugins/word-counter.py
    config:
      min_words: 100

  - name: link-checker
    command: ./plugins/link-checker
    enabled: true

  - name: custom-syntax
    command: python3
    args:
      - ./plugins/syntax.py
```

### Plugin Configuration Options

- `name` (required) - Unique identifier for the plugin
- `command` (required) - Path to the executable
- `args` (optional) - Arguments passed to the command
- `config` (optional) - Custom configuration passed to the plugin
- `enabled` (optional) - Enable/disable the plugin (default: true)
- `hooks` (optional) - Specific hooks to run (default: all)

## Execution Order

Plugins execute in the order they're defined in `book.yaml`. The build process follows this sequence:

```
1. mdPress starts build
2. before_build hooks run (in order)
3. Parse content
4. after_parse hooks run (in order)
5. before_render hooks run (in order)
6. Render to output format
7. after_render hooks run (in order)
8. after_build hooks run (in order)
9. Build complete
```

For serve mode:
```
1. before_serve hooks run
2. Server starts (watch for file changes)
3. On file change: rebuild triggers hooks
4. after_serve hooks run (when server stops)
```

### Controlling Execution Order

If plugin execution order matters, organize them in `book.yaml` appropriately:

```yaml
plugins:
  # This runs first
  - name: preprocess
    command: ./plugins/preprocess.py

  # This runs after preprocess
  - name: validate
    command: ./plugins/validate.py

  # This runs last
  - name: postprocess
    command: ./plugins/postprocess.py
```

## Plugin Lifecycle

Each plugin goes through these stages:

1. **Load** - mdPress reads the plugin configuration
2. **Initialize** - Plugin executable starts, reads JSON, initializes
3. **Execute Hooks** - Plugin processes hooks as requested
4. **Cleanup** - Plugin performs cleanup, sends final response

## Example: Simple Word Counter

Here's a minimal word-counting plugin to demonstrate the model:

```python
#!/usr/bin/env python3
import json
import sys

def count_words(text):
    return len(text.split())

def main():
    # Read plugin request from mdPress
    request = json.load(sys.stdin)

    # Process based on hook
    response = {
        "name": "word-counter",
        "version": "1.0",
        "status": "success",
        "data": {
            "word_count": count_words(request['content']),
            "characters": len(request['content'])
        }
    }

    # Send response back to mdPress
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

## Next Steps

- [Plugin API Reference](./api.md) - Complete API specification
- [Lifecycle Hooks](./lifecycle-hooks.md) - Detailed hook documentation
- [Building a Plugin](./building-a-plugin.md) - Step-by-step tutorial
