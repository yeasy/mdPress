# Plugin API Reference

This document specifies the complete Plugin API, including the interface plugins must implement and the data structures exchanged between mdPress and plugins.

## Plugin Interface

Every mdPress plugin must implement these core methods:

### Init

Initialize the plugin. Called once when mdPress starts.

**Request:**
```json
{
  "action": "init",
  "config": {
    "min_words": 100,
    "custom_option": "value"
  }
}
```

**Response:**
```json
{
  "name": "word-counter",
  "version": "1.0.0",
  "description": "Counts words in documentation",
  "status": "success",
  "capabilities": ["after_parse", "after_build"],
  "errors": []
}
```

### Execute

Execute hooks during the build process. Called multiple times with different hook contexts.

**Request:**
```json
{
  "action": "execute_hook",
  "hook": "after_parse",
  "context": {
    "content": "# Chapter Title\n\nSome content...",
    "phase": "parse",
    "chapter_index": 0,
    "chapter_file": "chapters/intro.md",
    "output_path": "build/intro.html",
    "output_format": "html"
  }
}
```

**Response:**
```json
{
  "status": "success",
  "action": "continue",
  "modified_content": "# Chapter Title\n\nModified content...",
  "metadata": {
    "word_count": 42
  },
  "errors": []
}
```

### Cleanup

Clean up resources. Called when mdPress shuts down.

**Request:**
```json
{
  "action": "cleanup"
}
```

**Response:**
```json
{
  "status": "success",
  "errors": []
}
```

## Plugin Metadata

Plugins must provide metadata describing themselves:

### Required Fields

```json
{
  "name": "unique-plugin-name",
  "version": "1.0.0",
  "description": "Brief description of what the plugin does"
}
```

### Optional Fields

```json
{
  "name": "plugin-name",
  "version": "1.0.0",
  "description": "What this plugin does",
  "author": "Your Name",
  "license": "MIT",
  "homepage": "https://github.com/...",
  "documentation": "https://docs.example.com/plugin"
}
```

## Hook Context

The HookContext contains all information about the current processing stage:

### Context Fields

```typescript
{
  "context": {
    // Current content being processed
    "content": "# Chapter Title\n\nContent here",

    // Metadata about the document
    "metadata": {
      "title": "Chapter Title",
      "author": "Author Name",
      "date": "2026-03-23",
      "custom": {}
    },

    // Build phase identifier
    "phase": "parse",  // or "render", "serve", etc.

    // Index of current chapter (0-based)
    "chapter_index": 0,

    // Source markdown file path
    "chapter_file": "chapters/01-intro.md",

    // Output file path
    "output_path": "build/html/intro.html",

    // Target output format
    "output_format": "html",  // or "pdf", "epub"

    // Book configuration
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0",
      "description": "Description"
    },

    // Build configuration
    "config": {
      "theme": "technical",
      "language": "en"
    }
  }
}
```

### Available Metadata

The `metadata` object contains:

```json
{
  "metadata": {
    "title": "Chapter Title",
    "author": "Original Author",
    "date": "2026-03-23",
    "tags": ["tag1", "tag2"],
    "custom": {
      "custom_field": "custom_value"
    }
  }
}
```

## Hook Response Actions

After processing, plugins can specify what mdPress should do:

### Continue (Default)

Continue to the next plugin or next phase:

```json
{
  "status": "success",
  "action": "continue",
  "modified_content": "Updated content"
}
```

### Stop

Stop processing and fail the build:

```json
{
  "status": "error",
  "action": "stop",
  "errors": ["Critical error message"]
}
```

### Skip

Skip remaining plugins for this hook:

```json
{
  "status": "success",
  "action": "skip",
  "reason": "Conditions not met for further processing"
}
```

## Return Values

All responses must include:

### Status

```json
{
  "status": "success"  // or "error" or "warning"
}
```

### Errors Array

```json
{
  "errors": [
    "Error message 1",
    "Error message 2"
  ]
}
```

### Modified Content

When processing content, return modifications:

```json
{
  "modified_content": "Updated markdown or HTML",
  "content_type": "markdown"  // or "html"
}
```

### Custom Metadata

Add or modify metadata:

```json
{
  "metadata": {
    "word_count": 1250,
    "reading_time": "5 min",
    "custom_field": "value"
  }
}
```

### Data Output

Return arbitrary data for logging or external use:

```json
{
  "data": {
    "processed_items": 5,
    "warnings": 2,
    "custom_metric": 42
  }
}
```

## Complete Response Example

Here's a complete response showing all possible fields:

```json
{
  "name": "example-plugin",
  "status": "success",
  "action": "continue",
  "modified_content": "# Updated Content\n\nModified by plugin",
  "content_type": "markdown",
  "metadata": {
    "word_count": 42,
    "processed": true,
    "plugin_version": "1.0.0"
  },
  "data": {
    "links_checked": 15,
    "broken_links": 0,
    "processing_time_ms": 125
  },
  "errors": [],
  "warnings": []
}
```

## Error Handling

### Validation Errors

Return `status: error` for validation issues:

```json
{
  "status": "error",
  "action": "stop",
  "errors": [
    "Line 5: Invalid syntax in code block",
    "Line 12: Missing required metadata field"
  ]
}
```

### Warnings

Return `status: warning` for non-critical issues:

```json
{
  "status": "warning",
  "action": "continue",
  "warnings": [
    "External link timeout (retrying): https://example.com",
    "Missing author name in chapter metadata"
  ],
  "modified_content": "..."
}
```

### Exceptions

On unexpected errors, respond with:

```json
{
  "status": "error",
  "action": "stop",
  "errors": [
    "Unexpected error: Database connection failed",
    "Stack trace details (optional)"
  ]
}
```

## Protocol Details

### JSON Encoding

All communication uses UTF-8 encoded JSON:

```python
import json
import sys

# Read from mdPress
request = json.load(sys.stdin)

# Process request...

# Send response
json.dump(response, sys.stdout)
```

### Single Request/Response

Each hook execution follows a single request/response pattern:

1. mdPress sends JSON on stdin
2. Plugin reads and processes
3. Plugin sends response on stdout
4. Connection closes

### Timeout Handling

mdPress waits for plugin response with a timeout (default: 30 seconds). Long-running operations should:

1. Send intermediate responses
2. Implement timeouts gracefully
3. Report progress in metadata

Example long-running operation:

```json
{
  "status": "success",
  "action": "continue",
  "metadata": {
    "progress": "Processing 42 of 100 items",
    "estimated_remaining": "2 seconds"
  }
}
```

## Configuration Schema

Plugins define their configuration schema in `init` response:

```json
{
  "name": "advanced-plugin",
  "configuration_schema": {
    "properties": {
      "enabled": {
        "type": "boolean",
        "default": true,
        "description": "Enable this plugin"
      },
      "output_format": {
        "type": "string",
        "default": "json",
        "enum": ["json", "csv", "xml"],
        "description": "Output format"
      },
      "max_items": {
        "type": "integer",
        "default": 100,
        "description": "Maximum items to process"
      }
    },
    "required": ["enabled"]
  }
}
```

Users provide configuration in `book.yaml`:

```yaml
plugins:
  - name: advanced-plugin
    command: ./plugins/advanced.py
    config:
      enabled: true
      output_format: csv
      max_items: 500
```

## Full API Example

Here's a complete example plugin implementing the full API:

```python
#!/usr/bin/env python3
import json
import sys

class DocumentPlugin:
    def __init__(self):
        self.name = "full-example"
        self.version = "1.0.0"
        self.config = {}

    def init(self, config):
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Complete API example plugin",
            "status": "success"
        }

    def execute_hook(self, hook, context):
        if hook == "after_parse":
            return self.handle_after_parse(context)
        elif hook == "after_render":
            return self.handle_after_render(context)
        else:
            return {
                "status": "success",
                "action": "continue"
            }

    def handle_after_parse(self, context):
        content = context.get("content", "")
        word_count = len(content.split())

        return {
            "status": "success",
            "action": "continue",
            "metadata": {
                "word_count": word_count
            },
            "data": {
                "chapter": context.get("chapter_file"),
                "processed": True
            }
        }

    def handle_after_render(self, context):
        return {
            "status": "success",
            "action": "continue"
        }

    def cleanup(self):
        return {
            "status": "success"
        }

    def process_request(self, request):
        action = request.get("action")

        if action == "init":
            return self.init(request.get("config", {}))
        elif action == "execute_hook":
            return self.execute_hook(
                request.get("hook"),
                request.get("context", {})
            )
        elif action == "cleanup":
            return self.cleanup()
        else:
            return {
                "status": "error",
                "errors": [f"Unknown action: {action}"]
            }

def main():
    plugin = DocumentPlugin()
    request = json.load(sys.stdin)
    response = plugin.process_request(request)
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

See [Lifecycle Hooks](./lifecycle-hooks.md) for detailed hook documentation and [Building a Plugin](./building-a-plugin.md) for tutorials.
