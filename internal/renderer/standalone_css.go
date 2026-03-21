package renderer

// standaloneCSS contains the embedded CSS styles for the standalone HTML renderer.
// This includes color variables, layout styles, typography, callouts, tables, code blocks, and theme support.
const standaloneCSS = `    /* ============================================================
       CSS 变量 - 亮色模式默认值
       ============================================================ */
    :root {
      --color-bg:           #ffffff;
      --color-bg-alt:       #f8f9fa;
      --color-bg-sidebar:   #f5f5f7;
      --color-text:         #1f2328;
      --color-text-muted:   #656d76;
      --color-heading:      #0d1117;
      --color-link:         #0969da;
      --color-link-hover:   #0550ae;
      --color-border:       #d0d7de;
      --color-accent:       #0969da;
      --color-accent-light: rgba(9, 105, 218, 0.08);

      /* 代码块 */
      --color-code-bg:      #f6f8fa;
      --color-code-border:  #d0d7de;
      --color-code-text:    #cf222e;
      --color-code-lang:    #57606a;

      /* 侧边栏 */
      --color-sidebar-hover:  rgba(9, 105, 218, 0.06);
      --color-sidebar-active: #0969da;
      --color-sidebar-active-bg: rgba(9, 105, 218, 0.1);

      /* 表格 */
      --color-table-header: #f6f8fa;
      --color-table-stripe: #ffffff;
      --color-table-stripe-alt: #f6f8fa;
      --color-table-hover:  #eef2ff;

      /* Callout 提示框 */
      --callout-note-bg:        #dbeafe;
      --callout-note-border:    #2563eb;
      --callout-note-color:     #1e40af;
      --callout-warning-bg:     #fef3c7;
      --callout-warning-border: #d97706;
      --callout-warning-color:  #92400e;
      --callout-tip-bg:         #dcfce7;
      --callout-tip-border:     #16a34a;
      --callout-tip-color:      #15803d;
      --callout-important-bg:   #fee2e2;
      --callout-important-border: #dc2626;
      --callout-important-color: #9f1239;

      /* 进度条 */
      --color-progress: #0969da;
    }

    /* ============================================================
       深色模式覆盖
       ============================================================ */
    [data-theme="dark"] {
      --color-bg:           #1a1a1a;
      --color-bg-alt:       #2a2a2a;
      --color-bg-sidebar:   #1c1c1e;
      --color-text:         #c9d1d9;
      --color-text-muted:   #8b949e;
      --color-heading:      #f0f6fc;
      --color-link:         #58a6ff;
      --color-link-hover:   #79c0ff;
      --color-border:       #30363d;
      --color-accent:       #58a6ff;
      --color-accent-light: rgba(88, 166, 255, 0.15);

      /* 代码块 */
      --color-code-bg:      #242424;
      --color-code-border:  #3a3a3a;
      --color-code-text:    #ff7b72;
      --color-code-lang:    #8b949e;

      /* 侧边栏 */
      --color-sidebar-hover:  rgba(88, 166, 255, 0.1);
      --color-sidebar-active: #58a6ff;
      --color-sidebar-active-bg: rgba(88, 166, 255, 0.15);

      /* 表格 */
      --color-table-header: #161b22;
      --color-table-stripe: #0d1117;
      --color-table-stripe-alt: #010409;
      --color-table-hover:  #1f6feb;

      /* Callout 提示框 */
      --callout-note-bg:        #0f2d4d;
      --callout-note-border:    #0969da;
      --callout-note-color:     #79c0ff;
      --callout-warning-bg:     #3d2817;
      --callout-warning-border: #d29922;
      --callout-warning-color:  #d29922;
      --callout-tip-bg:         #0f3d1f;
      --callout-tip-border:     #3fb950;
      --callout-tip-color:      #3fb950;
      --callout-important-bg:   #3d1f1a;
      --callout-important-border: #da3633;
      --callout-important-color: #f85149;

      /* 进度条 */
      --color-progress: #58a6ff;
    }

    /* ============================================================
       全局样式
       ============================================================ */
    * {
      box-sizing: border-box;
    }

    html, body {
      margin: 0;
      padding: 0;
    }

    body {
      background-color: var(--color-bg);
      color: var(--color-text);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Heiti SC", "Heiti TC", "Microsoft YaHei", "Noto Sans SC", "Noto Sans CJK SC", "Source Han Sans SC", "WenQuanYi Micro Hei", "Noto Sans", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";
      font-size: 16px;
      line-height: 1.7;
      font-feature-settings: 'kern' 1;
      -webkit-font-smoothing: antialiased;
      transition: background-color 0.3s, color 0.3s;
    }

    /* ============================================================
       阅读进度条
       ============================================================ */
    #reading-progress {
      position: fixed;
      top: 0;
      left: 0;
      height: 2px;
      background: var(--color-progress);
      z-index: 999;
      transition: width 0.1s ease;
    }

    /* ============================================================
       顶部工具栏
       ============================================================ */
    .toolbar {
      position: sticky;
      top: 0;
      z-index: 100;
      background-color: var(--color-bg);
      border-bottom: 1px solid var(--color-border);
      padding: 0 1rem;
      height: 60px;
      display: flex;
      align-items: center;
      gap: 1rem;
      box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
    }

    .toolbar-btn {
      background: none;
      border: none;
      color: var(--color-text);
      cursor: pointer;
      padding: 0.5rem 0.75rem;
      border-radius: 6px;
      font-size: 14px;
      white-space: nowrap;
      transition: all 0.15s ease;
      min-height: 44px;
      min-width: 44px;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .toolbar-btn:hover {
      background-color: var(--color-bg-alt);
      color: var(--color-link);
    }

    .toolbar-btn:active {
      transform: scale(0.97);
    }

    .toolbar-btn.icon-only {
      padding: 0.5rem 0.25rem;
    }

    .toolbar-brand {
      margin-right: auto;
      font-weight: 600;
      text-decoration: none;
      color: var(--color-heading);
      font-size: 16px;
    }

    /* ============================================================
       主容器布局（三栏）
       ============================================================ */
    body {
      display: grid;
      grid-template-columns: 250px 1fr 300px;
      grid-template-rows: 60px 1fr;
      height: 100vh;
      gap: 0;
    }

    #reading-progress {
      grid-column: 1 / -1;
      grid-row: 1;
      z-index: 101;
    }

    .toolbar {
      grid-column: 1 / -1;
      grid-row: 1;
    }

    #left-sidebar {
      grid-column: 1;
      grid-row: 2;
      background-color: var(--color-bg-sidebar);
      border-right: 1px solid var(--color-border);
      overflow-y: auto;
      padding: 1rem 0;
      max-height: calc(100vh - 60px);
    }

    #main-content {
      grid-column: 2;
      grid-row: 2;
      overflow-y: auto;
      padding: 2rem 2.5rem;
    }

    #right-toc-nav {
      grid-column: 3;
      grid-row: 2;
      background-color: var(--color-bg);
      border-left: 1px solid var(--color-border);
      overflow-y: auto;
      padding: 1.5rem 1rem;
      max-height: calc(100vh - 60px);
      font-size: 13px;
    }

    /* Desktop sidebar collapse states */
    body.sidebar-collapsed {
      grid-template-columns: 0px 1fr 300px;
    }

    #left-sidebar.sidebar-collapsed {
      display: none;
    }

    #main-content.left-expanded {
      grid-column: 2;
    }

    .toc-title {
      font-weight: 600;
      color: var(--color-heading);
      margin-bottom: 1rem;
      padding: 0 0.5rem;
    }

    .toc-list {
      list-style: none;
      padding: 0;
      margin: 0;
    }

    /* ============================================================
       左侧边栏（全局 TOC）
       ============================================================ */
    #sidebar-nav {
      --sidebar-padding-left: 1rem;
    }

    .toc-group {
      margin: 0;
      padding: 0;
    }

    .toc-group:first-of-type {
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.5rem;
      margin-bottom: 0.5rem;
    }

    .toc-row {
      display: flex;
      align-items: center;
      padding: 0.5rem var(--sidebar-padding-left);
    }

    .toc-toggle, .toc-spacer {
      width: 20px;
      height: 24px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .toc-toggle {
      background: none;
      border: none;
      color: var(--color-text-muted);
      cursor: pointer;
      padding: 0;
      transition: transform 0.2s;
    }

    .toc-toggle::after {
      content: '▶';
      font-size: 12px;
    }

    .toc-toggle[aria-expanded="true"]::after {
      transform: rotate(90deg);
    }

    .toc-toggle:hover {
      color: var(--color-text);
    }

    .toc-link {
      flex: 1;
      text-decoration: none;
      color: var(--color-text);
      padding: 0.5rem 0.75rem;
      border-radius: 6px;
      transition: all 0.15s ease;
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-size: 14px;
    }

    .toc-link:hover {
      background-color: rgba(9, 105, 218, 0.1);
      color: var(--color-text);
      transform: translateX(2px);
    }

    .toc-link.active {
      color: var(--color-sidebar-active);
      background-color: rgba(9, 105, 218, 0.08);
      font-weight: 500;
      border-left: 3px solid var(--color-accent);
      padding-left: calc(0.75rem - 3px);
    }

    .toc-link-chapter {
      font-weight: 500;
    }

    .toc-depth-1 { --indent: 0px; }
    .toc-depth-2 { --indent: 0px; }
    .toc-depth-3 { --indent: 16px; }
    .toc-depth-4 { --indent: 32px; }
    .toc-depth-5 { --indent: 48px; }
    .toc-depth-6 { --indent: 64px; }

    .toc-heading-depth-1 { margin-left: 0; }
    .toc-heading-depth-2 { margin-left: 16px; }
    .toc-heading-depth-3 { margin-left: 32px; }
    .toc-heading-depth-4 { margin-left: 48px; }
    .toc-heading-depth-5 { margin-left: 64px; }
    .toc-heading-depth-6 { margin-left: 80px; }

    .toc-children {
      padding-left: 16px;
    }

    .toc-children[hidden] {
      display: none;
    }

    /* ============================================================
       移动端侧边栏遮罩
       ============================================================ */
    .sidebar-overlay {
      display: none;
      position: fixed;
      top: 60px;
      left: 0;
      right: 0;
      bottom: 0;
      background: rgba(0, 0, 0, 0.3);
      backdrop-filter: blur(10px);
      z-index: 99;
      transition: opacity 0.3s ease;
    }

    .sidebar-overlay.visible {
      display: block;
    }

    /* ============================================================
       主内容区域
       ============================================================ */
    #main-content {
      max-width: 860px;
      margin: 0 auto;
    }

    .chapter {
      margin-bottom: 3rem;
      scroll-margin-top: 80px;
    }

    .chapter-title {
      font-size: 2.5rem;
      font-weight: 700;
      margin: 1rem 0 0.5rem 0;
      color: var(--color-heading);
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.5rem;
    }

    /* ============================================================
       标题（h1 - h6）
       ============================================================ */
    h1, h2, h3, h4, h5, h6 {
      color: var(--color-heading);
      font-weight: 600;
      margin: 1.5rem 0 0.5rem 0;
      scroll-margin-top: 80px;
      letter-spacing: -0.02em;
    }

    h1 {
      font-size: 2rem;
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.3rem;
    }

    h2 {
      font-size: 1.75rem;
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.5rem;
    }

    h3 { font-size: 1.5rem; }
    h4 { font-size: 1.25rem; }
    h5 { font-size: 1.1rem; }
    h6 { font-size: 1rem; }

    /* ============================================================
       段落和基础文本
       ============================================================ */
    p {
      margin: 1rem 0;
      line-height: 1.8;
    }

    a {
      color: var(--color-link);
      text-decoration: none;
      transition: color 0.2s;
    }

    a:hover {
      color: var(--color-link-hover);
      text-decoration: underline;
    }

    a:focus-visible {
      outline: 2px solid var(--color-accent);
      outline-offset: 2px;
      border-radius: 2px;
    }

    button:focus-visible,
    input:focus-visible,
    textarea:focus-visible,
    select:focus-visible {
      outline: 2px solid var(--color-accent);
      outline-offset: 2px;
    }

    strong {
      font-weight: 600;
      color: var(--color-heading);
    }

    em {
      font-style: italic;
    }

    code {
      background-color: var(--color-code-bg);
      color: var(--color-code-text);
      padding: 0.2em 0.4em;
      border-radius: 3px;
      font-family: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, "Courier New", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans Mono CJK SC", monospace;
      font-size: 0.9em;
      border: 1px solid var(--color-code-border);
    }

    pre {
      background-color: var(--color-code-bg);
      border: 1px solid var(--color-code-border);
      border-radius: 8px;
      border-top-left-radius: 0;
      border-top-right-radius: 0;
      padding: 1rem;
      overflow-x: auto;
      margin: 1rem 0;
      position: relative;
    }

    pre code {
      background: none;
      color: var(--color-text);
      padding: 0;
      border: none;
      font-size: 0.85rem;
    }

    /* ============================================================
       代码块增强
       ============================================================ */
    .code-block {
      position: relative;
      margin: 1rem 0;
      border-radius: 8px;
      overflow: hidden;
    }

    .code-block::before {
      content: attr(data-lang);
      display: block;
      background-color: var(--color-bg-alt);
      padding: 0.5rem 1rem;
      font-size: 12px;
      font-weight: 600;
      color: var(--color-code-lang);
      text-transform: uppercase;
      border-bottom: 1px solid var(--color-border);
    }

    .code-lang {
      position: absolute;
      top: 0.5rem;
      right: 3rem;
      font-size: 12px;
      color: var(--color-code-lang);
      text-transform: uppercase;
      font-weight: 600;
      z-index: 1;
      background-color: transparent;
      padding: 0;
      border-radius: 0;
      display: none;
    }

    .code-copy {
      position: absolute;
      top: 0.5rem;
      right: 0.5rem;
      background-color: transparent;
      border: none;
      color: var(--color-text-muted);
      border-radius: 6px;
      padding: 0.5rem 0.5rem;
      font-size: 14px;
      cursor: pointer;
      transition: all 0.15s ease;
      z-index: 2;
      opacity: 0;
    }

    .code-block:hover .code-copy {
      opacity: 1;
      color: var(--color-text);
    }

    .code-copy:hover {
      background-color: var(--color-bg-alt);
      color: var(--color-accent);
    }

    .code-copy.copied {
      background-color: var(--color-accent-light);
      color: var(--color-accent);
    }

    /* ============================================================
       列表
       ============================================================ */
    ul, ol {
      margin: 1rem 0;
      padding-left: 2rem;
    }

    li {
      margin: 0.5rem 0;
    }

    li p {
      margin: 0.25rem 0;
    }

    /* ============================================================
       引用
       ============================================================ */
    blockquote {
      margin: 1rem 0;
      padding-left: 1.5rem;
      border-left: 3px solid var(--color-border);
      background-color: transparent;
      padding: 0.5rem 1rem;
      border-radius: 0;
      color: var(--color-text-muted);
    }

    blockquote p {
      margin: 0.5rem 0;
    }

    /* ============================================================
       表格
       ============================================================ */
    .table-wrapper {
      border-radius: 8px;
      overflow: hidden;
      border: 1px solid var(--color-border);
      margin: 1rem 0;
    }

    table {
      width: 100%;
      border-collapse: collapse;
      margin: 0;
      border: none;
    }

    thead {
      background-color: var(--color-table-header);
    }

    th {
      padding: 0.75rem;
      text-align: left;
      font-weight: 500;
      color: var(--color-text);
      border: none;
      border-bottom: 1px solid var(--color-border);
    }

    td {
      padding: 0.75rem;
      border: none;
      border-bottom: 1px solid var(--color-border);
    }

    tbody tr:last-child td {
      border-bottom: none;
    }

    tbody tr:nth-child(even) {
      background-color: var(--color-table-stripe-alt);
    }

    tbody tr:hover {
      background-color: var(--color-table-hover);
    }

    /* ============================================================
       Callout 提示框（note, warning, tip, important）
       ============================================================ */
    .callout {
      margin: 1rem 0;
      padding: 1rem 1rem 1rem 3rem;
      border-left: 3px solid;
      border-radius: 8px;
      background-color: var(--color-bg-alt);
      position: relative;
    }

    .callout::before {
      position: absolute;
      left: 1rem;
      top: 1rem;
      font-size: 1.25rem;
      width: 1.5rem;
      height: 1.5rem;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    .callout-title {
      font-weight: 600;
      margin-bottom: 0.5rem;
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }

    .callout-note {
      border-color: var(--callout-note-border);
      background-color: rgba(219, 234, 254, 0.4);
      color: var(--callout-note-color);
    }

    .callout-note::before {
      content: "ℹ️";
    }

    .callout-note .callout-title {
      color: var(--callout-note-color);
    }

    .callout-warning {
      border-color: var(--callout-warning-border);
      background-color: rgba(254, 243, 199, 0.4);
      color: var(--callout-warning-color);
    }

    .callout-warning::before {
      content: "⚠️";
    }

    .callout-warning .callout-title {
      color: var(--callout-warning-color);
    }

    .callout-tip {
      border-color: var(--callout-tip-border);
      background-color: rgba(220, 252, 231, 0.4);
      color: var(--callout-tip-color);
    }

    .callout-tip::before {
      content: "💡";
    }

    .callout-tip .callout-title {
      color: var(--callout-tip-color);
    }

    .callout-important {
      border-color: var(--callout-important-border);
      background-color: rgba(254, 226, 226, 0.4);
      color: var(--callout-important-color);
    }

    .callout-important::before {
      content: "🔴";
    }

    .callout-important .callout-title {
      color: var(--callout-important-color);
    }

    .callout p {
      margin: 0.5rem 0;
    }

    /* ============================================================
       图片
       ============================================================ */
    img {
      max-width: 100%;
      height: auto;
      display: block;
      margin: 1rem 0;
      border-radius: 6px;
      cursor: pointer;
      transition: opacity 0.2s;
    }

    img:hover {
      opacity: 0.8;
    }

    /* ============================================================
       图片灯箱
       ============================================================ */
    .img-lightbox {
      display: none;
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: rgba(0, 0, 0, 0.8);
      z-index: 1000;
      align-items: center;
      justify-content: center;
      padding: 2rem;
    }

    .img-lightbox.visible {
      display: flex;
    }

    .img-lightbox img {
      max-width: 90vw;
      max-height: 90vh;
      margin: 0;
    }

    /* ============================================================
       回到顶部按钮
       ============================================================ */
    #back-to-top {
      position: fixed;
      bottom: 2rem;
      right: 2rem;
      width: 50px;
      height: 50px;
      background-color: var(--color-accent);
      color: white;
      border: none;
      border-radius: 50%;
      font-size: 20px;
      cursor: pointer;
      display: none;
      align-items: center;
      justify-content: center;
      z-index: 98;
      transition: all 0.3s ease;
      box-shadow: 0 4px 12px rgba(9, 105, 218, 0.3);
      opacity: 0;
      transform: scale(0.9);
    }

    #back-to-top.visible {
      display: flex;
      opacity: 1;
      transform: scale(1);
    }

    #back-to-top:hover {
      background-color: var(--color-link-hover);
      box-shadow: 0 6px 16px rgba(9, 105, 218, 0.4);
      transform: scale(1.05);
    }

    #back-to-top:active {
      transform: scale(0.97);
    }

    /* ============================================================
       全文搜索
       ============================================================ */
    #search-overlay {
      display: none;
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: rgba(0, 0, 0, 0.5);
      backdrop-filter: blur(8px);
      z-index: 1001;
      align-items: flex-start;
      justify-content: center;
      padding-top: 10vh;
    }

    #search-overlay.visible {
      display: flex;
    }

    .search-box {
      background-color: var(--color-bg);
      border-radius: 12px;
      width: 90%;
      max-width: 640px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      overflow: hidden;
      display: flex;
      flex-direction: column;
      max-height: 80vh;
    }

    .search-input {
      padding: 1.25rem 1.5rem;
      border: none;
      font-size: 16px;
      background-color: var(--color-bg);
      color: var(--color-text);
      outline: none;
      border-bottom: 1px solid var(--color-border);
      box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
    }

    .search-input::placeholder {
      color: var(--color-text-muted);
    }

    .search-results-list {
      overflow-y: auto;
      flex: 1;
      padding: 0.5rem;
    }

    .search-result {
      padding: 0.75rem 1rem;
      margin-bottom: 0.5rem;
      border-radius: 6px;
      cursor: pointer;
      transition: all 0.15s ease;
      border: 1px solid transparent;
    }

    .search-result:hover {
      background-color: var(--color-bg-alt);
      border-color: var(--color-accent);
    }

    .search-result-title {
      font-weight: 600;
      color: var(--color-heading);
      margin-bottom: 0.3rem;
    }

    .search-result-excerpt {
      font-size: 13px;
      color: var(--color-text-muted);
      line-height: 1.5;
      overflow: hidden;
      text-overflow: ellipsis;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
    }

    .search-result-excerpt mark {
      background-color: rgba(255, 235, 59, 0.3);
      color: var(--color-text);
      font-weight: 500;
    }

    .search-no-results {
      padding: 2rem 1rem;
      text-align: center;
      color: var(--color-text-muted);
    }

    .search-count-label {
      padding: 0.5rem 1rem;
      font-size: 12px;
      color: var(--color-text-muted);
      border-bottom: 1px solid var(--color-border);
    }

    /* ============================================================
       前后章导航
       ============================================================ */
    .chapter-nav {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1rem;
      margin-top: 3rem;
      padding-top: 2rem;
      border-top: 1px solid var(--color-border);
    }

    .chapter-nav a {
      padding: 1rem;
      border: 1px solid var(--color-border);
      border-radius: 6px;
      text-decoration: none;
      transition: background-color 0.2s, border-color 0.2s;
      display: flex;
      flex-direction: column;
      justify-content: flex-start;
    }

    .chapter-nav a:hover {
      border-color: var(--color-accent);
      background-color: var(--color-accent-light);
    }

    .nav-label {
      font-size: 12px;
      color: var(--color-text-muted);
      margin-bottom: 0.5rem;
    }

    .nav-title {
      color: var(--color-link);
      font-weight: 500;
    }

    .nav-prev {
      justify-self: start;
    }

    .nav-next {
      justify-self: end;
    }

    /* ============================================================
       响应式设计
       ============================================================ */
    @media (max-width: 1200px) {
      body {
        grid-template-columns: 250px 1fr;
      }

      #right-toc-nav {
        display: none;
      }
    }

    @media (max-width: 768px) {
      body {
        grid-template-columns: 1fr;
        grid-template-rows: 52px 1fr;
      }

      .toolbar {
        height: 52px;
      }

      #reading-progress {
        top: 52px;
      }

      #left-sidebar {
        position: fixed;
        left: -250px;
        top: 52px;
        height: calc(100vh - 52px);
        z-index: 99;
        transition: left 0.3s;
        width: 250px;
      }

      #left-sidebar.mobile-open {
        left: 0;
      }

      .sidebar-overlay {
        top: 52px;
      }

      .sidebar-overlay.visible {
        display: block;
      }

      #main-content {
        padding: 1.5rem 1rem;
      }

      .toolbar-btn.icon-only {
        padding: 0.5rem;
      }

      .chapter-title {
        font-size: 1.75rem;
      }

      .chapter-nav {
        grid-template-columns: 1fr;
      }

      #back-to-top {
        bottom: 1rem;
        right: 1rem;
        width: 45px;
        height: 45px;
        font-size: 18px;
      }
    }


    /* ============================================================
       减弱动画偏好设置
       ============================================================ */
    @media (prefers-reduced-motion: reduce) {
      * {
        animation-duration: 0.01ms !important;
        animation-iteration-count: 1 !important;
        transition-duration: 0.01ms !important;
      }

      html {
        scroll-behavior: auto !important;
      }
    }

    /* ============================================================
       打印样式
       ============================================================ */
    @media print {
      body {
        display: block;
        grid-template-columns: none;
        grid-template-rows: none;
        height: auto;
      }

      .toolbar,
      #left-sidebar,
      #right-toc-nav,
      .sidebar-overlay,
      #back-to-top,
      #search-overlay {
        display: none !important;
      }

      #main-content {
        grid-column: auto;
        grid-row: auto;
        padding: 0;
        margin: 0;
        max-width: 100%;
        width: 100%;
        overflow: visible;
        border: none;
      }

      h1, h2, h3, h4, h5, h6 {
        page-break-after: avoid;
        page-break-inside: avoid;
      }

      p {
        widows: 3;
        orphans: 3;
        page-break-inside: avoid;
      }

      a {
        color: #0969da;
        text-decoration: underline;
      }

      a::after {
        content: " (" attr(href) ")";
        font-size: 0.8em;
        color: #656d76;
      }

      .code-block,
      pre,
      blockquote {
        page-break-inside: avoid;
      }

      img {
        max-width: 100%;
        page-break-inside: avoid;
      }

      table {
        border-collapse: collapse;
        page-break-inside: avoid;
      }

      body {
        background-color: white !important;
        color: black !important;
      }

      * {
        background-color: white !important;
        color: black !important;
        box-shadow: none !important;
        text-shadow: none !important;
      }

      a, strong {
        color: black !important;
      }

      code, pre {
        background-color: #f6f8fa !important;
        color: #1f2328 !important;
      }

      .chapter-nav {
        border-top: 1px solid #d0d7de;
        page-break-inside: avoid;
      }
    }

    /* ============================================================
       自定义主题覆盖
       ============================================================ */
    {{.CSS}}
  `
