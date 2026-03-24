# Lifecycle Hooks in Detail

mdPress provides six lifecycle hooks that allow plugins to intercept and modify the build process. This guide explains when each hook fires, what data is available, and practical use cases for each.

## Hook Sequence

The build process follows this sequence, with hooks firing at key points:

```
START BUILD
  ↓
[before_build] - Plugins can initialize, validate setup
  ↓
PARSE CONTENT
  ↓
[after_parse] - Plugins can modify parsed content
  ↓
[before_render] - Plugins can prepare for rendering
  ↓
RENDER TO OUTPUT
  ↓
[after_render] - Plugins can post-process output
  ↓
[after_build] - Build complete, final tasks
  ↓
END BUILD
```

For serve mode (live development):

```
[before_serve] - Server initializing
  ↓
WATCH & REBUILD
  ├─ File changes trigger rebuild
  ├─ All build hooks execute
  └─ Repeat
  ↓
[after_serve] - Server stopping
  ↓
SHUTDOWN
```

## before_build

Fires once at the very start of the build process, before any content is processed.

### Use Cases

- Validate configuration and dependencies
- Initialize external resources (databases, APIs)
- Set up logging or monitoring
- Clean previous build artifacts
- Prepare build-wide state

### Data Available

The hook context includes:

```json
{
  "context": {
    "phase": "before_build",
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0"
    },
    "config": {
      "theme": "technical",
      "output_format": "html"
    },
    "output_path": "build/",
    "chapters": [
      "chapters/01-intro.md",
      "chapters/02-guide.md"
    ]
  }
}
```

### Example: Validate External API

```python
#!/usr/bin/env python3
import json
import sys
import requests

def validate_api(config):
    try:
        # Check if external API is reachable
        response = requests.get(config.get("api_url"), timeout=5)
        if response.status_code != 200:
            return {
                "status": "error",
                "action": "stop",
                "errors": ["API returned status code {}".format(response.status_code)]
            }

        return {
            "status": "success",
            "action": "continue",
            "data": {"api_validated": True}
        }
    except Exception as e:
        return {
            "status": "error",
            "action": "stop",
            "errors": ["API validation failed: {}".format(str(e))]
        }

# In main()
if request.get("action") == "execute_hook" and request.get("hook") == "before_build":
    response = validate_api(request.get("config", {}))
```

### Example: Clean Build Directory

```bash
#!/bin/bash

# Read JSON from stdin
read -r request

# Clean build directories
rm -rf build/html build/pdf build/epub

# Respond with success
echo '{"status": "success", "action": "continue"}'
```

## after_parse

Fires after each chapter's markdown is parsed into an AST (Abstract Syntax Tree), but before rendering.

### Use Cases

- Modify or transform parsed content
- Extract and process metadata
- Validate content structure
- Generate supplementary content
- Expand custom syntax

### Data Available

The hook context includes parsed content:

```json
{
  "context": {
    "phase": "parse",
    "content": "# Chapter Title\n\nContent here...",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html",
    "metadata": {
      "title": "Introduction",
      "author": "John Doe"
    }
  }
}
```

### Example: Expand Custom Macros

```python
#!/usr/bin/env python3
import json
import sys
import re
from datetime import datetime

def expand_macros(content):
    # Expand {{date}} macro
    content = re.sub(
        r'{{date}}',
        datetime.now().strftime('%Y-%m-%d'),
        content
    )

    # Expand {{updated}} macro to current timestamp
    content = re.sub(
        r'{{updated}}',
        datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        content
    )

    return content

request = json.load(sys.stdin)
if request.get("hook") == "after_parse":
    context = request.get("context", {})
    original = context.get("content", "")
    modified = expand_macros(original)

    response = {
        "status": "success",
        "action": "continue",
        "modified_content": modified,
        "content_type": "markdown"
    }

    json.dump(response, sys.stdout)
```

### Example: Extract and Validate Links

```python
#!/usr/bin/env python3
import json
import sys
import re

def extract_links(content):
    pattern = r'\[([^\]]+)\]\(([^\)]+)\)'
    matches = re.findall(pattern, content)
    return matches

request = json.load(sys.stdin)
if request.get("hook") == "after_parse":
    content = request.get("context", {}).get("content", "")
    links = extract_links(content)

    response = {
        "status": "success",
        "action": "continue",
        "metadata": {
            "links": links,
            "link_count": len(links)
        }
    }

    json.dump(response, sys.stdout)
```

## before_render

Fires right before a chapter is rendered from parsed content to the output format (HTML, PDF, etc.).

### Use Cases

- Prepare content for specific output formats
- Apply format-specific transformations
- Inject rendering hints or metadata
- Optimize content for output format

### Data Available

Same as `after_parse`, plus rendering context:

```json
{
  "context": {
    "phase": "render",
    "content": "# Chapter Title\n\nContent...",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html"
  }
}
```

### Example: Modify Content for PDF

```python
#!/usr/bin/env python3
import json
import sys

request = json.load(sys.stdin)

if request.get("hook") == "before_render":
    context = request.get("context", {})
    content = context.get("content", "")
    output_format = context.get("output_format", "")

    # For PDF output, add page breaks before H1 headings
    if output_format == "pdf":
        modified = content.replace(
            "# ",
            "\n---\n# "  # Page break before headings
        )
    else:
        modified = content

    response = {
        "status": "success",
        "action": "continue",
        "modified_content": modified,
        "content_type": "markdown"
    }

    json.dump(response, sys.stdout)
```

## after_render

Fires after a chapter has been rendered to the final output format (HTML, PDF, etc.).

### Use Cases

- Post-process generated output
- Add or modify output files
- Generate alternate formats
- Minify or optimize output
- Extract information from rendered content

### Data Available

The context includes the rendered output:

```json
{
  "context": {
    "phase": "render",
    "content": "<h1>Chapter Title</h1>\n<p>Content...</p>",
    "chapter_index": 0,
    "chapter_file": "chapters/01-intro.md",
    "output_path": "build/html/intro.html",
    "output_format": "html",
    "content_type": "html"
  }
}
```

### Example: Minify HTML

```python
#!/usr/bin/env python3
import json
import sys
import re

def minify_html(html):
    # Remove comments
    html = re.sub(r'<!--.*?-->', '', html, flags=re.DOTALL)

    # Remove unnecessary whitespace between tags
    html = re.sub(r'>\s+<', '><', html)

    # Remove leading/trailing whitespace in lines
    html = '\n'.join(line.strip() for line in html.split('\n'))

    return html

request = json.load(sys.stdin)

if request.get("hook") == "after_render":
    context = request.get("context", {})

    if context.get("output_format") == "html":
        original = context.get("content", "")
        minified = minify_html(original)

        response = {
            "status": "success",
            "action": "continue",
            "modified_content": minified,
            "content_type": "html",
            "metadata": {
                "original_size": len(original),
                "minified_size": len(minified),
                "compression_ratio": f"{len(minified) / len(original) * 100:.1f}%"
            }
        }
    else:
        response = {
            "status": "success",
            "action": "continue"
        }

    json.dump(response, sys.stdout)
```

### Example: Generate AMP Version

```python
#!/usr/bin/env python3
import json
import sys
import re
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_render":
    context = request.get("context", {})

    if context.get("output_format") == "html":
        html = context.get("content", "")
        output_path = context.get("output_path", "")

        # Convert to AMP-friendly HTML
        amp_html = html
        amp_html = re.sub(r'<img ', '<amp-img ', amp_html)
        amp_html = re.sub(r'<iframe ', '<amp-iframe ', amp_html)

        # Write AMP version
        amp_path = output_path.replace('.html', '.amp.html')
        os.makedirs(os.path.dirname(amp_path), exist_ok=True)
        with open(amp_path, 'w') as f:
            f.write(amp_html)

        response = {
            "status": "success",
            "action": "continue",
            "data": {
                "amp_generated": True,
                "amp_path": amp_path
            }
        }
    else:
        response = {"status": "success", "action": "continue"}

    json.dump(response, sys.stdout)
```

## after_build

Fires once after all chapters are processed and the entire build is complete.

### Use Cases

- Post-build validation and verification
- Generate build reports and statistics
- Upload documentation to servers
- Generate search indexes
- Create archives or backups
- Notify external services

### Data Available

The hook context includes build-wide information:

```json
{
  "context": {
    "phase": "after_build",
    "book": {
      "title": "My Documentation",
      "author": "Author Name",
      "version": "1.0.0"
    },
    "output_path": "build/",
    "output_format": "html",
    "chapters_processed": 5,
    "build_duration_ms": 2500
  }
}
```

### Example: Generate Build Report

```python
#!/usr/bin/env python3
import json
import sys
from datetime import datetime

request = json.load(sys.stdin)

if request.get("hook") == "after_build":
    context = request.get("context", {})

    report = {
        "timestamp": datetime.now().isoformat(),
        "book": context.get("book", {}).get("title"),
        "version": context.get("book", {}).get("version"),
        "format": context.get("output_format"),
        "chapters": context.get("chapters_processed", 0),
        "duration_ms": context.get("build_duration_ms", 0),
        "output_path": context.get("output_path")
    }

    with open("build/build-report.json", "w") as f:
        json.dump(report, f, indent=2)

    response = {
        "status": "success",
        "action": "continue",
        "data": {
            "report_generated": True,
            "report_path": "build/build-report.json"
        }
    }

    json.dump(response, sys.stdout)
```

### Example: Upload to Server

```python
#!/usr/bin/env python3
import json
import sys
import requests
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_build":
    context = request.get("context", {})
    config = request.get("config", {})
    output_path = context.get("output_path")

    upload_url = config.get("upload_url")
    api_key = config.get("api_key")

    # Create archive
    archive_path = "build/docs.tar.gz"
    os.system(f"tar -czf {archive_path} {output_path}")

    # Upload
    try:
        with open(archive_path, 'rb') as f:
            files = {'file': f}
            headers = {'Authorization': f'Bearer {api_key}'}
            response = requests.post(upload_url, files=files, headers=headers)

        if response.status_code == 200:
            return {
                "status": "success",
                "action": "continue",
                "data": {"uploaded": True}
            }
        else:
            return {
                "status": "error",
                "action": "continue",
                "warnings": [f"Upload failed: {response.status_code}"]
            }
    except Exception as e:
        return {
            "status": "warning",
            "action": "continue",
            "warnings": [f"Upload error: {str(e)}"]
        }
```

## before_serve

Fires once when the development server starts, before watching for file changes.

### Use Cases

- Initialize development-mode features
- Start local services or databases
- Set up live reload configuration
- Verify development environment

### Data Available

```json
{
  "context": {
    "phase": "serve",
    "server_host": "localhost",
    "server_port": 8000,
    "watch_paths": ["chapters/", "docs/"]
  }
}
```

### Example: Start Dev Services

```bash
#!/bin/bash

echo '{"status": "success", "action": "continue"}'

# Could start Docker containers, local services, etc.
```

## after_serve

Fires once when the development server stops.

### Use Cases

- Clean up development resources
- Stop local services
- Generate final development reports
- Archive development artifacts

### Data Available

```json
{
  "context": {
    "phase": "serve",
    "uptime_ms": 3600000,
    "files_watched": 45
  }
}
```

### Example: Clean Up Dev Services

```python
#!/usr/bin/env python3
import json
import sys
import os

request = json.load(sys.stdin)

if request.get("hook") == "after_serve":
    # Clean up temporary files
    if os.path.exists(".dev-cache"):
        os.system("rm -rf .dev-cache")

    response = {
        "status": "success",
        "action": "continue",
        "data": {"cleanup_complete": True}
    }

    json.dump(response, sys.stdout)
```

## Hook Filtering

Specify which hooks a plugin should receive in `book.yaml`:

```yaml
plugins:
  - name: build-plugin
    command: ./plugins/build.py
    hooks:
      - before_build
      - after_build
```

Only the specified hooks will be sent to the plugin, improving performance.

## Complete Lifecycle Example

Here's a plugin implementing all lifecycle hooks:

```python
#!/usr/bin/env python3
import json
import sys

class LifecyclePlugin:
    def __init__(self):
        self.stats = {
            "chapters_processed": 0,
            "links_found": 0,
            "errors": 0
        }

    def execute(self, hook, context):
        if hook == "before_build":
            print("Build starting...", file=sys.stderr)
            return {"status": "success", "action": "continue"}

        elif hook == "after_parse":
            self.stats["chapters_processed"] += 1
            return {"status": "success", "action": "continue"}

        elif hook == "before_render":
            return {"status": "success", "action": "continue"}

        elif hook == "after_render":
            return {"status": "success", "action": "continue"}

        elif hook == "after_build":
            print(f"Build complete: {self.stats}", file=sys.stderr)
            return {
                "status": "success",
                "action": "continue",
                "data": self.stats
            }

        elif hook == "before_serve":
            return {"status": "success", "action": "continue"}

        elif hook == "after_serve":
            return {"status": "success", "action": "continue"}

        return {"status": "success", "action": "continue"}

plugin = LifecyclePlugin()
request = json.load(sys.stdin)

if request.get("action") == "execute_hook":
    response = plugin.execute(
        request.get("hook"),
        request.get("context", {})
    )
    json.dump(response, sys.stdout)
```

See [Plugin API](./api.md) for complete API reference and [Building a Plugin](./building-a-plugin.md) for step-by-step tutorials.
