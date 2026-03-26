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
      background: white;
      color: #1A5490;
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
      page-break-inside: avoid;
      padding: 0;
    }

    .toc-title {
      font-size: 2em;
      font-weight: 700;
      margin-bottom: 2rem;
      text-align: center;
    }

    .toc-list {
      list-style: none;
      padding: 0;
      margin: 0;
    }

    .toc-item {
      margin: 0.5rem 0;
      padding-left: 2rem;
      text-indent: -2rem;
    }

    .toc-item a {
      color: #333;
      text-decoration: none;
    }

    .toc-item a:hover {
      text-decoration: underline;
    }

    .toc-item-level-1 {
      font-weight: 600;
      font-size: 1.1em;
      margin-top: 1rem;
    }

    .toc-item-level-2 {
      margin-left: 2rem;
      font-size: 0.95em;
      color: #666;
    }

    .toc-item-level-3 {
      margin-left: 4rem;
      font-size: 0.9em;
      color: #999;
    }

    /* ============================================
       Chapter styles
       ============================================ */
    .chapter {
      page-break-before: always;
      page-break-inside: avoid;
      padding: 0;
      margin: 0;
    }

    .chapter:first-of-type {
      page-break-before: avoid;
    }

    .chapter-title {
      font-size: 2em;
      font-weight: 700;
      margin: 0 0 1.5rem 0;
      color: #222;
      line-height: 1.3;
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
      page-break-inside: avoid;
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
      page-break-inside: avoid;
      table-layout: fixed;
    }

    table th {
      background-color: #f5f5f5;
      border: 1px solid #ddd;
      padding: 0.8rem;
      text-align: left;
      font-weight: 600;
      overflow-wrap: anywhere;
      word-break: break-word;
    }

    table td {
      border: 1px solid #ddd;
      padding: 0.8rem;
      overflow-wrap: anywhere;
      word-break: break-word;
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
      page-break-inside: avoid;
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

    /* Standalone images (sole child of a paragraph) render as centered blocks. */
    p > img:only-child,
    p > a:only-child > img {
      display: block;
      margin: 1rem auto;
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

      /* Strip decorative backgrounds that inflate PDF size.
         Code blocks keep their background since it aids readability. */
      table tbody tr:nth-child(even) {
        background-color: transparent !important;
      }
      blockquote {
        background-color: transparent !important;
      }
      table th {
        background-color: transparent !important;
      }

      /* Avoid splitting page content unexpectedly */
      .chapter {
        page-break-inside: avoid;
      }

      /* Print: links should be body color, not blue */
      a {
        color: inherit;
      }

      /* Keep tables and code blocks on the same page when possible */
      table, pre {
        page-break-inside: avoid;
      }

      /* Avoid isolated headings */
      h1, h2, h3, h4, h5, h6 {
        page-break-after: avoid;
        page-break-inside: avoid;
      }

      /* Avoid splitting lists */
      ul, ol {
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
        renderMathInElement(document.body, {
          delimiters: [
            {left: '$$', right: '$$', display: true},
            {left: '$',  right: '$',  display: false}
          ],
          throwOnError: false
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
