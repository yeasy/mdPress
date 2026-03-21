package renderer

// standaloneJS contains the embedded JavaScript for interactive features
// of the standalone HTML renderer, including theme switching, navigation,
// search, code block enhancements, callouts, tables, lightbox, and more.
const standaloneJS = `
  'use strict';

  // ============================================================
  // 主题管理：三档切换（light / dark / system），无闪烁
  // ============================================================
  var THEME_KEY = 'mdpress-theme';
  var themeBtn  = document.getElementById('btn-theme');
  var themes    = ['light', 'dark', 'system'];
  var themeIcons  = { light: '☀️', dark: '🌙', system: '🖥' };
  var themeLabels = { light: '亮色', dark: '暗色', system: '跟随系统' };
  var currentTheme = localStorage.getItem(THEME_KEY) || 'system';

  function applyTheme(t) {
    currentTheme = t;
    try { localStorage.setItem(THEME_KEY, t); } catch(e) {}
    var dark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : '');
    themeBtn.textContent = themeIcons[t];
    themeBtn.title = '主题：' + themeLabels[t] + '（点击切换）';
  }

  themeBtn.addEventListener('click', function() {
    var idx = themes.indexOf(currentTheme);
    applyTheme(themes[(idx + 1) % themes.length]);
  });

  // 监听系统主题变化（仅在 system 模式下生效）
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
    if (currentTheme === 'system') applyTheme('system');
  });

  applyTheme(currentTheme);

  // ============================================================
  // 侧边栏控制（桌面推拉 + 移动端滑入）
  // ============================================================
  var leftSidebar   = document.getElementById('left-sidebar');
  var mainContent   = document.getElementById('main-content');
  var sidebarOverlay = document.getElementById('sidebar-overlay');
  var sidebarHidden = false;

  function isMobile() { return window.innerWidth <= 768; }

  function showSidebar() {
    sidebarHidden = false;
    if (isMobile()) {
      leftSidebar.classList.add('mobile-open');
      sidebarOverlay.classList.add('visible');
    } else {
      leftSidebar.classList.remove('sidebar-collapsed');
      mainContent.classList.remove('left-expanded');
    }
  }

  function hideSidebar() {
    sidebarHidden = true;
    if (isMobile()) {
      leftSidebar.classList.remove('mobile-open');
      sidebarOverlay.classList.remove('visible');
    } else {
      leftSidebar.classList.add('sidebar-collapsed');
      mainContent.classList.add('left-expanded');
    }
  }

  document.getElementById('btn-sidebar').addEventListener('click', function() {
    sidebarHidden ? showSidebar() : hideSidebar();
  });

  sidebarOverlay.addEventListener('click', hideSidebar);

  // 移动端点击链接后自动关闭侧边栏
  leftSidebar.addEventListener('click', function(e) {
    if (e.target.tagName === 'A' && isMobile()) hideSidebar();
  });

  // ============================================================
  // 左侧 TOC 折叠/展开（带过渡动画，支持多个章节同时展开）
  // ============================================================

  // 用户点击导航时设置此标志，抑制 scroll spy 的手风琴行为
  var navClickLock = false;
  var navClickTimer = null;
  function lockNavClick() {
    navClickLock = true;
    if (navClickTimer) clearTimeout(navClickTimer);
    navClickTimer = setTimeout(function() { navClickLock = false; }, 600);
  }

  // 判断是否为顶级章节组（父元素是 #sidebar-nav 本身）
  function isTopLevelGroup(item) {
    return item && item.parentElement && item.parentElement.id === 'sidebar-nav';
  }

  // 展开子章节列表（带 max-height 动画）
  function expandTocGroup(children, btn) {
    if (!children.hidden && children.style.maxHeight === '') return; // 已展开
    children.style.maxHeight = '0';
    children.removeAttribute('hidden');
    void children.offsetHeight;
    children.style.maxHeight = children.scrollHeight + 'px';
    if (btn) btn.setAttribute('aria-expanded', 'true');
    children.addEventListener('transitionend', function onEnd() {
      children.removeEventListener('transitionend', onEnd);
      if (!children.hidden) children.style.maxHeight = '';
    });
  }

  // 折叠子章节列表（带 max-height 动画）
  function collapseTocGroup(children, btn) {
    if (children.hidden) return; // 已折叠
    children.style.maxHeight = children.scrollHeight + 'px';
    void children.offsetHeight;
    children.style.maxHeight = '0';
    if (btn) btn.setAttribute('aria-expanded', 'false');
    children.addEventListener('transitionend', function onEnd() {
      children.removeEventListener('transitionend', onEnd);
      children.setAttribute('hidden', '');
      children.style.maxHeight = '';
    });
  }

  // 收起某节点的同级已展开顶级章节（仅顶级手风琴）
  function collapseTopLevelSiblings(item) {
    if (!isTopLevelGroup(item)) return;
    var parent = item.parentElement;
    parent.querySelectorAll(':scope > .toc-group.has-children').forEach(function(sib) {
      if (sib === item) return;
      var sc = sib.querySelector(':scope > .toc-children');
      var sb = sib.querySelector(':scope > .toc-row > .toc-toggle');
      if (sc && !sc.hidden) collapseTocGroup(sc, sb);
    });
  }

  document.querySelectorAll('.toc-toggle').forEach(function(btn) {
    btn.addEventListener('click', function(e) {
      e.stopPropagation();
      lockNavClick();
      var item = btn.closest('.toc-group');
      var children = item ? item.querySelector(':scope > .toc-children') : null;
      if (!children) return;
      var expanded = btn.getAttribute('aria-expanded') === 'true';
      if (expanded) {
        collapseTocGroup(children, btn);
      } else {
        // 仅顶级章节互斥折叠，子章节可同时展开
        collapseTopLevelSiblings(item);
        expandTocGroup(children, btn);
      }
    });
  });

  // ============================================================
  // 平滑滚动：拦截 TOC 链接和章节导航按钮的点击，防止页面闪烁
  // ============================================================
  function handleAnchorClick(e) {
    var href = this.getAttribute('href');
    if (!href || href.charAt(0) !== '#') return;
    var targetId = href.slice(1);
    if (!document.getElementById(targetId)) return;
    e.preventDefault();
    lockNavClick(); // 抑制滚动期间 scroll spy 的手风琴
    // 使用 JS 平滑滚动，避免浏览器默认的瞬间跳转（闪烁）
    document.getElementById(targetId).scrollIntoView({ behavior: 'smooth', block: 'start' });
    // 更新地址栏 hash，不触发浏览器默认跳转
    if (history.pushState) history.pushState(null, '', href);
  }

  function initSmoothNav() {
    // 侧边栏目录链接
    document.querySelectorAll('#sidebar-nav .toc-link').forEach(function(link) {
      link.addEventListener('click', handleAnchorClick);
    });
    // 上一页/下一页章节导航按钮
    document.querySelectorAll('.chapter-nav-btn').forEach(function(link) {
      link.addEventListener('click', handleAnchorClick);
    });
  }

  // ============================================================
  // Scroll Spy: highlight left TOC + update right TOC.
  // Uses IntersectionObserver so updates fire only when elements enter or
  // leave the observation zone, eliminating per-frame flicker that occurs
  // with scroll-event polling during smooth navigation.
  // ============================================================
  var activeChapterId = '';
  var activeHeadingId = '';

  function initScrollSpy() {
    var chapters = Array.from(document.querySelectorAll('.chapter'));
    var headings = Array.from(
      document.querySelectorAll('.chapter-content h1[id], .chapter-content h2[id], .chapter-content h3[id], .chapter-content h4[id]')
    );

    // Pre-map each heading to its parent chapter id for O(1) lookup.
    headings.forEach(function(h) {
      h._chapterId = h.closest('.chapter') ? h.closest('.chapter').id : '';
    });

    // Visibility state: true when element is inside the observation zone.
    var visibleHeadings = {};
    var visibleChapters = {};

    // Determine which chapter/heading is currently active and push updates
    // to the left and right TOC components only when the state actually changes.
    function syncActive() {
      // The topmost visible heading (first in DOM order) wins.
      var newHeadingId = '';
      var newChapterId = '';
      for (var i = 0; i < headings.length; i++) {
        if (visibleHeadings[headings[i].id]) {
          newHeadingId = headings[i].id;
          newChapterId = headings[i]._chapterId;
          break;
        }
      }
      // No heading in zone — fall back to the topmost visible chapter.
      if (!newChapterId) {
        for (var j = 0; j < chapters.length; j++) {
          if (visibleChapters[chapters[j].id]) { newChapterId = chapters[j].id; break; }
        }
        if (!newChapterId && chapters.length > 0) newChapterId = chapters[0].id;
      }

      if (newChapterId !== activeChapterId || newHeadingId !== activeHeadingId) {
        activeChapterId = newChapterId;
        activeHeadingId = newHeadingId;
        updateLeftTOC(newChapterId, newHeadingId);
        updateRightTOC(newChapterId, newHeadingId);
      }
    }

    // Observe headings in a band from 80 px below the viewport top (below
    // the fixed toolbar) down to 50 % up from the bottom.  The observer
    // fires only on entry/exit — not on every scroll frame.
    var headingObserver = new IntersectionObserver(function(entries) {
      entries.forEach(function(e) { visibleHeadings[e.target.id] = e.isIntersecting; });
      syncActive();
    }, { rootMargin: '-80px 0px -50% 0px', threshold: 0 });

    headings.forEach(function(h) { headingObserver.observe(h); });

    // Observe chapters with a wider band to handle chapters that have no headings.
    var chapterObserver = new IntersectionObserver(function(entries) {
      entries.forEach(function(e) { visibleChapters[e.target.id] = e.isIntersecting; });
      syncActive();
    }, { rootMargin: '-80px 0px -20% 0px', threshold: 0 });

    chapters.forEach(function(ch) { chapterObserver.observe(ch); });
  }

  // 更新左侧 TOC 高亮
  function updateLeftTOC(chapterId, headingId) {
    var activeTarget = headingId || chapterId;
    document.querySelectorAll('#sidebar-nav .toc-link').forEach(function(link) {
      var target = link.getAttribute('data-target');
      link.classList.toggle('active', target === activeTarget);
    });

    // 展开包含活跃链接的章节组
    var activeLink = document.querySelector('#sidebar-nav .toc-link.active');
    if (activeLink) {
      var group = activeLink.closest('.toc-group');
      while (group) {
        var toggle = group.querySelector(':scope > .toc-row > .toc-toggle');
        var children = group.querySelector(':scope > .toc-children');
        if (toggle && children && children.hidden) {
          // 仅在非点击导航期间才做顶级手风琴折叠
          if (!navClickLock) {
            collapseTopLevelSiblings(group);
          }
          expandTocGroup(children, toggle);
        }
        var parent = group.parentElement;
        group = parent ? parent.closest('.toc-group') : null;
      }

      // 确保活跃链接在侧边栏可视区域内
      activeLink.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }

  // 右侧页内 TOC 缓存（避免每帧重建 DOM）
  var rightTOCCache = {};
  var currentRightChapter = '';

  function updateRightTOC(chapterId, headingId) {
    var rightNav = document.getElementById('right-toc-nav');
    if (!rightNav) return;

    // 章节切换时重新构建右侧 TOC 链接列表
    if (chapterId !== currentRightChapter) {
      currentRightChapter = chapterId;
      if (!rightTOCCache[chapterId]) {
        var chapter = document.getElementById(chapterId);
        rightTOCCache[chapterId] = chapter
          ? Array.from(chapter.querySelectorAll('.chapter-content h1[id], .chapter-content h2[id], .chapter-content h3[id]'))
              .map(function(h) { return { id: h.id, text: h.textContent, level: parseInt(h.tagName.slice(1)) }; })
          : [];
      }
      rightNav.innerHTML = '';
      rightTOCCache[chapterId].forEach(function(item) {
        var a = document.createElement('a');
        a.href = '#' + item.id;
        a.className = 'right-toc-link rtoc-d' + (item.level - 1);
        a.setAttribute('data-target', item.id);
        a.textContent = item.text;
        // Use smooth scroll + history.pushState (same behaviour as left TOC links).
        a.addEventListener('click', handleAnchorClick);
        rightNav.appendChild(a);
      });
    }

    // 高亮当前标题
    rightNav.querySelectorAll('.right-toc-link').forEach(function(link) {
      link.classList.toggle('active', link.getAttribute('data-target') === headingId);
    });
  }

  // onScroll only handles the reading progress bar and the back-to-top button.
  // Chapter/heading tracking is now handled by IntersectionObserver in initScrollSpy.
  function onScroll() {
    var scrollTop = window.scrollY || document.documentElement.scrollTop;
    var docH = document.documentElement.scrollHeight - window.innerHeight;
    var pct  = docH > 0 ? Math.min(100, (scrollTop / docH) * 100) : 0;
    document.getElementById('reading-progress').style.width = pct + '%';
    document.getElementById('back-to-top').classList.toggle('visible', scrollTop > 400);
  }

  // Throttle scroll events with requestAnimationFrame to avoid excessive repaints.
  var rafPending = false;
  window.addEventListener('scroll', function() {
    if (rafPending) return;
    rafPending = true;
    requestAnimationFrame(function() { onScroll(); rafPending = false; });
  }, { passive: true });

  // ============================================================
  // 代码块增强：自动包装 pre > code，添加语言标签和复制按钮
  // ============================================================
  function enhanceCodeBlocks() {
    document.querySelectorAll('.chapter-content pre').forEach(function(pre) {
      // 已处理过的跳过（幂等）
      if (pre.parentElement && pre.parentElement.classList.contains('code-block-wrapper')) return;

      var code = pre.querySelector('code');
      var lang = '';
      if (code) {
        Array.from(code.classList).some(function(cls) {
          var m = cls.match(/^language-(.+)$/);
          if (m) { lang = m[1]; return true; }
          return false;
        });
      }

      // 创建包装容器
      var wrapper = document.createElement('div');
      wrapper.className = 'code-block-wrapper';

      // 头部：语言标签 + 复制按钮
      var header = document.createElement('div');
      header.className = 'code-block-header';

      var langLabel = document.createElement('span');
      langLabel.className = 'code-lang-label';
      langLabel.textContent = lang || 'text';

      var copyBtn = document.createElement('button');
      copyBtn.className = 'code-copy-btn';
      copyBtn.textContent = '复制';
      copyBtn.title = '复制代码';
      copyBtn.setAttribute('aria-label', '复制代码');

      // 复制逻辑（优先 navigator.clipboard，降级 execCommand）
      copyBtn.addEventListener('click', function() {
        var text = code ? code.textContent : pre.textContent;
        var doFeedback = function() {
          copyBtn.textContent = '已复制 ✓';
          copyBtn.classList.add('copied');
          setTimeout(function() {
            copyBtn.textContent = '复制';
            copyBtn.classList.remove('copied');
          }, 2000);
        };
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(doFeedback).catch(function() {
            fallbackCopy(text, doFeedback);
          });
        } else {
          fallbackCopy(text, doFeedback);
        }
      });

      header.appendChild(langLabel);
      header.appendChild(copyBtn);
      wrapper.appendChild(header);

      // 将原 pre 移入 wrapper
      pre.parentNode.insertBefore(wrapper, pre);
      wrapper.appendChild(pre);
    });
  }

  // execCommand 降级复制（用于不支持 Clipboard API 的环境）
  function fallbackCopy(text, cb) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.cssText = 'position:fixed;top:-9999px;opacity:0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch(e) {}
    document.body.removeChild(ta);
    if (cb) cb();
  }

  // ============================================================
  // Callout 提示框：将特定格式的 blockquote 转换为彩色提示框
  //
  // 支持格式（Markdown 中）：
  //   > **Note**: 内容
  //   > **Warning**: 内容
  //   > **Tip**: 内容
  //   > **Important**: 内容
  // ============================================================
  var CALLOUT_MAP = {
    'Note':      { type: 'note',      icon: 'ℹ️' },
    'Warning':   { type: 'warning',   icon: '⚠️' },
    'Tip':       { type: 'tip',       icon: '💡' },
    'Important': { type: 'important', icon: '❗' },
    'Danger':    { type: 'important', icon: '🚨' },
    '注意':      { type: 'note',      icon: 'ℹ️' },
    '警告':      { type: 'warning',   icon: '⚠️' },
    '提示':      { type: 'tip',       icon: '💡' },
    '重要':      { type: 'important', icon: '❗' },
  };

  function transformCallouts() {
    document.querySelectorAll('.chapter-content blockquote').forEach(function(bq) {
      var firstP = bq.querySelector('p:first-child');
      if (!firstP) return;

      var firstStrong = firstP.querySelector('strong:first-child');
      if (!firstStrong) return;

      var keyword = firstStrong.textContent.replace(/:$/, '').trim();
      var info    = CALLOUT_MAP[keyword];
      if (!info)  return;

      // 构建 callout 容器
      var callout = document.createElement('div');
      callout.className = 'callout callout-' + info.type;

      var icon = document.createElement('span');
      icon.className = 'callout-icon';
      icon.setAttribute('aria-hidden', 'true');
      icon.textContent = info.icon;

      var body = document.createElement('div');
      body.className = 'callout-body';

      // 清理 strong 标签和紧跟的冒号/空格
      firstStrong.remove();
      var firstTextNode = firstP.firstChild;
      if (firstTextNode && firstTextNode.nodeType === 3) {
        firstTextNode.textContent = firstTextNode.textContent.replace(/^[:\s]+/, '');
      }

      // 将 blockquote 中的内容移入 body
      while (bq.firstChild) body.appendChild(bq.firstChild);

      callout.appendChild(icon);
      callout.appendChild(body);
      bq.parentNode.replaceChild(callout, bq);
    });
  }

  // ============================================================
  // 表格包装：使宽表格可横向滚动
  // ============================================================
  function wrapTables() {
    document.querySelectorAll('.chapter-content table').forEach(function(table) {
      if (table.parentElement && table.parentElement.classList.contains('table-wrapper')) return;
      var wrapper = document.createElement('div');
      wrapper.className = 'table-wrapper';
      table.parentNode.insertBefore(wrapper, table);
      wrapper.appendChild(table);
    });
  }

  // ============================================================
  // 图片灯箱：点击图片全屏查看
  // ============================================================
  var lightbox    = document.getElementById('img-lightbox');
  var lightboxImg = document.getElementById('img-lightbox-src');

  function openLightbox(src, alt) {
    lightboxImg.src = src;
    lightboxImg.alt = alt || '';
    lightbox.classList.add('visible');
    document.body.style.overflow = 'hidden';
  }

  function closeLightbox() {
    lightbox.classList.remove('visible');
    document.body.style.overflow = '';
    // 延迟清空 src，避免图片闪烁
    setTimeout(function() { if (!lightbox.classList.contains('visible')) lightboxImg.src = ''; }, 300);
  }

  lightbox.addEventListener('click', function(e) {
    if (e.target !== lightboxImg) closeLightbox();
  });

  function initLightbox() {
    document.querySelectorAll('.chapter-content img').forEach(function(img) {
      img.addEventListener('click', function() { openLightbox(img.src, img.alt); });
    });
  }

  // ============================================================
  // 全文搜索（⌘K / Ctrl+K 打开模态框，支持中文）
  // ============================================================
  var searchOverlay     = document.getElementById('search-overlay');
  var searchInput       = document.getElementById('search-input');
  var searchResultsList = document.getElementById('search-results-list');
  var searchCountLabel  = document.getElementById('search-count-label');
  var searchFocusIdx    = -1;

  function openSearch() {
    searchOverlay.classList.add('visible');
    searchInput.focus();
    searchInput.select();
  }

  function closeSearch() {
    searchOverlay.classList.remove('visible');
    searchFocusIdx = -1;
  }

  document.getElementById('btn-search').addEventListener('click', openSearch);

  searchOverlay.addEventListener('click', function(e) {
    if (e.target === searchOverlay) closeSearch();
  });

  document.addEventListener('keydown', function(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      searchOverlay.classList.contains('visible') ? closeSearch() : openSearch();
      return;
    }
    if (e.key === 'Escape') {
      if (lightbox.classList.contains('visible')) { closeLightbox(); return; }
      closeSearch();
      return;
    }
    if (!searchOverlay.classList.contains('visible')) return;
    if (e.key === 'ArrowDown') { e.preventDefault(); moveFocus(1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); moveFocus(-1); }
    else if (e.key === 'Enter') { e.preventDefault(); activateFocused(); }
  });

  function moveFocus(delta) {
    var items = searchResultsList.querySelectorAll('.search-result');
    if (!items.length) return;
    searchFocusIdx = Math.max(0, Math.min(items.length - 1, searchFocusIdx + delta));
    items.forEach(function(item, i) { item.classList.toggle('focused', i === searchFocusIdx); });
  }

  function activateFocused() {
    var item = searchResultsList.querySelector('.search-result.focused');
    if (item) {
      scrollToId(item.getAttribute('data-target'));
      closeSearch();
    }
  }

  function scrollToId(id) {
    var el = document.getElementById(id);
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  function escapeRe(s) { return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'); }

  var searchTimer = null;
  searchInput.addEventListener('input', function() {
    searchFocusIdx = -1;
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = setTimeout(doSearch, 200);
  });

  function doSearch() {
    var query = searchInput.value.trim();
    searchResultsList.innerHTML = '';
    if (!query) { searchCountLabel.textContent = ''; return; }

    var re = new RegExp(escapeRe(query), 'gi');
    var results = [];

    document.querySelectorAll('.chapter').forEach(function(chapter) {
      if (results.length >= 50) return;
      var content = chapter.querySelector('.chapter-content');
      if (!content) return;

      var walker = document.createTreeWalker(content, NodeFilter.SHOW_TEXT, {
        acceptNode: function(node) {
          var tag = node.parentElement ? node.parentElement.tagName : '';
          if (tag === 'SCRIPT' || tag === 'STYLE') return NodeFilter.FILTER_REJECT;
          return NodeFilter.FILTER_ACCEPT;
        }
      });

      var seen = new Set();
      var node;
      while ((node = walker.nextNode()) && results.length < 50) {
        var text = node.textContent;
        if (!re.test(text)) { re.lastIndex = 0; continue; }
        re.lastIndex = 0;

        var match;
        while ((match = re.exec(text)) !== null && results.length < 50) {
          var s = Math.max(0, match.index - 40);
          var e = Math.min(text.length, match.index + query.length + 40);
          var excerpt = (s > 0 ? '…' : '') + text.slice(s, e) + (e < text.length ? '…' : '');

          // 找最近标题作为结果标题
          var nearH = node.parentElement ? node.parentElement.closest('h1,h2,h3,h4') : null;
          var itemTitle  = nearH ? nearH.textContent : (chapter.querySelector('.chapter-content h1,h2,h3') || {textContent: chapter.id}).textContent;
          var targetId   = nearH ? nearH.id : chapter.id;

          var key = targetId + '|' + excerpt.slice(0, 20);
          if (!seen.has(key)) {
            seen.add(key);
            results.push({ title: itemTitle, excerpt: excerpt, targetId: targetId });
          }
        }
        re.lastIndex = 0;
      }
    });

    searchCountLabel.textContent = results.length + ' 条结果' + (results.length >= 50 ? '（前 50 条）' : '');

    if (!results.length) {
      var q = query.replace(/</g, '&lt;');
      searchResultsList.innerHTML = '<div class="search-no-results">未找到与 "' + q + '" 相关的内容</div>';
      return;
    }

    var re2 = new RegExp('(' + escapeRe(query) + ')', 'gi');
    results.forEach(function(r, i) {
      var div = document.createElement('div');
      div.className = 'search-result';
      div.setAttribute('data-target', r.targetId);

      var title = document.createElement('div');
      title.className = 'search-result-title';
      title.textContent = r.title;

      var excerpt = document.createElement('div');
      excerpt.className = 'search-result-excerpt';
      excerpt.innerHTML = r.excerpt.replace(/</g, '&lt;').replace(re2, '<mark>$1</mark>');

      div.appendChild(title);
      div.appendChild(excerpt);
      div.addEventListener('click', function() {
        scrollToId(r.targetId);
        closeSearch();
      });
      searchResultsList.appendChild(div);
    });
  }

  // ============================================================
  // 回到顶部按钮
  // ============================================================
  document.getElementById('back-to-top').addEventListener('click', function() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });

  // ============================================================
  // 初始化：DOM 就绪后执行所有增强操作
  // ============================================================
  function init() {
    initScrollSpy();
    initSmoothNav();
    enhanceCodeBlocks();
    transformCallouts();
    wrapTables();
    initLightbox();
    onScroll(); // 初始化进度条和 TOC 高亮
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  `
