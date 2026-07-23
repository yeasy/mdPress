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
// This file is advertised as readable offline, but Mermaid and KaTeX still come
// from a CDN. Until they are vendored, the next best thing is to make the
// dependency honest: every URL is version-pinned and integrity-checked, and a
// failure produces a visible notice next to the affected diagram or formula
// instead of a blank gap and a console warning nobody reads.
const standaloneHTMLTail = `  </script>

  <!-- Third-party asset loading: version-pinned, integrity-checked, with a
       visible fallback when the CDN is unreachable or blocked. -->
  <style>
  .mermaid[data-mdpress-asset-error] {
    display: block; white-space: pre; overflow-x: auto; text-align: left;
    font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
    font-size: 0.85rem; background: #f6f8fa; border: 1px solid #e1e4e8;
    border-radius: 4px; padding: 12px 14px;
  }
  .math[data-mdpress-asset-error] { border-bottom: 1px dotted #b26a00; }
  .asset-error {
    display: block; margin: 1em 0 0.35em; padding: 8px 12px;
    border-left: 3px solid #b26a00; background: #fff8e6; color: #6b4500;
    font-size: 0.85rem; border-radius: 0 4px 4px 0;
  }
  @media print { .asset-error { border-left-color: #666; background: none; color: #444; } }
  </style>
  <script>
  // perNode is true for diagrams, which are block level and far apart; it is
  // false for inline math, where one banner per formula would shred the prose.
  function mdpressAssetFailure(selector, message, perNode) {
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
    if (!flagged.length) return;
    if (perNode) {
      for (var j = 0; j < flagged.length; j++) {
        flagged[j].parentNode.insertBefore(notice(), flagged[j]);
      }
      return;
    }
    var host = document.getElementById('main-content') || document.body;
    host.insertBefore(notice(), host.firstChild);
  }

  if (document.querySelector('.mermaid')) {
    var s = document.createElement('script');
    s.src = '{{MERMAID_CDN_URL}}';
    s.integrity = '{{MERMAID_SRI}}';
    s.crossOrigin = 'anonymous';
    s.referrerPolicy = 'no-referrer';
    s.addEventListener('load', function() { mermaid.initialize({startOnLoad:true, theme:'default', securityLevel:'strict', themeVariables:{fontFamily:'"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif'}}); }, { once: true });
    s.addEventListener('error', function() {
      mdpressAssetFailure('.mermaid', 'Diagram not rendered: the Mermaid library could not be loaded (offline, or the CDN is blocked). Its source is shown below.', true);
    }, { once: true });
    document.body.appendChild(s);
  }

  if (document.querySelector('.math')) {
    var mathFailed = function() {
      mdpressAssetFailure('.math', 'Some formulas on this page are not rendered: the KaTeX library could not be loaded (offline, or the CDN is blocked). They are shown as LaTeX source.', false);
    };
    var link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = '{{KATEX_CSS_URL}}';
    link.integrity = '{{KATEX_CSS_SRI}}';
    link.crossOrigin = 'anonymous';
    link.referrerPolicy = 'no-referrer';
    link.addEventListener('error', mathFailed, { once: true });
    document.head.appendChild(link);
    var s = document.createElement('script');
    s.src = '{{KATEX_JS_URL}}';
    s.integrity = '{{KATEX_JS_SRI}}';
    s.crossOrigin = 'anonymous';
    s.referrerPolicy = 'no-referrer';
    s.addEventListener('error', mathFailed, { once: true });
    s.addEventListener('load', function() {
      var ar = document.createElement('script');
      ar.src = '{{KATEX_AUTO_RENDER_URL}}';
      ar.integrity = '{{KATEX_AUTO_RENDER_SRI}}';
      ar.crossOrigin = 'anonymous';
      ar.referrerPolicy = 'no-referrer';
      ar.addEventListener('error', mathFailed, { once: true });
      ar.addEventListener('load', function() {
        renderMathInElement(document.getElementById('main-content') || document.body, {
          delimiters: [
            {left: '$$', right: '$$', display: true},
            {left: '$',  right: '$',  display: false}
          ],
          throwOnError: false
        });
      }, { once: true });
      document.body.appendChild(ar);
    }, { once: true });
    document.body.appendChild(s);
  }
  </script>
</body>
</html>
`
