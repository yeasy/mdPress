package renderer

// standaloneJS contains the embedded JavaScript for interactive features
// of the standalone HTML renderer, including theme switching, navigation,
// search, code block enhancements, callouts, tables, lightbox, and more.
const standaloneJS = `
  'use strict';

  // ============================================================
  // Theme Management: three-way toggle (light / dark / system)
  // with smooth icon rotation animation
  // ============================================================
  var THEME_KEY = 'mdpress-theme';
  var themeBtn  = document.getElementById('btn-theme');
  var themes    = ['light', 'dark', 'system'];

  var themeSvgs = {
    light: '<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><circle cx="12" cy="12" r="5"/><path d="M12 1v6m0 6v6M23 12h-6m-6 0H1M20.485 3.515l-4.243 4.243m0 5.484l4.243 4.243M3.515 3.515l4.243 4.243m0 5.484l-4.243 4.243"/></svg>',
    dark: '<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>',
    system: '<svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>'
  };

  var themeLabels = { light: 'Light', dark: 'Dark', system: 'System' };
  var currentTheme = localStorage.getItem(THEME_KEY) || 'system';

  function applyTheme(t) {
    currentTheme = t;
    try { localStorage.setItem(THEME_KEY, t); } catch(e) {}
    var dark = t === 'dark' || (t === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    document.documentElement.setAttribute('data-theme', dark ? 'dark' : '');
    themeBtn.innerHTML = themeSvgs[t];
    themeBtn.title = 'Theme: ' + themeLabels[t] + ' (click to toggle)';
    themeBtn.classList.add('icon-rotating');
    setTimeout(function() { themeBtn.classList.remove('icon-rotating'); }, 300);
  }

  themeBtn.addEventListener('click', function() {
    var idx = themes.indexOf(currentTheme);
    applyTheme(themes[(idx + 1) % themes.length]);
  });

  // Listen to system theme changes (only effective in system mode)
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function() {
    if (currentTheme === 'system') applyTheme('system');
  });

  applyTheme(currentTheme);

  // ============================================================
  // Sidebar control (desktop collapse/expand + mobile slide-in)
  // with scroll position memory for chapter navigation
  // ============================================================
  var leftSidebar   = document.getElementById('left-sidebar');
  var mainContent   = document.getElementById('main-content');
  var sidebarOverlay = document.getElementById('sidebar-overlay');
  var sidebarHidden = false;
  var SIDEBAR_SCROLL_PREFIX = 'mdpress-sidebar-scroll-';

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

  function saveSidebarScroll() {
    try {
      var scrollPos = leftSidebar.querySelector('#sidebar-nav').scrollTop;
      localStorage.setItem(SIDEBAR_SCROLL_PREFIX + 'current', scrollPos);
    } catch(e) {}
  }

  function restoreSidebarScroll() {
    try {
      var scrollPos = localStorage.getItem(SIDEBAR_SCROLL_PREFIX + 'current');
      if (scrollPos !== null) {
        leftSidebar.querySelector('#sidebar-nav').scrollTop = parseInt(scrollPos, 10);
      }
    } catch(e) {}
  }

  document.getElementById('btn-sidebar').addEventListener('click', function() {
    sidebarHidden ? showSidebar() : hideSidebar();
  });

  sidebarOverlay.addEventListener('click', hideSidebar);

  // Auto-close sidebar on mobile when link is clicked
  leftSidebar.addEventListener('click', function(e) {
    if (e.target.tagName === 'A' && isMobile()) {
      saveSidebarScroll();
      hideSidebar();
    }
  });

  // Save sidebar scroll position periodically and on unload
  setInterval(saveSidebarScroll, 1000);
  window.addEventListener('beforeunload', saveSidebarScroll);

  // Restore sidebar scroll on page load
  restoreSidebarScroll();

  // ============================================================
  // Left TOC collapse/expand with transition animations
  // Supports multiple expanded sections simultaneously
  // ============================================================

  // Set flag when user clicks navigation to suppress scroll spy accordion
  var navClickLock = false;
  var navClickTimer = null;
  function lockNavClick() {
    navClickLock = true;
    if (navClickTimer) clearTimeout(navClickTimer);
    navClickTimer = setTimeout(function() { navClickLock = false; }, 600);
  }

  // Expand subsection list with max-height animation
  function expandTocGroup(children, btn) {
    if (!children.hidden && children.style.maxHeight === '') return; // already expanded
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

  // Collapse subsection list with max-height animation
  function collapseTocGroup(children, btn) {
    if (children.hidden) return; // already collapsed
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

  // Collapse sibling groups at any level (accordion behavior)
  function collapseSiblingGroups(item) {
    if (!item || !item.parentElement) return;
    var container = item.parentElement;
    container.querySelectorAll(':scope > .toc-group.has-children').forEach(function(sib) {
      if (sib === item) return;
      var sc = sib.querySelector(':scope > .toc-children');
      var sb = sib.querySelector(':scope > .toc-row > .toc-toggle');
      if (sc && !sc.hidden) {
        // Also collapse any expanded children within the sibling
        sib.querySelectorAll('.toc-group.has-children').forEach(function(nested) {
          var nc = nested.querySelector(':scope > .toc-children');
          var nb = nested.querySelector(':scope > .toc-row > .toc-toggle');
          if (nc && !nc.hidden) collapseTocGroup(nc, nb);
        });
        collapseTocGroup(sc, sb);
      }
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
        collapseSiblingGroups(item);
        expandTocGroup(children, btn);
      }
    });
  });

  // ============================================================
  // Smooth scroll: intercept TOC links and chapter nav clicks
  // with offset for sticky header and highlight flash effect
  // ============================================================
  function handleAnchorClick(e) {
    var href = this.getAttribute('href');
    if (!href || href.charAt(0) !== '#') return;
    var targetId = href.slice(1);
    var targetEl = document.getElementById(targetId);
    if (!targetEl) return;
    e.preventDefault();
    lockNavClick(); // suppress accordion during scroll

    // Smooth scroll with offset for sticky header (80px)
    var offsetY = 80;
    var elementPosition = targetEl.getBoundingClientRect().top + window.scrollY - offsetY;
    window.scrollTo({ top: elementPosition, behavior: 'smooth' });

    // Update address bar hash without triggering default jump
    if (history.pushState) history.pushState(null, '', href);

    // Add brief highlight flash after scroll completes
    setTimeout(function() {
      targetEl.classList.add('highlight-flash');
      setTimeout(function() {
        targetEl.classList.remove('highlight-flash');
      }, 1500);
    }, 300);
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

  // Update left TOC highlight
  function updateLeftTOC(chapterId, headingId) {
    var activeTarget = headingId || chapterId;
    document.querySelectorAll('#sidebar-nav .toc-link').forEach(function(link) {
      var target = link.getAttribute('data-target');
      link.classList.toggle('active', target === activeTarget);
    });

    // Expand section groups containing active links
    var activeLink = document.querySelector('#sidebar-nav .toc-link.active');
    if (activeLink) {
      var group = activeLink.closest('.toc-group');
      while (group) {
        var toggle = group.querySelector(':scope > .toc-row > .toc-toggle');
        var children = group.querySelector(':scope > .toc-children');
        if (toggle && children && children.hidden) {
          // Collapse sibling sections during non-click navigation
          if (!navClickLock) {
            collapseSiblingGroups(group);
          }
          expandTocGroup(children, toggle);
        }
        var parent = group.parentElement;
        group = parent ? parent.closest('.toc-group') : null;
      }

      // Ensure active link is visible in sidebar
      activeLink.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    }
  }

  // Right TOC cache (avoid rebuilding DOM every frame)
  var rightTOCCache = {};
  var currentRightChapter = '';

  function updateRightTOC(chapterId, headingId) {
    var rightNav = document.getElementById('right-toc-nav');
    if (!rightNav) return;

    // Rebuild right TOC link list when chapter changes
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
        // Use smooth scroll + history.pushState (same as left TOC links)
        a.addEventListener('click', handleAnchorClick);
        rightNav.appendChild(a);
      });
    }

    // Highlight current heading
    rightNav.querySelectorAll('.right-toc-link').forEach(function(link) {
      link.classList.toggle('active', link.getAttribute('data-target') === headingId);
    });
  }

  // onScroll handles reading progress bar (GPU-accelerated with transform)
  // and back-to-top button. Chapter/heading tracking uses IntersectionObserver.
  function onScroll() {
    var scrollTop = window.scrollY || document.documentElement.scrollTop;
    var docH = document.documentElement.scrollHeight - window.innerHeight;
    var pct  = docH > 0 ? Math.min(100, (scrollTop / docH) * 100) : 0;

    // Use transform scaleX for GPU acceleration instead of width
    var progressBar = document.getElementById('reading-progress');
    progressBar.style.transform = 'scaleX(' + (pct / 100) + ')';
    progressBar.style.transformOrigin = '0 50%';

    // Show back-to-top only after scrolling past 300px
    document.getElementById('back-to-top').classList.toggle('visible', scrollTop > 300);
  }

  // Throttle scroll events with requestAnimationFrame to avoid excessive repaints.
  var rafPending = false;
  window.addEventListener('scroll', function() {
    if (rafPending) return;
    rafPending = true;
    requestAnimationFrame(function() { onScroll(); rafPending = false; });
  }, { passive: true });

  // ============================================================
  // Code block enhancement: auto-wrap pre > code, add language
  // labels and copy button with icon and feedback animations
  // ============================================================
  function enhanceCodeBlocks() {
    document.querySelectorAll('.chapter-content pre').forEach(function(pre) {
      // Skip already processed blocks (idempotent)
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

      // Create wrapper container
      var wrapper = document.createElement('div');
      wrapper.className = 'code-block-wrapper';

      // Header: language label + copy button
      var header = document.createElement('div');
      header.className = 'code-block-header';

      var langLabel = document.createElement('span');
      langLabel.className = 'code-lang-label';
      langLabel.textContent = lang || 'text';

      var copyBtn = document.createElement('button');
      copyBtn.className = 'code-copy-btn';
      copyBtn.title = 'Copy code';
      copyBtn.setAttribute('aria-label', 'Copy code');

      // SVG clipboard icon
      var clipboardSvg = '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>';
      var checkmarkSvg = '<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/></svg>';

      copyBtn.innerHTML = clipboardSvg;

      // Copy logic (prefer navigator.clipboard, fallback to execCommand)
      copyBtn.addEventListener('click', function() {
        var text = code ? code.textContent : pre.textContent;
        var doFeedback = function() {
          copyBtn.innerHTML = checkmarkSvg;
          copyBtn.classList.add('copied');
          copyBtn.classList.add('flash-success');
          setTimeout(function() {
            copyBtn.innerHTML = clipboardSvg;
            copyBtn.classList.remove('copied');
            copyBtn.classList.remove('flash-success');
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

      // Move original pre into wrapper
      pre.parentNode.insertBefore(wrapper, pre);
      wrapper.appendChild(pre);
    });
  }

  // execCommand fallback copy (for environments without Clipboard API)
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
  // Callout boxes: convert specific blockquote formats to colored callouts
  //
  // Supported formats (in Markdown):
  //   > **Note**: content
  //   > **Warning**: content
  //   > **Tip**: content
  //   > **Important**: content
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

      // Build callout container
      var callout = document.createElement('div');
      callout.className = 'callout callout-' + info.type;

      var icon = document.createElement('span');
      icon.className = 'callout-icon';
      icon.setAttribute('aria-hidden', 'true');
      icon.textContent = info.icon;

      var body = document.createElement('div');
      body.className = 'callout-body';

      // Clean up strong tag and following colons/spaces
      firstStrong.remove();
      var firstTextNode = firstP.firstChild;
      if (firstTextNode && firstTextNode.nodeType === 3) {
        firstTextNode.textContent = firstTextNode.textContent.replace(/^[:\s]+/, '');
      }

      // Move blockquote content into body
      while (bq.firstChild) body.appendChild(bq.firstChild);

      callout.appendChild(icon);
      callout.appendChild(body);
      bq.parentNode.replaceChild(callout, bq);
    });
  }

  // ============================================================
  // Expandable sections: enhance <details> and <summary> elements
  // with smooth open/close animations and chevron indicator
  // ============================================================
  function styleExpandableSections() {
    document.querySelectorAll('.chapter-content details').forEach(function(details) {
      if (details.classList.contains('expandable-styled')) return;
      details.classList.add('expandable-styled');

      var summary = details.querySelector('summary');
      if (!summary) return;

      summary.classList.add('expandable-header');

      // Add chevron indicator if not already present
      if (!summary.querySelector('.expandable-chevron')) {
        var chevron = document.createElement('span');
        chevron.className = 'expandable-chevron';
        chevron.setAttribute('aria-hidden', 'true');
        chevron.textContent = '▸';
        summary.insertBefore(chevron, summary.firstChild);
      }

      // Handle open/close animation
      details.addEventListener('toggle', function() {
        if (details.open) {
          summary.setAttribute('aria-expanded', 'true');
          summary.classList.add('open');
        } else {
          summary.setAttribute('aria-expanded', 'false');
          summary.classList.remove('open');
        }
      });

      summary.setAttribute('aria-expanded', details.open ? 'true' : 'false');
      if (details.open) summary.classList.add('open');
    });
  }

  // ============================================================
  // Table wrapping: enable horizontal scroll for wide tables
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
  // Image lightbox with zoom animation, background blur,
  // keyboard navigation (Escape, arrow keys), and focus trap
  // ============================================================
  var lightbox    = document.getElementById('img-lightbox');
  var lightboxImg = document.getElementById('img-lightbox-src');
  var allImages   = [];
  var currentImageIndex = -1;
  var lightboxPrevFocus = null;

  // Get all focusable elements within lightbox
  function getLightboxFocusableElements() {
    var focusableSelector = 'button, a, [tabindex]:not([tabindex="-1"])';
    var focusables = lightbox.querySelectorAll(focusableSelector);
    return Array.from(focusables).filter(function(el) {
      return el.offsetParent !== null && getComputedStyle(el).visibility !== 'hidden';
    });
  }

  function openLightbox(src, alt) {
    lightboxImg.src = src;
    lightboxImg.alt = alt || '';
    lightbox.classList.add('visible');
    lightbox.classList.add('zoom-in');
    document.body.style.overflow = 'hidden';
    lightboxPrevFocus = document.activeElement;

    // Track current image for keyboard navigation
    currentImageIndex = allImages.findIndex(function(img) { return img.src === src; });
  }

  function closeLightbox() {
    lightbox.classList.remove('visible');
    lightbox.classList.remove('zoom-in');
    document.body.style.overflow = '';
    currentImageIndex = -1;
    if (lightboxPrevFocus && typeof lightboxPrevFocus.focus === 'function') {
      lightboxPrevFocus.focus();
    }
    // Delay clearing src to avoid flicker
    setTimeout(function() { if (!lightbox.classList.contains('visible')) lightboxImg.src = ''; }, 300);
  }

  function showNextImage() {
    if (allImages.length === 0) return;
    currentImageIndex = (currentImageIndex + 1) % allImages.length;
    openLightbox(allImages[currentImageIndex].src, allImages[currentImageIndex].alt);
  }

  function showPrevImage() {
    if (allImages.length === 0) return;
    currentImageIndex = (currentImageIndex - 1 + allImages.length) % allImages.length;
    openLightbox(allImages[currentImageIndex].src, allImages[currentImageIndex].alt);
  }

  lightbox.addEventListener('click', function(e) {
    if (e.target !== lightboxImg) closeLightbox();
  });

  // Focus trap for lightbox: handle Tab key to cycle within lightbox
  lightbox.addEventListener('keydown', function(e) {
    if (e.key !== 'Tab') return;
    if (!lightbox.classList.contains('visible')) return;
    var focusables = getLightboxFocusableElements();
    if (focusables.length === 0) return;
    var currentIdx = focusables.indexOf(document.activeElement);
    var nextIdx;
    if (e.shiftKey) {
      nextIdx = (currentIdx - 1 + focusables.length) % focusables.length;
    } else {
      nextIdx = (currentIdx + 1) % focusables.length;
    }
    e.preventDefault();
    focusables[nextIdx].focus();
  });

  // Keyboard navigation for lightbox
  document.addEventListener('keydown', function(e) {
    if (!lightbox.classList.contains('visible')) return;
    if (e.key === 'ArrowRight') showNextImage();
    else if (e.key === 'ArrowLeft') showPrevImage();
  });

  function initLightbox() {
    allImages = Array.from(document.querySelectorAll('.chapter-content img'));
    allImages.forEach(function(img) {
      img.style.cursor = 'zoom-in';
      img.addEventListener('click', function() { openLightbox(img.src, img.alt); });
    });
  }

  // ============================================================
  // Full-text Search (⌘K / Ctrl+K to open modal)
  // with platform-specific keyboard shortcut display
  // ============================================================
  var searchOverlay     = document.getElementById('search-overlay');
  var searchInput       = document.getElementById('search-input');
  var searchResultsList = document.getElementById('search-results-list');
  var searchCountLabel  = document.getElementById('search-count-label');
  var searchFocusIdx    = -1;
  var searchPrevFocus   = null;

  // Get all focusable elements within search modal
  function getSearchFocusableElements() {
    var focusableSelector = 'input, button, a, [tabindex]:not([tabindex="-1"])';
    var focusables = searchOverlay.querySelectorAll(focusableSelector);
    return Array.from(focusables).filter(function(el) {
      return el.offsetParent !== null && getComputedStyle(el).visibility !== 'hidden';
    });
  }

  function openSearch() {
    searchOverlay.classList.add('visible');
    searchPrevFocus = document.activeElement;
    searchInput.focus();
    searchInput.select();
  }

  function closeSearch() {
    searchOverlay.classList.remove('visible');
    searchFocusIdx = -1;
    if (searchPrevFocus && typeof searchPrevFocus.focus === 'function') {
      searchPrevFocus.focus();
    }
  }

  var btnSearch = document.getElementById('btn-search');
  btnSearch.addEventListener('click', openSearch);

  // Add keyboard shortcut badge to search button
  var isMac = navigator.platform.includes('Mac');
  var shortcutText = isMac ? '⌘K' : 'Ctrl+K';
  var shortcutBadge = document.createElement('span');
  shortcutBadge.className = 'search-shortcut-badge';
  shortcutBadge.textContent = shortcutText;
  btnSearch.appendChild(shortcutBadge);

  searchOverlay.addEventListener('click', function(e) {
    if (e.target === searchOverlay) closeSearch();
  });

  // Focus trap for search modal: handle Tab key to cycle within modal
  searchOverlay.addEventListener('keydown', function(e) {
    if (e.key !== 'Tab') return;
    var focusables = getSearchFocusableElements();
    if (focusables.length === 0) return;
    var currentIdx = focusables.indexOf(document.activeElement);
    var nextIdx;
    if (e.shiftKey) {
      nextIdx = (currentIdx - 1 + focusables.length) % focusables.length;
    } else {
      nextIdx = (currentIdx + 1) % focusables.length;
    }
    e.preventDefault();
    focusables[nextIdx].focus();
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

    searchCountLabel.textContent = results.length + ' results' + (results.length >= 50 ? ' (showing first 50)' : '');

    if (!results.length) {
      var q = query.replace(/</g, '&lt;');
      searchResultsList.innerHTML = '<div class="search-no-results"><div class="search-no-results-title">No results found</div><div class="search-no-results-text">Try different keywords or check the spelling</div></div>';
      return;
    }

    var re2 = new RegExp('(' + escapeRe(query) + ')', 'gi');
    results.forEach(function(r, i) {
      var div = document.createElement('div');
      div.className = 'search-result';
      div.setAttribute('data-target', r.targetId);

      var titleContainer = document.createElement('div');
      titleContainer.className = 'search-result-header';

      var title = document.createElement('div');
      title.className = 'search-result-title';
      title.textContent = r.title;

      var context = document.createElement('div');
      context.className = 'search-result-context';
      context.textContent = '→ ' + r.targetId;

      titleContainer.appendChild(title);
      titleContainer.appendChild(context);

      var excerpt = document.createElement('div');
      excerpt.className = 'search-result-excerpt';
      excerpt.innerHTML = r.excerpt.replace(/</g, '&lt;').replace(re2, '<mark>$1</mark>');

      div.appendChild(titleContainer);
      div.appendChild(excerpt);
      div.addEventListener('click', function() {
        scrollToId(r.targetId);
        closeSearch();
      });
      searchResultsList.appendChild(div);
    });
  }

  // ============================================================
  // Back-to-top button with smooth fade and scroll animation
  // ============================================================
  document.getElementById('back-to-top').addEventListener('click', function() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });

  // ============================================================
  // Initialization: run all enhancements when DOM is ready
  // ============================================================
  function init() {
    initScrollSpy();
    initSmoothNav();
    enhanceCodeBlocks();
    transformCallouts();
    styleExpandableSections();
    wrapTables();
    initLightbox();
    onScroll(); // Initialize progress bar and TOC highlight
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  `
