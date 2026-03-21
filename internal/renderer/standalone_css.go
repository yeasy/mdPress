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
      --color-bg:           #0d1117;
      --color-bg-alt:       #161b22;
      --color-bg-sidebar:   #161b22;
      --color-text:         #c9d1d9;
      --color-text-muted:   #8b949e;
      --color-heading:      #f0f6fc;
      --color-link:         #58a6ff;
      --color-link-hover:   #79c0ff;
      --color-border:       #30363d;
      --color-accent:       #58a6ff;
      --color-accent-light: rgba(88, 166, 255, 0.15);

      /* 代码块 */
      --color-code-bg:      #010409;
      --color-code-border:  #30363d;
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
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
      font-size: 16px;
      line-height: 1.6;
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
      padding: 0.5rem;
      border-radius: 6px;
      font-size: 14px;
      white-space: nowrap;
      transition: background-color 0.2s, color 0.2s;
    }

    .toolbar-btn:hover {
      background-color: var(--color-bg-alt);
      color: var(--color-link);
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

    .sidebar {
      grid-column: 1;
      grid-row: 2;
      background-color: var(--color-bg-sidebar);
      border-right: 1px solid var(--color-border);
      overflow-y: auto;
      padding: 1rem 0;
      max-height: calc(100vh - 60px);
    }

    .content-area {
      grid-column: 2;
      grid-row: 2;
      overflow-y: auto;
      padding: 2rem 3rem;
    }

    .toc {
      grid-column: 3;
      grid-row: 2;
      background-color: var(--color-bg);
      border-left: 1px solid var(--color-border);
      overflow-y: auto;
      padding: 1.5rem 1rem;
      max-height: calc(100vh - 60px);
      font-size: 13px;
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
    .sidebar {
      --sidebar-padding-left: 1rem;
    }

    .toc-group {
      margin: 0;
      padding: 0;
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
      padding: 0.25rem 0.5rem;
      border-radius: 4px;
      transition: background-color 0.15s, color 0.15s;
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .toc-link:hover {
      background-color: var(--color-sidebar-hover);
      color: var(--color-link);
    }

    .toc-link.active {
      color: var(--color-sidebar-active);
      background-color: var(--color-sidebar-active-bg);
      font-weight: 500;
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
      background: rgba(0, 0, 0, 0.5);
      z-index: 99;
    }

    .sidebar-overlay.active {
      display: block;
    }

    /* ============================================================
       主内容区域
       ============================================================ */
    .content-area {
      max-width: 900px;
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
    }

    h1 {
      font-size: 2rem;
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.3rem;
    }

    h2 {
      font-size: 1.75rem;
      border-bottom: 1px solid var(--color-border);
      padding-bottom: 0.2rem;
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
      font-family: "SF Mono", Monaco, "Cascadia Code", "Roboto Mono", Consolas, "Courier New", monospace;
      font-size: 0.9em;
      border: 1px solid var(--color-code-border);
    }

    pre {
      background-color: var(--color-code-bg);
      border: 1px solid var(--color-code-border);
      border-radius: 8px;
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
      background-color: var(--color-code-bg);
      padding: 0.25rem 0.5rem;
      border-radius: 3px;
    }

    .code-copy {
      position: absolute;
      top: 0.5rem;
      right: 0.5rem;
      background-color: var(--color-bg-alt);
      border: 1px solid var(--color-border);
      color: var(--color-text);
      border-radius: 4px;
      padding: 0.4rem 0.8rem;
      font-size: 12px;
      cursor: pointer;
      transition: background-color 0.2s, border-color 0.2s;
      z-index: 2;
    }

    .code-copy:hover {
      background-color: var(--color-accent-light);
      border-color: var(--color-accent);
      color: var(--color-accent);
    }

    .code-copy.copied {
      background-color: var(--color-accent-light);
      color: var(--color-accent);
      border-color: var(--color-accent);
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
      padding-left: 1rem;
      border-left: 3px solid var(--color-accent);
      background-color: var(--color-bg-alt);
      padding: 0.5rem 1rem;
      border-radius: 4px;
    }

    blockquote p {
      margin: 0.5rem 0;
    }

    /* ============================================================
       表格
       ============================================================ */
    table {
      width: 100%;
      border-collapse: collapse;
      margin: 1rem 0;
      border: 1px solid var(--color-table-header);
    }

    thead {
      background-color: var(--color-table-header);
    }

    th {
      padding: 0.75rem;
      text-align: left;
      font-weight: 600;
      color: var(--color-heading);
      border: 1px solid var(--color-border);
    }

    td {
      padding: 0.75rem;
      border: 1px solid var(--color-border);
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
      padding: 1rem;
      border-left: 4px solid;
      border-radius: 4px;
      background-color: var(--color-bg-alt);
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
      background-color: var(--callout-note-bg);
      color: var(--callout-note-color);
    }

    .callout-note .callout-title {
      color: var(--callout-note-color);
    }

    .callout-warning {
      border-color: var(--callout-warning-border);
      background-color: var(--callout-warning-bg);
      color: var(--callout-warning-color);
    }

    .callout-warning .callout-title {
      color: var(--callout-warning-color);
    }

    .callout-tip {
      border-color: var(--callout-tip-border);
      background-color: var(--callout-tip-bg);
      color: var(--callout-tip-color);
    }

    .callout-tip .callout-title {
      color: var(--callout-tip-color);
    }

    .callout-important {
      border-color: var(--callout-important-border);
      background-color: var(--callout-important-bg);
      color: var(--callout-important-color);
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

    .img-lightbox.active {
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
      transition: background-color 0.3s;
    }

    #back-to-top.show {
      display: flex;
    }

    #back-to-top:hover {
      background-color: var(--color-link-hover);
    }

    /* ============================================================
       全文搜索
       ============================================================ */
    .search-dialog {
      display: none;
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: rgba(0, 0, 0, 0.6);
      z-index: 1001;
      align-items: flex-start;
      justify-content: center;
      padding-top: 10vh;
    }

    .search-dialog.active {
      display: flex;
    }

    .search-box {
      background-color: var(--color-bg);
      border-radius: 8px;
      width: 90%;
      max-width: 600px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      overflow: hidden;
      display: flex;
      flex-direction: column;
      max-height: 80vh;
    }

    .search-input {
      padding: 1rem;
      border: none;
      font-size: 16px;
      background-color: var(--color-bg);
      color: var(--color-text);
      outline: none;
      border-bottom: 1px solid var(--color-border);
    }

    .search-input::placeholder {
      color: var(--color-text-muted);
    }

    .search-results {
      overflow-y: auto;
      flex: 1;
      padding: 0;
    }

    .search-result {
      padding: 1rem;
      border-bottom: 1px solid var(--color-border);
      cursor: pointer;
      transition: background-color 0.15s;
    }

    .search-result:hover {
      background-color: var(--color-bg-alt);
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

    .search-count {
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

      .toc {
        display: none;
      }
    }

    @media (max-width: 768px) {
      body {
        grid-template-columns: 1fr;
      }

      .sidebar {
        position: fixed;
        left: -250px;
        top: 60px;
        height: calc(100vh - 60px);
        z-index: 99;
        transition: left 0.3s;
        width: 250px;
      }

      .sidebar.active {
        left: 0;
      }

      .content-area {
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
       自定义主题覆盖
       ============================================================ */
    {{.CSS}}
  `
