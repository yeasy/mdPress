# Using mdpress doctor

The `mdpress doctor` command checks your system environment and project configuration for potential issues. Use it to diagnose build problems before they occur.

## Quick Start

```bash
mdpress doctor

# Example output:
# Checking environment...
# ✓ Platform: Linux
# ✓ Go: 1.25.0
# ✓ Chrome/Chromium: /usr/bin/chromium
# ✓ CJK fonts: Noto Sans CJK SC
# ✓ PlantUML: not installed (optional)
# ✓ Cache directory: ~/.cache/mdpress/
# ✗ book.yaml: missing

# Doctor completed: 1 warning
```

## What Doctor Checks

### Environment Detection

**Platform and OS:**
```
✓ Platform: Linux (ubuntu-latest)
✓ OS version: 6.8.0
```

Identifies your operating system for system-specific guidance.

**Go Installation:**
```
✓ Go: 1.25.0 installed
✓ go binary: /usr/local/go/bin/go
```

Required for mdPress to function. If missing, doctor suggests installation.

**Chrome/Chromium:**
```
✓ Chrome: /usr/bin/chromium (version 120.0)
```

Required for PDF rendering. Doctor auto-detects or uses `MDPRESS_CHROME_PATH` environment variable.

**CJK Fonts:**
```
✓ CJK fonts detected:
  - Noto Sans CJK SC (Chinese Simplified)
  - Noto Sans CJK JP (Japanese)
```

Optional but recommended for books with Chinese, Japanese, or Korean text.

**PlantUML:**
```
✗ PlantUML: not installed
  → Optional for diagram rendering
  → Install from: http://plantuml.com/download
```

Optional dependency for diagram support.

### Project Configuration

**Config File:**
```
✓ book.yaml found
✓ Config valid (12 chapters, 24 images)
```

Checks for `book.yaml` and validates basic syntax.

**Chapter Files:**
```
✓ All 12 chapter files exist
✓ Total chapters: 12 (4000 lines)
```

Verifies all chapters referenced in config exist on disk.

**Image Assets:**
```
✓ 24 images found
  - PNG: 18 (5.2 MB total)
  - SVG: 4 (150 KB total)
  - JPEG: 2 (800 KB total)
```

Lists all embedded images and total size.

**Links Validation:**
```
✓ Internal links checked
✓ No broken cross-references
```

Validates relative links between chapters.

### Cache and Build Readiness

**Cache Directory:**
```
✓ Cache: ~/.cache/mdpress/ (2.4 MB)
  - 12 cached chapters
  - Last updated: 2 hours ago
```

Shows cache location, size, and freshness.

**Build Readiness:**
```
✓ Ready to build
  Recommended format: pdf (Chrome available)
```

Final verdict on whether system is ready to build.

## Running Doctor

### Basic Check

```bash
mdpress doctor
```

Checks current directory for configuration and environment.

### Check Specific Directory

```bash
mdpress doctor /path/to/book
```

Diagnoses a project in a different directory.

### Generate Report

```bash
# JSON format (for parsing)
mdpress doctor --report json > report.json

# Markdown format (for documentation)
mdpress doctor --report markdown > report.md

# Plain text (default)
mdpress doctor
```

### Example: Generate Markdown Report

```bash
mdpress doctor --report markdown > DOCTOR_REPORT.md
```

Report includes:
- System information
- Installed tools and versions
- Configuration validation results
- Recommendations for fixes

## Interpreting Doctor Output

### Green Checkmarks (✓)

Everything is working correctly:

```
✓ Platform: Linux
✓ Go: 1.25.0
✓ Chrome: /usr/bin/chromium
```

No action needed.

### Yellow Warnings (!)

Optional dependencies missing, but not critical:

```
! PlantUML: not installed
  → This is optional
  → Diagram rendering will be skipped
  → To enable: install PlantUML from http://plantuml.com
```

PlantUML is optional. Books build fine without it, but diagrams won't render.

### Red Errors (✗)

Critical issues preventing builds:

```
✗ Chrome: not found
  → Required for PDF rendering
  → To fix: Install Chrome or set MDPRESS_CHROME_PATH
  → Install: https://www.google.com/chrome/
```

Errors must be fixed before building.

## Common Doctor Findings and Fixes

### Chrome/Chromium Not Found

**Finding:**
```
✗ Chrome: not found
  → Required for PDF rendering with --format pdf
```

**Fix:**

```bash
# Install Chrome/Chromium
# Linux
sudo apt-get install chromium-browser

# macOS
brew install chromium

# Windows
# Download from https://www.google.com/chrome/

# Then set path (if not auto-detected)
export MDPRESS_CHROME_PATH=/path/to/chrome
mdpress build --format pdf
```

Or use Typst as alternative:

```bash
mdpress build --format typst
```

### Missing CJK Fonts

**Finding:**
```
! CJK fonts: not detected
  → Books with Chinese, Japanese, or Korean text need CJK fonts
  → Optional if your book is English-only
```

**Fix (if needed):**

```bash
# Ubuntu/Debian
sudo apt-get install fonts-noto-cjk

# macOS
brew install font-noto-sans-cjk

# Alpine
apk add font-noto-cjk
```

Then rebuild:

```bash
mdpress build --format pdf --no-cache
```

### Missing book.yaml

**Finding:**
```
✗ book.yaml: not found
  → Configuration file required
  → Create one with: mdpress init
```

**Fix:**

```bash
# Generate template
mdpress init

# Or create manually
cat > book.yaml << 'EOF'
book:
  title: "My Book"
  author: "Your Name"

chapters:
  - title: "Chapter 1"
    file: "ch01.md"
EOF

# Then build
mdpress build
```

### Broken Chapter References

**Finding:**
```
✗ Config validation failed
  Chapter 1 references missing file: chapters/ch01.md
```

**Fix:**

```bash
# Check what files exist
ls -la

# Update book.yaml with correct paths
vim book.yaml

# Verify fix
mdpress validate
```

### Broken Image References

**Finding:**
```
! Image not found: ../assets/diagram.png
  → Referenced in: chapters/ch02.md
```

**Fix:**

```bash
# Check if image exists
ls -la assets/

# If missing, add image:
cp /path/to/diagram.png assets/

# If path wrong, update Markdown:
# Change: ![Diagram](diagram.png)
# To: ![Diagram](../assets/diagram.png)

# Verify
mdpress validate
```

## Using Doctor in CI/CD

### GitHub Actions

```yaml
- name: Run doctor check
  run: mdpress doctor

- name: Generate doctor report
  if: always()
  run: mdpress doctor --report markdown > doctor-report.md

- name: Upload report as artifact
  uses: actions/upload-artifact@v3
  with:
    name: doctor-report
    path: doctor-report.md
```

### GitLab CI

```yaml
doctor:
  stage: validate
  script:
    - mdpress doctor
    - mdpress doctor --report markdown > doctor-report.md
  artifacts:
    paths:
      - doctor-report.md
    expire_in: 30 days
  allow_failure: true  # Warnings don't fail build
```

### Pre-Build Validation

```bash
#!/bin/bash
set -e

echo "Running doctor check..."
mdpress doctor

# If doctor fails, stop build
if [ $? -ne 0 ]; then
  echo "Doctor check failed. Please fix issues above."
  exit 1
fi

echo "Doctor check passed. Building..."
mdpress build --format pdf
```

## Doctor Report Fields

When using `--report json`:

```json
{
  "platform": "Linux",
  "osVersion": "6.8.0",
  "go": {
    "installed": true,
    "version": "1.25.0",
    "path": "/usr/local/go/bin/go"
  },
  "chrome": {
    "found": true,
    "path": "/usr/bin/chromium",
    "version": "120.0.0.0"
  },
  "cjkFonts": ["Noto Sans CJK SC", "Noto Sans CJK JP"],
  "plantUml": {
    "installed": false,
    "recommended": false
  },
  "project": {
    "configValid": true,
    "chaptersCount": 12,
    "imagesCount": 24,
    "totalImageSize": "6.2 MB"
  },
  "recommendations": [
    "Install PlantUML for diagram support (optional)"
  ]
}
```

Parse programmatically:

```bash
# Get Chrome version
mdpress doctor --report json | jq '.chrome.version'

# Check if project ready
mdpress doctor --report json | jq '.project.configValid'

# List recommendations
mdpress doctor --report json | jq '.recommendations[]'
```

## Continuous Monitoring

### Health Check Script

```bash
#!/bin/bash

echo "=== mdPress Health Check ==="
echo ""

echo "1. Running doctor..."
mdpress doctor --report json > /tmp/doctor.json

if ! jq empty /tmp/doctor.json 2>/dev/null; then
  echo "✗ Doctor check failed"
  exit 1
fi

echo ""
echo "2. Validating configuration..."
if ! mdpress validate; then
  echo "✗ Validation failed"
  exit 1
fi

echo ""
echo "3. Test build (HTML format, fastest)..."
if ! mdpress build --format html --output /tmp/test.html; then
  echo "✗ Build failed"
  exit 1
fi

echo ""
echo "✓ All checks passed!"
echo "Ready to build PDF: mdpress build --format pdf"
```

Run regularly:

```bash
# Weekly health check
0 0 * * 0 /path/to/health-check.sh
```

## Doctor Limitations

Doctor checks the **environment** and **configuration**, but doesn't:

- Validate Markdown syntax (use `mdpress validate` for that)
- Check for grammar or spelling errors
- Verify image quality or resolution
- Detect performance issues during build

For comprehensive validation:

```bash
# Complete validation
mdpress validate      # Config and structure
mdpress doctor        # Environment readiness
mdpress build --format html  # Try actual build
```

## Next Steps After Doctor

If doctor reports all green:

```bash
# Try building
mdpress serve       # Preview with live reload
mdpress build       # Build PDF

# If build fails despite green doctor:
mdpress build --verbose  # Enable debug output
```

If doctor reports warnings:

- **Optional features:** Build works, but some features unavailable
- **Errors:** Fix before attempting builds

See [common-issues.md](common-issues.md) for detailed fix instructions for each error type.
