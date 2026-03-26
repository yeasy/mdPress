package renderer

// standaloneHTMLHead contains the opening HTML structure and first script (FOUC prevention).
const standaloneHTMLHead = `<!DOCTYPE html>
<html lang="{{if .Language}}{{.Language}}{{else}}en{{end}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="author" content="{{.Author}}">
  <title>{{.Title}}</title>
  <!--
    Prevent theme flash (FOUC): read theme setting from localStorage and apply immediately before page renders.
    This script must be placed inside <head> and executed before any CSS.
  -->
  <script>
  (function() {
    try {
      var t = localStorage.getItem('mdpress-theme') || 'system';
      var dark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
      if (dark) document.documentElement.setAttribute('data-theme', 'dark');
    } catch(e) {}
  })();
  </script>
  <style>
`

// standaloneHTMLMiddle separates CSS from the main JavaScript section.
const standaloneHTMLMiddle = `  </style>
</head>
<body>
  <!-- Reading progress bar -->
  <div id="reading-progress"></div>

  <!-- Top toolbar -->
  <header class="toolbar">
    <button class="toolbar-btn icon-only" id="btn-sidebar" title="Toggle table of contents" aria-label="Toggle table of contents">☰</button>
    <a class="toolbar-brand" href="#">{{.Title}}</a>
    <button class="toolbar-btn" id="btn-search" title="Full-text search (⌘K / Ctrl+K)" aria-label="Search">🔍 Search</button>
    <button class="toolbar-btn icon-only" id="btn-theme" title="Toggle theme" aria-label="Toggle theme">🌙</button>
  </header>

  <!-- Mobile sidebar overlay -->
  <div class="sidebar-overlay" id="sidebar-overlay"></div>

  <!-- Left sidebar: global TOC sidebar -->
  <nav id="left-sidebar">
    <div id="sidebar-nav">
      {{.SidebarHTML}}
    </div>
  </nav>

  <!-- Center: main content area -->
  <main id="main-content">
    {{range .Chapters}}
    <article class="chapter" id="{{.ID}}" data-title="{{.Title}}">
      <h1 class="chapter-title">{{.Title}}</h1>
      <div class="chapter-content">
        {{.Content | safeHTML}}
      </div>
      {{if or .PrevTitle .NextTitle}}
      <nav class="chapter-nav">
        {{if .PrevTitle}}
        <a href="#{{.PrevID}}" class="nav-prev">
          <div class="nav-label">← Previous chapter</div>
          <div class="nav-title">{{.PrevTitle}}</div>
        </a>
        {{end}}
        {{if .NextTitle}}
        <a href="#{{.NextID}}" class="nav-next">
          <div class="nav-label">Next chapter →</div>
          <div class="nav-title">{{.NextTitle}}</div>
        </a>
        {{end}}
      </nav>
      {{end}}
    </article>
    {{end}}
  </main>

  <!-- Right sidebar: current page TOC -->
  <nav id="right-toc-nav">
    <div class="toc-title">On this page</div>
    <div id="toc-list" class="toc-list"></div>
  </nav>

  <!-- Search dialog -->
  <div id="search-overlay" class="search-dialog">
    <div class="search-box">
      <input
        id="search-input"
        type="text"
        class="search-input"
        placeholder="Search documents... (press ESC to close)"
        autocomplete="off"
      >
      <div class="search-count-label" id="search-count-label"></div>
      <div class="search-results-list" id="search-results-list"></div>
    </div>
  </div>

  <!-- Image lightbox -->
  <div class="img-lightbox" id="img-lightbox" role="dialog" aria-modal="true" aria-label="Image preview">
    <img id="img-lightbox-src" src="" alt="">
  </div>

  <!-- Back to top -->
  <button id="back-to-top" aria-label="Back to top">↑</button>

  <script>
`

// standaloneHTMLTail completes the main JavaScript block and adds CDN-loaded scripts.
const standaloneHTMLTail = `  </script>

  <!-- Mermaid: auto-detect and load only when diagrams are present -->
  <script>
  if (document.querySelector('.mermaid')) {
    var s = document.createElement('script');
    s.src = '{{MERMAID_CDN_URL}}';
    s.onload = function() { mermaid.initialize({startOnLoad:true, theme:'default', themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}}); };
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
