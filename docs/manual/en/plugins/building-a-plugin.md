# Building a Plugin: Step-by-Step Tutorial

This tutorial walks through creating a complete mdPress plugin from scratch. We'll build a word-count plugin that analyzes documentation statistics and generates a report.

## Project Setup

### Create Plugin Directory

```bash
mkdir -p my-project/plugins
cd my-project
```

### Directory Structure

```
my-project/
├── book.yaml
├── chapters/
│   └── 01-intro.md
└── plugins/
    ├── word-count.py
    └── word-count.json
```

## Step 1: Define Plugin Metadata

Create a JSON file describing your plugin:

**`plugins/word-count.json`:**

```json
{
  "name": "word-count",
  "version": "1.0.0",
  "description": "Analyzes and reports word count statistics",
  "author": "Your Name",
  "license": "MIT",
  "hooks": ["after_parse", "after_build"],
  "configuration": {
    "min_words": 100,
    "warn_long_chapters": true,
    "warn_threshold": 5000
  }
}
```

## Step 2: Create the Plugin Script

Create the main plugin executable. We'll use Python, but you can use any language.

**`plugins/word-count.py`:**

```python
#!/usr/bin/env python3
"""
Word Count Plugin for mdPress
Analyzes word count and generates statistics report
"""

import json
import sys
import os
from pathlib import Path

class WordCountPlugin:
    """Tracks and reports documentation statistics."""

    def __init__(self):
        self.name = "word-count"
        self.version = "1.0.0"
        self.config = {}
        self.stats = {
            "total_words": 0,
            "total_characters": 0,
            "chapter_stats": [],
            "long_chapters": []
        }

    def init(self, config):
        """Initialize the plugin with configuration."""
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Analyzes and reports word count statistics",
            "status": "success",
            "capabilities": ["after_parse", "after_build"]
        }

    def count_words(self, content):
        """Count words in content."""
        # Remove markdown syntax for more accurate count
        words = content.split()
        return len(words)

    def count_characters(self, content):
        """Count characters (excluding whitespace)."""
        return len(content.replace(" ", "").replace("\n", ""))

    def execute_after_parse(self, context):
        """Process content after parsing."""
        content = context.get("content", "")
        chapter_file = context.get("chapter_file", "")
        chapter_index = context.get("chapter_index", 0)

        # Count words and characters
        word_count = self.count_words(content)
        char_count = self.count_characters(content)

        # Update totals
        self.stats["total_words"] += word_count
        self.stats["total_characters"] += char_count

        # Track per-chapter stats
        chapter_stat = {
            "file": chapter_file,
            "index": chapter_index,
            "words": word_count,
            "characters": char_count
        }
        self.stats["chapter_stats"].append(chapter_stat)

        # Check for long chapters
        threshold = self.config.get("warn_threshold", 5000)
        if self.config.get("warn_long_chapters") and word_count > threshold:
            self.stats["long_chapters"].append({
                "file": chapter_file,
                "words": word_count
            })

        return {
            "status": "success",
            "action": "continue",
            "metadata": {
                "word_count": word_count,
                "character_count": char_count
            }
        }

    def execute_after_build(self, context):
        """Generate report after build completes."""
        output_path = context.get("output_path", "build")

        # Calculate averages
        num_chapters = len(self.stats["chapter_stats"])
        avg_words = (self.stats["total_words"] // num_chapters
                    if num_chapters > 0 else 0)

        report = {
            "timestamp": context.get("timestamp", ""),
            "book": context.get("book", {}).get("title", "Unknown"),
            "statistics": {
                "total_words": self.stats["total_words"],
                "total_characters": self.stats["total_characters"],
                "chapters": num_chapters,
                "average_words_per_chapter": avg_words
            },
            "chapters": self.stats["chapter_stats"]
        }

        # Add warnings for long chapters
        if self.stats["long_chapters"]:
            report["warnings"] = {
                "long_chapters": self.stats["long_chapters"],
                "message": f"Found {len(self.stats['long_chapters'])} "
                          f"chapters exceeding {self.config.get('warn_threshold')} words"
            }

        # Write report to file
        report_path = os.path.join(output_path, "word-count-report.json")
        os.makedirs(output_path, exist_ok=True)

        with open(report_path, "w") as f:
            json.dump(report, f, indent=2)

        return {
            "status": "success",
            "action": "continue",
            "data": {
                "report_generated": True,
                "report_path": report_path,
                "total_words": self.stats["total_words"],
                "total_chapters": num_chapters
            }
        }

    def execute_hook(self, hook, context):
        """Route to appropriate hook handler."""
        if hook == "after_parse":
            return self.execute_after_parse(context)
        elif hook == "after_build":
            return self.execute_after_build(context)
        else:
            return {
                "status": "success",
                "action": "continue"
            }

    def cleanup(self):
        """Cleanup on shutdown."""
        return {
            "status": "success"
        }

    def process_request(self, request):
        """Process incoming request from mdPress."""
        action = request.get("action")

        try:
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
        except Exception as e:
            return {
                "status": "error",
                "action": "stop",
                "errors": [f"Plugin error: {str(e)}"]
            }

def main():
    """Entry point."""
    plugin = WordCountPlugin()

    # Read request from mdPress
    request = json.load(sys.stdin)

    # Process and respond
    response = plugin.process_request(request)
    json.dump(response, sys.stdout)

if __name__ == "__main__":
    main()
```

## Step 3: Make Script Executable

```bash
chmod +x plugins/word-count.py
```

## Step 4: Register Plugin in book.yaml

Add the plugin to your `book.yaml`:

```yaml
book:
  title: "My Documentation"
  author: "Your Name"

chapters:
  - chapters/01-intro.md

plugins:
  - name: word-count
    command: ./plugins/word-count.py
    config:
      min_words: 100
      warn_long_chapters: true
      warn_threshold: 5000
```

## Step 5: Create Sample Content

Create a test chapter:

**`chapters/01-intro.md`:**

```markdown
# Introduction

This is a sample chapter for testing the word-count plugin.
It contains enough text to demonstrate the plugin functionality.

## Features

The word-count plugin analyzes your documentation and provides:

- Total word count across all chapters
- Per-chapter statistics
- Character count
- Warnings for unusually long chapters
- JSON report output

This helps you understand the scope and complexity of your documentation.
```

## Step 6: Test the Plugin

Build your documentation:

```bash
mdpress build
```

You should see the plugin execute and generate a report. Check the output:

```bash
cat build/word-count-report.json
```

Output:

```json
{
  "timestamp": "2026-03-23",
  "book": "My Documentation",
  "statistics": {
    "total_words": 87,
    "total_characters": 652,
    "chapters": 1,
    "average_words_per_chapter": 87
  },
  "chapters": [
    {
      "file": "chapters/01-intro.md",
      "index": 0,
      "words": 87,
      "characters": 652
    }
  ],
  "warnings": []
}
```

## Step 7: Add More Features

### Log Output During Build

Enhance the plugin to provide console feedback:

```python
def execute_after_parse(self, context):
    content = context.get("content", "")
    chapter_file = context.get("chapter_file", "")
    word_count = self.count_words(content)

    # Log progress to stderr (visible during build)
    print(f"[word-count] Processing {chapter_file}: {word_count} words",
          file=sys.stderr)

    # ... rest of implementation
```

### Add Reading Time Estimation

```python
def estimate_reading_time(self, word_count, words_per_minute=200):
    """Estimate reading time in minutes."""
    return max(1, round(word_count / words_per_minute))

def execute_after_parse(self, context):
    # ... existing code ...

    reading_time = self.estimate_reading_time(word_count)

    return {
        "status": "success",
        "action": "continue",
        "metadata": {
            "word_count": word_count,
            "reading_time_minutes": reading_time
        }
    }
```

## Step 8: Debugging Tips

### Enable Verbose Logging

Modify your plugin to output debug information:

```python
import sys

def log_debug(msg):
    """Log debug message to stderr."""
    print(f"[word-count DEBUG] {msg}", file=sys.stderr)

# In hooks:
log_debug(f"Processing: {context.get('chapter_file')}")
log_debug(f"Word count: {word_count}")
```

Run with stderr visible:

```bash
mdpress build 2>&1 | tee build.log
```

### Test Plugin Independently

Create a test script to verify plugin behavior:

**`test-plugin.py`:**

```python
#!/usr/bin/env python3
import json
import subprocess

# Create test request
request = {
    "action": "init",
    "config": {
        "warn_threshold": 5000
    }
}

# Run plugin
result = subprocess.run(
    ["./plugins/word-count.py"],
    input=json.dumps(request),
    capture_output=True,
    text=True
)

# Check response
response = json.loads(result.stdout)
print("Response:", json.dumps(response, indent=2))

if result.stderr:
    print("Errors:", result.stderr)
```

Run the test:

```bash
python3 test-plugin.py
```

### Print Request/Response

In your plugin, save request/response for inspection:

```python
def process_request(self, request):
    # Save request for debugging
    with open("/tmp/mdpress-plugin-request.json", "w") as f:
        json.dump(request, f, indent=2)

    response = self._process(request)

    # Save response
    with open("/tmp/mdpress-plugin-response.json", "w") as f:
        json.dump(response, f, indent=2)

    return response
```

Check the files:

```bash
cat /tmp/mdpress-plugin-request.json
cat /tmp/mdpress-plugin-response.json
```

## Step 9: Handling Errors

Make your plugin robust:

```python
def execute_after_parse(self, context):
    try:
        content = context.get("content", "")
        if not content:
            return {
                "status": "warning",
                "action": "continue",
                "warnings": ["Empty content in chapter"]
            }

        word_count = self.count_words(content)

        if word_count == 0:
            return {
                "status": "warning",
                "action": "continue",
                "warnings": ["No words found in chapter"]
            }

        # ... normal processing ...

    except Exception as e:
        return {
            "status": "error",
            "action": "continue",  # Don't stop build
            "errors": [f"Failed to process chapter: {str(e)}"]
        }
```

## Step 10: Distribution

### Package Your Plugin

Create a standard structure for sharing:

```
word-count-plugin/
├── README.md
├── LICENSE
├── word-count.py
├── word-count.json
├── examples/
│   └── book.yaml
└── tests/
    └── test_word_count.py
```

### Document Installation

**`README.md`:**

```markdown
# Word Count Plugin

Analyzes word count statistics for mdPress documentation.

## Installation

1. Copy `word-count.py` to your `plugins/` directory
2. Add to `book.yaml`:

\`\`\`yaml
plugins:
  - name: word-count
    command: ./plugins/word-count.py
    config:
      warn_threshold: 5000
\`\`\`

3. Run `mdpress build`

## Output

Generates `word-count-report.json` in build output directory.
```

## Complete Working Example

Here's the full, tested implementation:

**`plugins/word-count.py`** (complete):

```python
#!/usr/bin/env python3
"""Word Count Plugin for mdPress"""

import json
import sys
import os

class WordCountPlugin:
    def __init__(self):
        self.name = "word-count"
        self.version = "1.0.0"
        self.config = {}
        self.stats = {
            "total_words": 0,
            "chapter_stats": []
        }

    def init(self, config):
        self.config = config
        return {
            "name": self.name,
            "version": self.version,
            "description": "Analyzes word count statistics",
            "status": "success"
        }

    def count_words(self, content):
        return len(content.split())

    def execute_hook(self, hook, context):
        if hook == "after_parse":
            content = context.get("content", "")
            word_count = self.count_words(content)
            self.stats["total_words"] += word_count
            self.stats["chapter_stats"].append({
                "file": context.get("chapter_file"),
                "words": word_count
            })
            return {
                "status": "success",
                "action": "continue",
                "metadata": {"word_count": word_count}
            }

        elif hook == "after_build":
            report_path = os.path.join(
                context.get("output_path", "build"),
                "word-count-report.json"
            )
            os.makedirs(os.path.dirname(report_path) or ".", exist_ok=True)
            with open(report_path, "w") as f:
                json.dump(self.stats, f, indent=2)
            return {
                "status": "success",
                "action": "continue",
                "data": {"report_path": report_path}
            }

        return {"status": "success", "action": "continue"}

    def process_request(self, request):
        action = request.get("action")
        if action == "init":
            return self.init(request.get("config", {}))
        elif action == "execute_hook":
            return self.execute_hook(
                request.get("hook"),
                request.get("context", {})
            )
        return {"status": "success"}

if __name__ == "__main__":
    plugin = WordCountPlugin()
    request = json.load(sys.stdin)
    json.dump(plugin.process_request(request), sys.stdout)
```

See [Plugin API Reference](./api.md) and [Lifecycle Hooks](./lifecycle-hooks.md) for more details on plugin development.
