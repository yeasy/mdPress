package renderer

// htmlTemplate is the full HTML5 document template used for PDF source HTML.
// Template variables:
//
//	.Title - Book title
//	.Author - Book author
//	.CSS - Complete CSS bundle including theme and custom styles
//	.CoverHTML - Cover HTML content
//	.TOCHTML - Table of contents HTML
//	.Chapters - Chapter array, each with .Title .ID .Content
//	.HeaderText - Header text
//	.FooterText - Footer text
const htmlTemplate = `<!DOCTYPE html>
<html lang="{{if .Language}}{{.Language}}{{else}}en{{end}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <meta name="author" content="{{.Author}}">
  <meta name="description" content="{{.Title}}">
  <title>{{.Title}}</title>

  <!-- Embedded CSS -->
  <style>
    /* ============================================
       Base styles and typography
       ============================================ */
    * {
      box-sizing: border-box;
    }

    html {
      font-size: 16px;
      -webkit-font-smoothing: antialiased;
      -moz-osx-font-smoothing: grayscale;
    }

    body {
      margin: 0;
      padding: 0;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Heiti SC", "Heiti TC", "Microsoft YaHei", "Noto Sans SC", "Noto Sans CJK SC", "Source Han Sans SC", "WenQuanYi Micro Hei", "Roboto", "Droid Sans", "Helvetica Neue", sans-serif;
      line-height: 1.6;
      color: #333;
      background-color: #fff;
      overflow-wrap: anywhere;
    }

    /* ============================================
       Cover page styles
       ============================================ */
    .cover-page {
      width: 100%;
      height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      text-align: center;
      page-break-after: always;
      background: transparent;
      position: relative;
    }

    .cover-content {
      padding: 2rem;
    }

    .cover-title {
      font-size: 2.5em;
      font-weight: 700;
      margin: 1rem 0;
      line-height: 1.3;
    }

    .cover-author {
      font-size: 1.2em;
      margin-top: 2rem;
      color: #555;
    }

    /* ============================================
       TOC page styles
       ============================================ */
    .toc-page {
      page-break-after: always;
      padding: 0;
    }

    .toc-title {
      font-size: 2em;
      font-weight: 700;
      margin-bottom: 2rem;
      text-align: center;
    }

    /* Table of contents — the generator emits <nav class="toc"><ul><li><a>. */
    .toc ul {
      list-style: none;
      margin: 0;
      padding: 0;
    }

    .toc li {
      margin: 0.4rem 0;
      line-height: 1.5;
    }

    .toc a {
      color: var(--color-heading, #14263b);
      text-decoration: none;
      /* Title, dot leader and page number sit on one baseline so the number
         is flushed to the right margin the way a printed book sets it. */
      display: flex;
      align-items: baseline;
      gap: 0.35rem;
    }

    .toc-entry-title {
      flex: 0 1 auto;
    }

    .toc-leader {
      flex: 1 1 auto;
      min-width: 1rem;
      /* An empty flex item takes its baseline from its bottom edge, so the
         dots land on the text baseline the way a printed TOC sets them. */
      border-bottom: 1px dotted currentColor;
      opacity: 0.35;
    }

    .toc-pageno {
      /* Reserved even while empty: the first print pass measures the layout
         that the second pass fills in, so the two must be identical. */
      flex: 0 0 auto;
      min-width: 2.2em;
      text-align: right;
      font-variant-numeric: tabular-nums;
      font-weight: 400;
    }

    /* No page number could be resolved — do not lead the eye to a blank. */
    .toc a:has(.toc-pageno:empty) .toc-leader {
      border-bottom: none;
    }

    /* Top-level entries (chapters) stand out. */
    .toc > ul > li {
      margin-top: 0.85rem;
    }

    .toc > ul > li > a {
      font-weight: 600;
      font-size: 1.08em;
    }

    /* Nested entries: indented and lighter. */
    .toc ul ul {
      padding-left: 1.4rem;
      margin-top: 0.15rem;
    }

    .toc ul ul a {
      color: #4a5561;
      font-weight: 400;
      font-size: 0.97em;
    }

    /* ============================================
       Chapter styles
       ============================================ */
    .chapter {
      page-break-before: always;
      padding: 0;
      margin: 0;
    }

    .chapter:first-of-type {
      page-break-before: avoid;
    }

    .chapter-title {
      font-size: 2.1em;
      font-weight: 700;
      margin: 0 0 1.6rem 0;
      padding-bottom: 0.5rem;
      border-bottom: 2px solid var(--color-accent, #1C5A9E);
      color: var(--color-heading, #12344D);
      line-height: 1.25;
    }

    .chapter-content {
      line-height: 1.8;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    /* ============================================
       Heading and paragraph styles
       ============================================ */
    h1 {
      font-size: 2em;
      margin: 1.5rem 0 1rem 0;
      font-weight: 700;
      color: #222;
      page-break-after: avoid;
    }

    h2 {
      font-size: 1.5em;
      margin: 1.3rem 0 0.8rem 0;
      font-weight: 600;
      color: #333;
      page-break-after: avoid;
    }

    h3 {
      font-size: 1.2em;
      margin: 1rem 0 0.6rem 0;
      font-weight: 600;
      color: #444;
      page-break-after: avoid;
    }

    h4, h5, h6 {
      font-size: 1em;
      margin: 0.8rem 0 0.4rem 0;
      font-weight: 600;
      color: #555;
      page-break-after: avoid;
    }

    p {
      margin: 0.8rem 0;
      text-align: justify;
      /* Justification without hyphenation stretches word spacing into rivers
         on a narrow measure; let the browser hyphenate instead. */
      hyphens: auto;
      -webkit-hyphens: auto;
    }

    /* ============================================
       List styles
       ============================================ */
    ul, ol {
      margin: 0.8rem 0;
      padding-left: 2rem;
      line-height: 1.8;
    }

    ul li, ol li {
      margin: 0.4rem 0;
    }

    li > p {
      margin: 0.2rem 0;
    }

    /* ============================================
       Code styles
       ============================================ */
    code {
      font-family: ui-monospace, "SF Mono", Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans Mono CJK SC", monospace;
      font-size: 0.85em;
      color: #333;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    pre {
      border: 1px solid #ddd;
      border-radius: 3px;
      padding: 0.8rem 1rem;
      overflow-x: auto;
      font-size: 0.82em;
      line-height: 1.5;
      margin: 0.8rem 0;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
      word-break: break-all;
      max-width: 100%;
      background: none;
    }

    pre code {
      background: none;
      padding: 0;
      color: #333;
      white-space: inherit;
      overflow-wrap: inherit;
      word-break: inherit;
      display: block;
    }

    /* ============================================
       Table styles
       ============================================ */
    table {
      width: 100%;
      border-collapse: collapse;
      margin: 1rem 0;
      table-layout: fixed;
    }

    /* overflow-wrap: anywhere already breaks a word that cannot fit the fixed
       column width; word-break would additionally chop words that fit fine,
       so cells full of ordinary prose came out hyphen-less and mid-word. */
    table th {
      background-color: #f5f5f5;
      border: 1px solid #ddd;
      padding: 0.8rem;
      text-align: left;
      font-weight: 600;
      overflow-wrap: anywhere;
    }

    table td {
      border: 1px solid #ddd;
      padding: 0.8rem;
      overflow-wrap: anywhere;
    }

    table tbody tr:nth-child(even) {
      background-color: #fafafa;
    }

    /* ============================================
       Blockquote styles
       ============================================ */
    blockquote {
      border-left: 4px solid #667eea;
      margin: 1rem 0;
      padding: 0.5rem 0 0.5rem 1rem;
      background-color: #f9f9f9;
      color: #666;
    }

    blockquote p {
      margin: 0.5rem 0;
    }

    /* ============================================
       Images and media
       ============================================ */
    img {
      max-width: 100%;
      height: auto;
      page-break-inside: avoid;
    }

    /* The print height cap for images is emitted by buildPrintCSS, which
       knows the page size and margins and can express it in millimeters. */

    /* Standalone images (sole child of a paragraph) render as centered blocks
       with a subtle frame. */
    p > img:only-child,
    p > a:only-child > img {
      display: block;
      margin: 1.4rem auto;
      border: 1px solid var(--color-border, #E4E7EB);
      border-radius: 6px;
    }

    figure {
      margin: 1rem 0;
      page-break-inside: avoid;
    }

    figcaption {
      text-align: center;
      font-size: 0.9em;
      color: #666;
      margin-top: 0.5rem;
      font-style: italic;
    }

    p.caption {
      text-align: center;
      font-size: 0.9em;
      color: #666;
    }

    /* ============================================
       Link styles
       ============================================ */
    a {
      color: #0066cc;
      text-decoration: none;
    }

    a:hover {
      text-decoration: underline;
    }

    /* ============================================
       Rules and miscellaneous elements
       ============================================ */
    hr {
      border: none;
      height: 1px;
      background-color: #ddd;
      margin: 2rem 0;
      page-break-after: avoid;
    }

    /* ============================================
       Print-specific styles
       ============================================ */
    @media print {
      /* Remove elements that should not be printed */
      .no-print {
        display: none !important;
      }

      /* Keep page layout clean during printing — pure white background
         to reduce PDF file size (no decorative backgrounds rendered). */
      body {
        margin: 0;
        padding: 0;
        background: white !important;
      }

      /* Blockquotes render without a fill in print — the accent bar carries
         them. Table header and zebra tints are kept: they aid scanning and
         compress well as flat fills. */
      blockquote {
        background-color: transparent !important;
      }

      /* ===== GFM callouts (> [!NOTE] etc.) =====
         Print output is always light; these colors used to be inline style=
         attributes, which nothing could override. */
      .alert { border-left: 4px solid; padding: 12px 16px; margin: 1em 0; border-radius: 0 6px 6px 0; }
      .alert-title { font-weight: 600; margin: 0 0 4px; }
      .alert > :last-child { margin-bottom: 0; }
      .alert-note { background: #ddf4ff; border-left-color: #54aeff; }
      .alert-note .alert-title { color: #0969da; }
      .alert-tip { background: #dafbe1; border-left-color: #4ac26b; }
      .alert-tip .alert-title { color: #1a7f37; }
      .alert-important { background: #fbefff; border-left-color: #c297ff; }
      .alert-important .alert-title { color: #8250df; }
      .alert-warning { background: #fff8c5; border-left-color: #d4a72c; }
      .alert-warning .alert-title { color: #9a6700; }
      .alert-caution { background: #ffebe9; border-left-color: #ff8182; }
      .alert-caution .alert-title { color: #cf222e; }

      /* Print: links should be body color, not blue */
      a {
        color: inherit;
      }

      /* ===== Pagination =====
         page-break-inside: avoid was previously applied to .chapter, ul, ol,
         table and pre. All of those are routinely taller than a page, and the
         rule is all-or-nothing: the browser cannot honor it, so it pushes the
         whole element to the next page and leaves the remainder of the current
         one blank. A book of lists and code blocks — i.e. technical
         documentation — came out riddled with near-empty pages.

         Keep the rule only for elements that are bounded by construction (a
         table row, a list item, a figure), and use orphans/widows for the rest,
         which degrades gracefully when the content does not fit. */
      p, li, blockquote {
        orphans: 3;
        widows: 3;
      }
      tr, li, figure, .alert {
        page-break-inside: avoid;
      }
      /* Repeat table headers when a table does spill across pages. */
      thead {
        display: table-header-group;
      }
      tfoot {
        display: table-footer-group;
      }

      /* Avoid isolated headings: a heading must not be the last thing on a
         page, and should not split. Headings are short, so avoid is safe. */
      h1, h2, h3, h4, h5, h6 {
        page-break-after: avoid;
        page-break-inside: avoid;
      }
    }

    /* ============================================
       Watermark styles
       ============================================ */
    @media print {
      .watermark {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%) rotate(-45deg);
        font-size: 80px;
        font-weight: bold;
        color: rgba(0, 0, 0, {{.WatermarkOpacity}});
        pointer-events: none;
        z-index: 9999;
        white-space: nowrap;
      }
    }

    /* ============================================
       Page rules
       ============================================ */
    @page {
      size: A4;
      margin: 2cm;
    }

    @page :first {
      margin: 0;
    }

    /* ============================================
       Custom styles
       ============================================ */
    {{.CSS}}
  </style>
</head>
<body>
  <!-- Watermark -->
  {{if .Watermark}}
  <div class="watermark">{{.Watermark}}</div>
  {{end}}

  <!-- Cover -->
  {{if .CoverHTML}}
  <div class="cover-page">
    {{.CoverHTML}}
  </div>
  {{end}}

  <!-- Table of contents -->
  {{if .TOCHTML}}
  <div class="toc-page">
    {{.TOCHTML}}
  </div>
  {{end}}

  <!-- Chapter content -->
  {{range .Chapters}}
  <div class="chapter" id="{{.ID}}">
    <h1 class="chapter-title">{{.Title}}</h1>
    <div class="chapter-content">
      {{.Content}}
    </div>
  </div>
  {{end}}

  <!-- Mermaid: auto-detect and load only when diagrams are present -->
  <script>
  if (document.querySelector('.mermaid')) {
    var s = document.createElement('script');
    s.src = '{{MERMAID_CDN_URL}}';
    s.onload = function() { mermaid.initialize({startOnLoad:true, theme:'default', securityLevel:'strict', themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}}); };
    document.body.appendChild(s);
  }
  </script>

  <!-- KaTeX: auto-detect and load only when math formulas are present -->
  <script>
  if (document.querySelector('.math')) {
    var link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = '{{KATEX_CSS_URL}}';
    document.head.appendChild(link);
    var s = document.createElement('script');
    s.src = '{{KATEX_JS_URL}}';
    s.onload = function() {
      var ar = document.createElement('script');
      ar.src = '{{KATEX_AUTO_RENDER_URL}}';
      ar.onload = function() {
        var els = document.querySelectorAll('.chapter-content');
        if (els.length === 0) els = [document.body];
        els.forEach(function(el) {
          renderMathInElement(el, {
            delimiters: [
              {left: '$$', right: '$$', display: true},
              {left: '$',  right: '$',  display: false}
            ],
            throwOnError: false
          });
        });
      };
      document.body.appendChild(ar);
    };
    document.body.appendChild(s);
  }
  </script>
</body>
</html>
`
