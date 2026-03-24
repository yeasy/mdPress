# Search Functionality

mdPress includes built-in full-text search that works entirely in the browser without requiring a backend server. This guide explains how search works and how to use it effectively.

## How Search Works

The search functionality in mdPress is client-side, meaning all search operations happen in your browser without contacting a server.

### Search Index

When you build your documentation, mdPress generates a search index file called `search-index.json`. This file contains:

- All headings and subheadings from your documentation
- Full text content from each section
- Page metadata (titles, URLs, chapter hierarchy)
- Word frequency data for relevance ranking

The search index is automatically included in your site and single-file HTML output.

### Index Generation

The search index is created during the build process:

```bash
mdpress build --format site
```

mdPress automatically:
1. Extracts all text content from your Markdown
2. Indexes headings with higher weight than body text
3. Creates entry points for each section
4. Compresses the index for fast loading
5. Embeds the index in your output files

### Index Size

The search index size depends on your documentation:
- Small documentation (10-50 pages): 100-300KB
- Medium documentation (50-200 pages): 300KB-1MB
- Large documentation (200+ pages): 1-5MB

The index is only loaded when needed and cached in the browser.

## Using the Search Feature

Users can access search in several ways depending on the output format.

### Opening the Search Dialog

Press the keyboard shortcut to open the search dialog:

- **Mac**: Cmd+K
- **Windows/Linux**: Ctrl+K

Alternatively, click the search button in the documentation interface (if visible).

### Performing a Search

1. Open the search dialog with Cmd/Ctrl+K
2. Type your search query
3. Results appear in real-time as you type
4. Click a result to jump to that section
5. Press Escape to close the search dialog

### Search Query Syntax

Basic search uses simple keyword matching:

```
api documentation
```

Searches for both "api" and "documentation" in order of relevance.

Multiple terms:
```
REST API authentication
```

Returns results containing all three terms, ranked by relevance.

Quoted phrases:
```
"authentication token"
```

Searches for the exact phrase "authentication token".

### Search Results

Results are displayed with:

- **Title**: Section or heading where the match occurs
- **Preview**: A snippet of text showing the match in context
- **Highlighting**: Search terms are highlighted in the preview
- **Score**: Relevance ranking (higher = better match)
- **Breadcrumb**: Chapter hierarchy showing where the result is located

## Search Result Highlighting

When you navigate to a search result, the page automatically:

1. Scrolls to the matching section
2. Highlights the search terms in the content
3. Updates the browser URL to point to the specific section
4. Updates the table of contents to show your current position

Highlighting is temporary and fades after you interact with the page.

## CJK Search Support

mdPress supports search in Chinese, Japanese, and Korean languages.

### CJK Tokenization

For CJK languages (Chinese, Japanese, Korean), mdPress uses specialized tokenization:

- Characters are indexed individually since these languages don't use spaces
- Each character can match independently
- Search results include all content containing the search characters
- Ranking accounts for character proximity and frequency

### Example CJK Searches

Chinese search:
```
API文档
```

Returns results containing "API" or "文档" (documentation).

Japanese search:
```
認証トークン
```

Returns results with matching characters.

Korean search:
```
인증 토큰
```

Returns results with matching characters and words.

### Mixed Language Search

Search across mixed language content:
```
API 文档 authentication
```

Finds results containing English and CJK terms.

## Search Configuration

Configure search behavior in your `book.yaml`:

```yaml
search:
  enabled: true
  languages: ["en", "zh", "ja", "ko"]
  maxResults: 10
  highlightResults: true
  indexPrivatePages: false
```

### Search Options

- `enabled`: Enable/disable search functionality (default: true)
- `languages`: Languages to support for tokenization
- `maxResults`: Maximum search results to display (default: 10)
- `highlightResults`: Highlight search terms in content (default: true)
- `indexPrivatePages`: Whether to index pages marked as private (default: false)

## Search in Different Output Formats

### Site Format

Full search functionality with:
- Real-time search as you type
- Multiple result highlighting
- Full-featured search dialog
- Keyboard shortcuts

### Single-File HTML

Search works identically to site format:
- Index is embedded in the HTML file
- No external requests needed
- Works offline completely
- All features available

### PDF Format

Search is provided by the PDF reader:
- Use your PDF viewer's search function (Cmd/Ctrl+F)
- No mdPress-specific search indexing
- Search capabilities depend on PDF viewer

### ePub Format

Search depends on the e-reader application:
- Use the e-reader's built-in search
- Quality depends on the device
- Works offline on the device

## Performance and Optimization

### Index Size Management

For very large documentation, the search index can be optimized:

```yaml
search:
  maxResults: 20
  excludePatterns:
    - "*.draft.md"
    - "appendix/**"
```

Excluding large sections reduces index size.

### Lazy Loading the Index

The search index is lazy-loaded, meaning:
- It's not loaded until the user opens search
- First search may take 100-500ms to load the index
- Subsequent searches are instant
- The index is cached for the session

### Search Response Times

Typical search response times:
- Indexing on build: 1-5 seconds for most projects
- First search (cold): 100-500ms
- Subsequent searches (warm): 10-50ms
- Displaying results: <100ms

## Advanced Search Features

### Proximity Search

Some search implementations support proximity:
```
"API" within 5 words of "authentication"
```

(Availability depends on mdPress version and configuration.)

### Wildcard Search

Search with wildcards:
```
auth*
```

Matches "authentication", "authorize", "authtoken", etc.

### Boolean Search

Combine search terms with AND/OR:
```
API AND authentication
```

Returns results containing both terms.

## Troubleshooting

### Search Not Working

Verify that:
1. JavaScript is enabled in your browser
2. The search index file is present (check in browser DevTools Network tab)
3. The search dialog opens (try Cmd/Ctrl+K)
4. Try in a different browser

If search still doesn't work, rebuild your documentation:
```bash
mdpress build --format site
```

### Slow Search Performance

For large documentation (500+ pages):
1. Limit max results in configuration
2. Reduce index size by excluding sections
3. Clear browser cache and reload
4. Use a modern browser (Chrome, Firefox, Safari, Edge)

### Incorrect Search Results

If search results seem wrong:
1. Verify the search index was regenerated in the latest build
2. Check that content is properly formatted in your Markdown
3. Search for different keywords
4. Rebuild with `--force` flag if available

### CJK Search Issues

If CJK search doesn't work:
1. Verify CJK languages are listed in `book.yaml` search configuration
2. Ensure content is encoded in UTF-8
3. Check browser console for errors
4. Try searching in the original language, not translated text

### Search Index Too Large

To reduce search index size:

```yaml
search:
  maxResults: 5
  excludePatterns:
    - "api-reference/**"  # Very large section
    - "appendix/**"
  indexLevel: 2  # Index only H1 and H2
```

## SEO and Search Engines

The mdPress search feature is for in-documentation search, not web search engine optimization.

### External Search Indexing

To help external search engines like Google index your documentation:

1. Ensure your site has a sitemap (`sitemap.xml` is generated automatically)
2. Add proper meta tags in your `book.yaml` configuration
3. Submit your sitemap to Google Search Console
4. Ensure your hosting allows search engine crawling

### Robots and Crawling

Control how search engines index your site:

```yaml
site:
  robots: "index, follow"
  canonical: "https://docs.example.com"
```

This helps search engines understand your documentation structure.

## Best Practices

### Structure for Better Search

Write clear headings and descriptive section titles:

```markdown
# Configuring API Authentication

## Bearer Token Authentication

## OAuth 2.0 Integration
```

This creates better search results than:

```markdown
# Setup

## Option 1

## Option 2
```

### Use Consistent Terminology

Use the same terms consistently throughout your documentation. For example, always use "authentication" rather than mixing "auth", "login", and "sign-in".

### Include Common Synonyms

In your content, mention alternate terms that users might search for:

```markdown
# Authentication (Login, Sign-in)

Users can authenticate, login, or sign-in to their accounts.
```

This helps search find your content even when users search for synonyms.

### Add Search-Friendly Metadata

Use your glossary and structured content to improve search:

```yaml
glossary:
  - term: "API Key"
    aliases: ["API token", "access key"]
```

This helps search find content even when using different terminology.
