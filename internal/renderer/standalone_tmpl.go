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
    防止主题闪烁（FOUC）：在页面渲染前从 localStorage 读取主题设置并立即应用。
    此脚本必须放在 <head> 内，在任何 CSS 之前执行。
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
  <!-- 阅读进度条 -->
  <div id="reading-progress"></div>

  <!-- 顶部工具栏 -->
  <header class="toolbar">
    <button class="toolbar-btn icon-only" id="btn-sidebar" title="切换目录" aria-label="切换目录">☰</button>
    <a class="toolbar-brand" href="#">{{.Title}}</a>
    <button class="toolbar-btn" id="btn-search" title="全文搜索 (⌘K / Ctrl+K)" aria-label="搜索">🔍 搜索</button>
    <button class="toolbar-btn icon-only" id="btn-theme" title="切换主题" aria-label="切换主题">🌙</button>
  </header>

  <!-- 移动端侧边栏遮罩 -->
  <div class="sidebar-overlay" id="sidebar-overlay"></div>

  <!-- 左侧栏：全局 TOC 侧边栏 -->
  <nav id="left-sidebar">
    <div id="sidebar-nav">
      {{.SidebarHTML}}
    </div>
  </nav>

  <!-- 中间栏：主内容区域 -->
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
          <div class="nav-label">← 上一章</div>
          <div class="nav-title">{{.PrevTitle}}</div>
        </a>
        {{end}}
        {{if .NextTitle}}
        <a href="#{{.NextID}}" class="nav-next">
          <div class="nav-label">下一章 →</div>
          <div class="nav-title">{{.NextTitle}}</div>
        </a>
        {{end}}
      </nav>
      {{end}}
    </article>
    {{end}}
  </main>

  <!-- 右侧栏：当前页 TOC -->
  <nav id="right-toc-nav">
    <div class="toc-title">本页目录</div>
    <div id="toc-list" class="toc-list"></div>
  </nav>

  <!-- 搜索对话框 -->
  <div id="search-overlay" class="search-dialog">
    <div class="search-box">
      <input
        id="search-input"
        type="text"
        class="search-input"
        placeholder="搜索文档... (按 ESC 关闭)"
        autocomplete="off"
      >
      <div class="search-count-label" id="search-count-label"></div>
      <div class="search-results-list" id="search-results-list"></div>
    </div>
  </div>

  <!-- 图片灯箱 -->
  <div class="img-lightbox" id="img-lightbox" role="dialog" aria-modal="true" aria-label="图片预览">
    <img id="img-lightbox-src" src="" alt="">
  </div>

  <!-- 回到顶部 -->
  <button id="back-to-top" aria-label="回到顶部">↑</button>

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
