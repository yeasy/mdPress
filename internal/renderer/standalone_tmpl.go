package renderer

// standaloneHTMLHead contains the opening HTML structure and first script (FOUC prevention).
const standaloneHTMLHead = `<!DOCTYPE html>
<html lang="{{if .Language}}{{.Language}}{{else}}en{{end}}">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="author" content="{{.Author}}">
  {{if .Description}}<meta name="description" content="{{.Description}}">{{end}}
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
    {{if .HasCover}}
    <!-- Cover hero: synthesized from book metadata (excluded from search/TOC).
         Titles use <div> so they never enter the document outline. -->
    <section class="cover-hero" aria-label="Book cover">
      <div class="cover-hero-inner">
        {{if .CoverImage}}<img class="cover-hero-image" src="{{.CoverImage}}" alt="">{{end}}
        {{if .Title}}<div class="cover-hero-title">{{.Title}}</div>{{end}}
        {{if .Subtitle}}<div class="cover-hero-subtitle">{{.Subtitle}}</div>{{end}}
        <div class="cover-hero-divider"></div>
        <div class="cover-hero-meta">
          {{if .Author}}<div class="cover-hero-meta-item">{{.Author}}</div>{{end}}
          {{if .Version}}<div class="cover-hero-meta-item">{{.Version}}</div>{{end}}
        </div>
      </div>
    </section>
    {{end}}
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
//
// Math is genuinely offline now: formulas are rendered to KaTeX markup at build
// time and the stylesheet, fonts and all, is inlined above, so a reader with no
// network sees typeset formulas rather than raw LaTeX.
//
// Diagrams are not, and deliberately so. mermaid.min.js is 3.5 MB minified
// (4.8 MB once base64-encoded into a data URI), which every portable file would
// carry — including the overwhelming majority that contain no diagram at all —
// to spare the minority a network fetch. So Mermaid still comes from a CDN, and
// the next best thing is to make that dependency honest: the URL is
// version-pinned and integrity-checked, and a failure produces a visible notice
// next to the affected diagram instead of a blank gap and a console warning
// nobody reads.
const standaloneHTMLTail = `  </script>

  <!-- Mermaid loading: version-pinned, integrity-checked, with a visible
       fallback when the CDN is unreachable or blocked. Math needs none of
       this — it was already rendered when the file was built. -->
  <style>
  .mermaid[data-mdpress-asset-error] {
    display: block; white-space: pre; overflow-x: auto; text-align: left;
    font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
    font-size: 0.85rem; background: #f6f8fa; border: 1px solid #e1e4e8;
    border-radius: 4px; padding: 12px 14px;
  }
  .asset-error {
    display: block; margin: 1em 0 0.35em; padding: 8px 12px;
    border-left: 3px solid #b26a00; background: #fff8e6; color: #6b4500;
    font-size: 0.85rem; border-radius: 0 4px 4px 0;
  }
  @media print { .asset-error { border-left-color: #666; background: none; color: #444; } }
  </style>
  <script>
  // One notice per node: diagrams are block level and far apart, so a banner
  // above each affected one reads naturally.
  function mdpressAssetFailure(selector, message) {
    function notice() {
      var note = document.createElement('span');
      note.className = 'asset-error';
      note.setAttribute('role', 'status');
      note.textContent = message;
      return note;
    }
    var nodes = document.querySelectorAll(selector);
    var flagged = [];
    for (var i = 0; i < nodes.length; i++) {
      var node = nodes[i];
      if (node.getAttribute('data-mdpress-asset-error') === 'true') continue;
      node.setAttribute('data-mdpress-asset-error', 'true');
      node.setAttribute('title', message);
      flagged.push(node);
    }
    for (var j = 0; j < flagged.length; j++) {
      flagged[j].parentNode.insertBefore(notice(), flagged[j]);
    }
  }

  if (document.querySelector('.mermaid')) {
    var s = document.createElement('script');
    s.src = '{{MERMAID_CDN_URL}}';
    s.integrity = '{{MERMAID_SRI}}';
    s.crossOrigin = 'anonymous';
    s.referrerPolicy = 'no-referrer';
    s.addEventListener('load', function() { mermaid.initialize({startOnLoad:true, theme:'default', securityLevel:'strict', themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}}); }, { once: true });
    s.addEventListener('error', function() {
      mdpressAssetFailure('.mermaid', 'Diagram not rendered: the Mermaid library could not be loaded (offline, or the CDN is blocked). Its source is shown below.');
    }, { once: true });
    document.body.appendChild(s);
  }
  </script>
</body>
</html>
`
