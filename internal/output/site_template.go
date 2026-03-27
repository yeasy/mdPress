package output

import "github.com/yeasy/mdpress/pkg/utils"

var sitePageTemplate = `<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="color-scheme" content="light dark">
<meta name="description" content="{{.Description}}">
<meta name="generator" content="mdPress">
<meta property="og:title" content="{{.PageTitle}} - {{.SiteTitle}}">
<meta property="og:description" content="{{.Description}}">
<meta property="og:type" content="article">
{{if .Author}}<meta name="author" content="{{.Author}}">{{end}}
<title>{{.PageTitle}} - {{.SiteTitle}}</title>
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='75' font-size='75' font-weight='bold' fill='%234285f4'>📚</text></svg>">
<link rel="sitemap" type="application/xml" href="{{.SitemapLink}}">
<style>
/* ===== Reset & Base ===== */
* { box-sizing: border-box; margin: 0; padding: 0; }
html { font-size: 16px; scroll-behavior: smooth; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans SC", "Noto Sans CJK SC", "Source Han Sans SC", "WenQuanYi Micro Hei", "Helvetica Neue", Arial, sans-serif;
  line-height: 1.7; color: #333; background: #fff;
  display: flex; min-height: 100vh;
}

/* ===== Sidebar ===== */
.sidebar {
  width: 280px; min-width: 200px; max-width: 50vw;
  background: #fafafa; border-right: 1px solid #e8e8e8;
  padding: 20px 0; overflow-y: auto;
  position: fixed; top: 0; bottom: 0; left: 0; z-index: 100;
  transition: transform 0.3s;
}
.sidebar-resize-handle {
  position: absolute; top: 0; right: -3px; width: 6px; height: 100%;
  cursor: col-resize; z-index: 101; background: transparent;
  transition: background 0.15s;
}
.sidebar-resize-handle:hover,
.sidebar-resize-handle.active {
  background: rgba(66, 133, 244, 0.3);
}
body.sidebar-resizing { cursor: col-resize; user-select: none; }
body.sidebar-resizing .sidebar { transition: none; }
body.sidebar-resizing .main { transition: none; }
.sidebar-header {
  padding: 18px 20px 16px; border-bottom: 1px solid #e8e8e8;
  margin-bottom: 8px;
}
.sidebar-title-row {
  display: flex; align-items: flex-start; justify-content: space-between; gap: 8px;
}
.sidebar-header h1 {
  font-size: 1.04rem; color: #333; font-weight: 650; line-height: 1.25;
  margin: 0; flex: 1;
}
.sidebar-home-link {
  color: inherit;
  text-decoration: none;
}
.sidebar-home-link:hover {
  text-decoration: underline;
}
.sidebar-close {
  background: #f5f5f5; border: 1px solid #e1e1e1; color: #8a8a8a; cursor: pointer;
  font-size: 0.95rem; width: 30px; height: 30px; border-radius: 999px; line-height: 1;
  flex-shrink: 0;
}
.sidebar-close:hover { background: #eee; color: #333; border-color: #d0d0d0; }
.sidebar-subtitle {
  font-size: 0.72rem; color: #7a7a7a; margin-bottom: 8px; line-height: 1.3;
  text-transform: uppercase; letter-spacing: 0.06em;
}
.sidebar-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 10px;
}
.sidebar-author {
  font-size: 0.78rem;
  color: #8d8d8d;
  font-weight: 600;
}
.sidebar-description {
  font-size: 0.84rem;
  color: #6f6f6f;
  margin-top: 10px;
  line-height: 1.55;
}
.theme-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  border-radius: 999px;
  border: 1px solid rgba(0,0,0,0.08);
  color: var(--color-link, #4285f4);
  background: rgba(255,255,255,0.9);
  font-size: 0.71rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.sidebar-nav { padding: 0 10px; }

.nav-group { margin: 2px 0; }
.nav-row {
  display: flex; align-items: center; gap: 4px;
  padding-right: 8px;
}
.nav-toggle {
  width: 24px; height: 24px; border: none; background: transparent;
  color: #666; cursor: pointer; border-radius: 4px; flex: 0 0 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}
.nav-toggle:hover {
  background: #e0e0e0;
}
.nav-toggle::before {
  content: ""; display: block; width: 0; height: 0;
  border-left: 4px solid transparent;
  border-right: 4px solid transparent;
  border-top: 5px solid currentColor;
  transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}
.nav-group.collapsed .nav-toggle::before { transform: rotate(-90deg); }
.nav-item {
  display: block; color: #555; text-decoration: none;
  font-size: 0.9rem; border-radius: 4px; margin: 1px 0; transition: all 0.15s;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.nav-item:hover { background: #e8e8e8; color: #111; }
.nav-item.active { background: var(--color-accent, #4285f4); color: #fff; font-weight: 500; }
.nav-chapter { flex: 1; padding: 6px 10px 6px 8px; font-weight: 600; }
.nav-row:not(:has(.nav-toggle)) .nav-chapter { padding-left: 14px; font-weight: 400; }
.nav-heading { padding: 5px 12px; font-size: 0.84rem; margin-left: 26px; }
.nav-depth-1 { padding-left: 8px; }
.nav-depth-2 { padding-left: 22px; }
.nav-depth-3 { padding-left: 36px; }
.nav-depth-4 { padding-left: 50px; }
.nav-heading-depth-1 { padding-left: 12px; }
.nav-heading-depth-2 { padding-left: 26px; }
.nav-heading-depth-3 { padding-left: 40px; font-size: 0.8rem; }
.nav-heading-depth-4 { padding-left: 54px; font-size: 0.78rem; }
.nav-children {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transition: grid-template-rows 0.28s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.nav-children-inner {
  min-height: 0;
  overflow: hidden;
  padding-bottom: 2px;
  padding-left: 26px;
}
.nav-group.expanded > .nav-children {
  grid-template-rows: 1fr;
  opacity: 1;
}

/* ===== Page Header ===== */
.page-header {
  padding: 12px 50px;
  border-bottom: 1px solid #e8e8e8;
  background: rgba(250,250,250,0.95);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 16px;
  position: sticky;
  top: 0;
  z-index: 50;
}
.page-breadcrumb {
  font-size: 0.85rem;
  color: #666;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.page-breadcrumb a {
  color: var(--color-link, #4285f4);
  text-decoration: none;
}
.page-breadcrumb a:hover {
  text-decoration: underline;
}
.bc-sep {
  color: #999;
  font-size: 0.8em;
}
.chapter-title {
  font-size: 1.8rem;
  font-weight: 700;
  margin: 0 0 1.2rem 0;
  padding-bottom: 0.4rem;
  border-bottom: 1px solid #e8e8e8;
  line-height: 1.3;
}

/* ===== Main Content ===== */
.main {
  margin-left: var(--sidebar-width, 280px); flex: 1; min-width: 0;
  transition: margin-left 0.3s;
}
.main-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 220px;
}
.content {
  max-width: min(860px, 100%); padding: 40px 50px 80px; overflow-wrap: anywhere;
  min-width: 0;
}
@media (min-width: 1400px) {
  .content { max-width: 960px; }
}
@media (min-width: 1600px) {
  .content { max-width: 1080px; }
}
@media (min-width: 1900px) {
  .content { max-width: 1200px; }
}

/* ===== Right-Side Page TOC ===== */
.page-toc {
  position: sticky; top: 64px; align-self: start;
  max-height: calc(100vh - 80px); overflow-y: auto;
  padding: 16px 16px 16px 0; font-size: 0.82rem; line-height: 1.5;
  border-left: 1px solid #e8e8e8;
}
.page-toc-header {
  font-weight: 600; color: #555; margin-bottom: 10px; padding-left: 16px;
  font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.05em;
}
.page-toc-nav a {
  display: block; padding: 3px 12px 3px 16px; color: #666;
  text-decoration: none; border-left: 2px solid transparent;
  margin-left: -1px; transition: color 0.15s, border-color 0.15s;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.page-toc-nav a:hover { color: #333; }
.page-toc-nav a.toc-active { color: var(--color-link, #4285f4); border-left-color: var(--color-accent, #4285f4); font-weight: 500; }
.page-toc-nav a.toc-depth-2 { padding-left: 28px; font-size: 0.78rem; }
.page-toc-nav a.toc-depth-3 { padding-left: 40px; font-size: 0.76rem; }
.page-toc:empty, .page-toc.toc-hidden { display: none; }
.content.is-navigating { pointer-events: none; }
.route-progress {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 3px;
  z-index: 320;
  pointer-events: none;
}
.route-progress-bar {
  width: 0;
  height: 100%;
  opacity: 0;
  background: linear-gradient(90deg, var(--color-accent, #4285f4) 0%, var(--color-link, #74a7ff) 100%);
  box-shadow: 0 0 14px rgba(66, 133, 244, 0.35);
  transition: width 0.22s ease, opacity 0.18s ease;
}
.route-progress.is-active .route-progress-bar {
  width: 68%;
  opacity: 1;
}
.route-progress.is-finishing .route-progress-bar {
  width: 100%;
  opacity: 1;
}
.content h1[id], .content h2[id], .content h3[id], .content h4[id], .content h5[id], .content h6[id] {
  scroll-margin-top: 64px;
}
.content h1 { font-size: 2em; margin: 0 0 0.8em; color: var(--color-heading, #1a1a2e); border-bottom: 2px solid var(--color-accent, #4285f4); padding-bottom: 0.3em; }
.content h2 { font-size: 1.5em; margin: 1.5em 0 0.6em; color: #333; padding-bottom: 0.3em; border-bottom: 1px solid #eee; }
.content h3 { font-size: 1.2em; margin: 1.3em 0 0.5em; color: #444; }
.content h4 { font-size: 1.05em; margin: 1em 0 0.4em; color: #555; }
.content h1[id] a.header-anchor,
.content h2[id] a.header-anchor,
.content h3[id] a.header-anchor,
.content h4[id] a.header-anchor {
  float: left;
  margin-left: -0.87em;
  padding-right: 0.23em;
  font-weight: 500;
  opacity: 0;
  color: var(--color-link, #4285f4);
  text-decoration: none;
  transition: opacity 0.15s;
}
.content h1:hover a.header-anchor,
.content h2:hover a.header-anchor,
.content h3:hover a.header-anchor,
.content h4:hover a.header-anchor { opacity: 1; }
html.dark .content h1[id] a.header-anchor,
html.dark .content h2[id] a.header-anchor,
html.dark .content h3[id] a.header-anchor,
html.dark .content h4[id] a.header-anchor { color: #89b4fa; }
.content p { margin: 0.6em 0; text-align: left; }
.content img { max-width: 100%; height: auto; border-radius: 4px; vertical-align: middle; }
.content p:has(> img), .content p:has(> a > img) { text-align: center; }
.content p > a:not(:only-child) > img,
.content p > a:not(:only-child) > svg { max-height: 20px; width: auto; vertical-align: middle; display: inline; }
.content p > img:only-child,
.content p > a:only-child > img:only-child {
  display: block; margin: 1em auto;
}
.content figure {
  margin: 1.25em auto;
  text-align: center;
}
.content figure img {
  margin: 0 auto;
}
.content figcaption {
  margin-top: 0.5em;
  color: #666;
  font-size: 0.84rem;
  text-align: center;
}
.content .mermaid {
  display: flex;
  justify-content: center;
  margin: 1.25em auto;
  text-align: center;
}
.content .mermaid svg,
.content .mermaid > * {
  margin: 0 auto;
}
.content blockquote {
  border-left: 4px solid var(--color-accent, #4285f4); background: #f4f7ff; margin: 1em 0;
  padding: 12px 16px; color: #555; border-radius: 0 4px 4px 0;
}
.content blockquote p { margin: 0.3em 0; }
.content code {
  background: #f0f0f0; padding: 2px 6px; border-radius: 3px;
  font-family: "Fira Code", "Consolas", "Monaco", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Noto Sans Mono CJK SC", monospace; font-size: 0.9em;
  overflow-wrap: anywhere; word-break: break-word;
}
.content pre {
  background: #f6f8fa; color: #24292e; padding: 16px 20px;
  border: 1px solid #e1e4e8; border-radius: 6px; overflow-x: auto; margin: 1em 0; line-height: 1.5;
  font-size: 0.88em; white-space: pre; word-break: normal;
}
.content pre code { background: transparent; color: inherit; padding: 0; font-size: inherit; display: block; }

/* ===== Code Block Copy Button ===== */
.code-wrapper { position: relative; }
.code-wrapper .copy-btn {
  position: absolute; top: 8px; right: 8px;
  background: rgba(255,255,255,.8); border: 1px solid #d0d7de; border-radius: 4px;
  padding: 4px 8px; font-size: 12px; cursor: pointer; color: #555;
  opacity: 0; transition: opacity .15s;
  line-height: 1; z-index: 1;
}
.code-wrapper:hover .copy-btn { opacity: 1; }
.copy-btn.copied { background: #2ea44f; color: #fff; border-color: #2ea44f; }
html.dark .code-wrapper .copy-btn { background: rgba(30,30,46,.8); border-color: #45475a; color: #a6adc8; }
html.dark .copy-btn.copied { background: #a6e3a1; color: #1e1e2e; border-color: #a6e3a1; }

.content table { border-collapse: collapse; width: 100%; margin: 1em 0; table-layout: auto; border-radius: 6px; overflow: hidden; border: 1px solid #e8e8e8; }
.content th, .content td { border: 1px solid #e8e8e8; padding: 10px 16px; text-align: left; overflow-wrap: anywhere; word-break: break-word; }
.content th { background: #f8f9fa; font-weight: 600; font-size: 0.88em; text-transform: none; letter-spacing: 0; color: #555; }
.content tr:nth-child(even) { background: #fcfcfc; }
.content th:first-child, .content td:first-child { border-left: none; }
.content th:last-child, .content td:last-child { border-right: none; }
.content tr:first-child th { border-top: none; }
.content tr:last-child td { border-bottom: none; }
.content a { color: var(--color-link, #4285f4); text-decoration: none; }
.content a:hover { text-decoration: underline; }
.content ul, .content ol { padding-left: 1.8em; margin: 0.5em 0; }
.content li { margin: 0.3em 0; }
.content hr { border: none; height: 1px; background: #e0e0e0; margin: 2em 0; }

/* ===== Glossary terms ===== */
.glossary-term {
  border-bottom: 1px dashed #4285f4; cursor: help;
}

/* ===== Page Navigation ===== */
.page-nav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-top: 3em;
  padding-top: 2em;
  border-top: 1px solid #e8e8e8;
}
.page-nav > span {
  /* Placeholder for empty nav slots */
}
.page-nav a {
  color: inherit; text-decoration: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 16px;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  transition: all 0.2s ease;
}
.page-nav a:hover {
  border-color: var(--color-accent, #4285f4);
  background: #f8fbff;
  box-shadow: 0 2px 8px rgba(66, 133, 244, 0.1);
}
.page-nav .nav-label {
  font-size: 0.75rem;
  color: #999;
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.page-nav .nav-title {
  color: var(--color-link, #4285f4);
  font-weight: 500;
  font-size: 0.95rem;
}
.page-nav .prev {
  grid-column: 1;
  justify-self: start;
}
.page-nav .prev .nav-label::before { content: "← "; }
.page-nav .next {
  grid-column: 2;
  justify-self: end;
  text-align: right;
}
.page-nav .next .nav-label::after { content: " →"; }

/* ===== Build Meta ===== */
.build-meta {
  margin-top: 3rem;
  padding-top: 2rem;
  border-top: 1px solid #e8e8e8;
  color: #999;
  font-size: 0.82rem;
  text-align: center;
}
.build-meta a {
  color: var(--color-link, #4285f4);
  text-decoration: none;
  font-weight: 500;
}
.build-meta a:hover {
  text-decoration: underline;
}
.page-meta {
  margin-top: 1.5rem;
  padding-top: 1rem;
  color: #999;
  font-size: 0.78rem;
  text-align: center;
  border-top: 1px solid #e8e8e8;
}

/* ===== Page Transition ===== */
@keyframes mdpress-page-out {
  from {
    opacity: 1;
  }
  to {
    opacity: 0;
  }
}

@keyframes mdpress-page-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes mdpress-sidebar-in {
  from {
    opacity: 0.84;
    transform: translate3d(-8px, 0, 0);
  }
  to {
    opacity: 1;
    transform: translate3d(0, 0, 0);
  }
}

/* ===== Sidebar Toggle ===== */
.sidebar-toggle {
  display: flex; position: fixed; top: 12px; left: 12px; z-index: 200;
  background: #4285f4; color: #fff; border: none; border-radius: 4px;
  width: 36px; height: 36px; font-size: 1.2rem; cursor: pointer;
  align-items: center; justify-content: center;
  opacity: 0; pointer-events: none; transition: opacity 0.2s;
}
.sidebar-toggle:hover,
.sidebar-toggle:focus-visible,
body.sidebar-collapsed .sidebar-toggle {
  opacity: 1; pointer-events: auto;
}
body.sidebar-collapsed .sidebar {
  transform: translateX(-100%);
}
body.sidebar-collapsed .main {
  margin-left: 0 !important;
}

/* ===== Mobile Overlay ===== */
body.sidebar-open::before {
  content: "";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 99;
  transition: opacity 0.3s ease;
}

/* ===== Responsive ===== */
@media (max-width: 900px) {
  .page-header {
    padding: 16px 24px;
  }
}

@media (max-width: 960px) {
  .page-toc { display: none; }
  .main-body { grid-template-columns: 1fr; }
}

@media (max-width: 768px) {
  .sidebar { transform: translateX(-100%); }
  .sidebar.open { transform: translateX(0); box-shadow: 2px 0 12px rgba(0,0,0,.2); }
  .sidebar-toggle { opacity: 1; pointer-events: auto; }
  .sidebar-resize-handle { display: none; }
  .main { margin-left: 0 !important; }
  .main-body { grid-template-columns: 1fr; }
  .content { padding: 24px 20px 80px; }
  .page-header { padding: 12px 16px 12px 56px; }
  .header-search-btn span { display: none; }
  .header-search-btn kbd { display: none; }
  .page-nav {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  .page-nav .prev,
  .page-nav .next {
    grid-column: auto;
    justify-self: stretch;
    text-align: left;
  }
  .page-nav a {
    padding: 16px 12px;
  }
  .sidebar-header {
    padding: 12px 16px;
  }
  .nav-chapter {
    font-size: 0.88rem;
  }
  .search-inline { width: 100%; }
  .search-backdrop.open { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  html { scroll-behavior: auto; }
  .sidebar, .nav-toggle::before, .nav-children, .nav-item { transition: none; }
}

/* ===== Focus Visible for Keyboard Navigation ===== */
a:focus-visible {
  outline: 2px solid #4285f4;
  outline-offset: 2px;
  border-radius: 2px;
}
button:focus-visible,
input:focus-visible,
textarea:focus-visible,
select:focus-visible {
  outline: 2px solid #4285f4;
  outline-offset: 2px;
}

/* ===== Skip to Content ===== */
.skip-link {
  position: absolute; top: -100%; left: 16px; z-index: 9999;
  background: #4285f4; color: #fff; padding: 8px 16px; border-radius: 4px;
  font-size: 0.9rem; text-decoration: none;
}
.skip-link:focus { top: 12px; }

/* ===== Search Panel (right-aligned dropdown) ===== */
.search-inline {
  display: none;
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 460px;
  max-width: 100vw;
  z-index: 200;
  flex-direction: column;
  background: #fff;
  border-left: 1px solid #e5e7eb;
  box-shadow: -4px 0 24px rgba(15, 23, 42, 0.10);
}
.search-inline.open { display: flex; }
.search-backdrop {
  display: none;
  position: fixed;
  top: 0; left: 0; right: 460px; bottom: 0;
  z-index: 199;
}
.search-backdrop.open { display: block; }
.search-header {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
  gap: 10px;
}
.search-header .search-icon { color: #999; font-size: 18px; flex-shrink: 0; }
.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  background: transparent;
  color: #333;
}
.search-input::placeholder { color: #aaa; }
.search-esc {
  font-size: 11px;
  color: #999;
  background: #f0f0f0;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 2px 6px;
  flex-shrink: 0;
}
.search-results {
  overflow-y: auto;
  padding: 8px;
  flex: 1;
  min-height: 0;
}
.search-status {
  padding: 6px 16px 0;
  font-size: 0.74rem;
  color: #8a8a8a;
  min-height: 1.2em;
}
.search-result {
  display: block;
  text-decoration: none;
  padding: 10px 12px;
  border-radius: 8px;
  color: inherit;
  cursor: pointer;
}
.search-result:hover, .search-result.search-active { background: #f0f4ff; }
.search-result-title {
  font-weight: 600;
  font-size: 0.9rem;
  color: #333;
  margin-bottom: 2px;
}
.search-result-meta {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 4px;
}
.search-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: 999px;
  background: #eef4ff;
  color: #3563b8;
  font-size: 0.67rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.search-result-snippet {
  font-size: 0.8rem;
  color: #666;
  line-height: 1.4;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.search-result-snippet mark {
  background: #fff3b0;
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
}
.search-result-path {
  font-size: 0.7rem;
  color: #999;
  margin-bottom: 2px;
}
.search-empty {
  text-align: center;
  padding: 24px 16px;
  color: #999;
  font-size: 0.9rem;
}
.search-jump-notice {
  margin: 0 0 14px;
  padding: 10px 12px;
  border-left: 3px solid var(--color-accent, #4285f4);
  border-radius: 10px;
  background: #f8fbff;
  color: #4b5563;
  font-size: 0.82rem;
}
.search-footer {
  padding: 8px 16px;
  border-top: 1px solid #e8e8e8;
  font-size: 0.75rem;
  color: #999;
  display: flex;
  gap: 16px;
}
.search-footer kbd {
  background: #f0f0f0;
  border: 1px solid #ddd;
  border-radius: 3px;
  padding: 1px 4px;
  font-size: 0.7rem;
  font-family: inherit;
}

/* ===== Dark Mode ===== */
/* JS adds .dark to <html> based on user preference or system default */
html.dark body { background: #1e1e2e; color: #cdd6f4; }
html.dark .sidebar { background: #181825; border-right-color: #313244; }
html.dark .sidebar-header { border-bottom-color: #313244; }
html.dark .sidebar-header h1 { color: #cdd6f4; }
html.dark .sidebar-home-link { color: inherit; }
html.dark .sidebar-subtitle { color: #8b91ab; }
html.dark .sidebar-author { color: #a6adc8; }
html.dark .sidebar-description { color: #8b91ab; }
html.dark .theme-badge { background: rgba(38,38,55,0.92); border-color: #363849; color: #89b4fa; }
html.dark .sidebar-close { background: #262637; border-color: #363849; color: #8b91ab; }
html.dark .sidebar-close:hover { background: #313244; color: #cdd6f4; border-color: #45475a; }
html.dark .nav-chapter { color: #bac2de; }
html.dark .nav-heading { color: #a6adc8; }
html.dark .nav-item:hover { background: #313244; color: #cdd6f4; }
html.dark .nav-item.active { background: rgba(137,180,250,.15); color: #89b4fa; }
html.dark .nav-toggle::before { border-color: #6c7086; }
html.dark .main { background: #1e1e2e; }
html.dark .page-header { border-bottom-color: #313244; background: rgba(24,24,37,0.95); }
html.dark .page-breadcrumb a { color: #89b4fa; }
html.dark .bc-sep { color: #6c7086; }
html.dark .chapter-title { color: #cdd6f4; border-bottom-color: #313244; }
html.dark .content { color: #cdd6f4; }
html.dark .content h1 { color: #cdd6f4; border-bottom-color: #89b4fa; }
html.dark .content h2, html.dark .content h3, html.dark .content h4 { color: #bac2de; }
html.dark .content h2 { border-bottom-color: #313244; }
html.dark .content a { color: #89b4fa; }
html.dark .content a:hover { color: #b4d0fb; }
html.dark .content pre { background: #262637; color: #cdd6f4; border-color: #363849; }
html.dark .content code { background: #363849; color: #cdd6f4; }
html.dark .content pre code { background: transparent; color: inherit; }
html.dark .content blockquote { border-left-color: #89b4fa; color: #bac2de; background: #262637; }
html.dark .content table th { background: #262637; color: #cdd6f4; border-color: #363849; }
html.dark .content table td { border-color: #363849; }
html.dark .content table tr:nth-child(even) { background: #22223a; }
html.dark .content img { border-color: #363849; }
html.dark .content figcaption { color: #a6adc8; }
html.dark .content hr { background: #363849; }
html.dark .page-nav a { background: #262637; border-color: #363849; color: #cdd6f4; }
html.dark .page-nav a:hover { border-color: #89b4fa; background: #2a2a3e; }
html.dark .page-meta, html.dark .build-meta { color: #6c7086; }
html.dark .build-meta a { color: #89b4fa; }
html.dark .sidebar-toggle { background: #89b4fa; color: #1e1e2e; }
html.dark .sidebar-resize-handle:hover, html.dark .sidebar-resize-handle.active { background: rgba(137,180,250,.3); }
html.dark body.sidebar-open::before { background: rgba(0,0,0,.5); }
html.dark .route-progress-bar { background: #89b4fa; }
html.dark .page-toc { border-left-color: #313244; }
html.dark .page-toc-header { color: #a6adc8; }
html.dark .page-toc-nav a { color: #6c7086; }
html.dark .page-toc-nav a:hover { color: #cdd6f4; }
html.dark .page-toc-nav a.toc-active { color: #89b4fa; border-left-color: #89b4fa; }
html.dark .search-inline { background: #1e1e2e; border-left-color: #313244; box-shadow: -4px 0 24px rgba(0,0,0,.3); }
html.dark .search-header { border-bottom-color: #313244; }
html.dark .search-input { color: #cdd6f4; }
html.dark .search-input::placeholder { color: #6c7086; }
html.dark .search-esc { background: #313244; border-color: #45475a; color: #a6adc8; }
html.dark .search-result:hover, html.dark .search-result.search-active { background: #313244; }
html.dark .search-result-title { color: #cdd6f4; }
html.dark .search-badge { background: rgba(137,180,250,.18); color: #b9d8ff; }
html.dark .search-result-snippet { color: #a6adc8; }
html.dark .search-result-path { color: #6c7086; }
html.dark .search-result-snippet mark { background: #45475a; color: #f9e2af; }
html.dark .search-empty { color: #6c7086; }
html.dark .search-jump-notice { background: #262637; color: #bac2de; }
html.dark .search-status { color: #6c7086; }
html.dark .search-footer { border-top-color: #313244; color: #6c7086; }
html.dark .search-footer kbd { background: #313244; border-color: #45475a; }
html.dark .header-search-btn { background: #313244; border-color: #45475a; color: #6c7086; }
html.dark .header-search-btn:hover { border-color: #585b70; background: #3b3d52; color: #a6adc8; }
html.dark .header-search-btn kbd { background: #45475a; border-color: #585b70; color: #6c7086; }
html.dark .theme-toggle { background: #313244; border-color: #45475a; }
html.dark .theme-toggle button { color: #a6adc8; }
html.dark .theme-toggle button:hover { background: #45475a; color: #cdd6f4; }
html.dark .theme-toggle button.active { background: #89b4fa; color: #1e1e2e; }
/* Theme toggle button */
.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
  flex-shrink: 0;
}
.header-search-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  border: 1px solid #d8d8d8;
  border-radius: 6px;
  background: #f5f5f5;
  color: #888;
  font-size: 0.82rem;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
  white-space: nowrap;
  line-height: 1.4;
}
.header-search-btn:hover {
  border-color: #bbb;
  background: #eee;
  color: #555;
}
.header-search-btn svg {
  width: 14px; height: 14px; flex-shrink: 0;
  stroke: currentColor; fill: none;
  stroke-width: 2; stroke-linecap: round; stroke-linejoin: round;
}
.header-search-btn kbd {
  font-size: 0.65rem;
  background: #e8e8e8;
  border: 1px solid #d0d0d0;
  border-radius: 3px;
  padding: 1px 5px;
  color: #999;
  font-family: inherit;
  line-height: 1.4;
}
.theme-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0;
  background: #f0f0f0;
  border: 1px solid #ddd;
  border-radius: 6px;
  padding: 2px;
  flex-shrink: 0;
}
.theme-toggle button {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 14px;
  color: #666;
  line-height: 1;
  transition: background 0.15s, color 0.15s;
}
.theme-toggle button:hover { background: #e0e0e0; color: #333; }
.theme-toggle button.active { background: #4285f4; color: #fff; box-shadow: 0 1px 3px rgba(66,133,244,0.3); }

/* ===== Print Styles ===== */
@media print {
  .sidebar, .sidebar-toggle, .sidebar-overlay, .route-progress, .page-toc, .search-inline { display: none !important; }
  .main { margin-left: 0; }
  .main-body { grid-template-columns: 1fr; }
  .content { padding: 0; margin: 0; max-width: 100%; }
  .page-header, .page-nav, .build-meta { display: none !important; }
  body { background: white; color: black; }
  * { background: transparent !important; box-shadow: none !important; color: black !important; }
  a { color: #0969da; text-decoration: underline; }
  a::after { content: " (" attr(href) ")"; font-size: 0.8em; color: #999; }
  h1, h2, h3, h4, h5, h6 { page-break-after: avoid; }
  p, pre, blockquote { page-break-inside: avoid; }
  img { max-width: 100%; page-break-inside: avoid; }
  table { page-break-inside: avoid; }
}

/* ===== Custom Theme CSS ===== */
{{safeCSS .CSS}}

/* ===== Site Layout Overrides ===== */
body {
  margin: 0 !important;
  padding: 0 !important;
}
</style>
<script>
/* Prevent flash of wrong theme */
(function(){var t=localStorage.getItem('mdpress-theme');if(t==='dark'||(t!=='light'&&window.matchMedia('(prefers-color-scheme:dark)').matches)){document.documentElement.classList.add('dark')}})();
</script>
</head>
<body>
  <a href="#main-content" class="skip-link">Skip to content</a>

  <div class="search-backdrop" id="search-backdrop"></div>
  <div class="search-inline" id="search-overlay" role="search" aria-label="{{.UIsearchButton}}">
    <div class="search-header">
      <span class="search-icon">&#128269;</span>
      <input type="text" class="search-input" id="search-input" placeholder="{{.UIsearchPlaceholder}}" autocomplete="off" spellcheck="false">
      <span class="search-esc">ESC</span>
    </div>
    <div class="search-status" id="search-status" aria-live="polite"></div>
    <div class="search-results" id="search-results">
      <div class="search-empty">{{.UIsearchPlaceholder}}</div>
    </div>
    <div class="search-footer">
      <span><kbd>↑</kbd><kbd>↓</kbd> {{.UIsearchNavigate}}</span>
      <span><kbd>↵</kbd> {{.UIsearchOpen}}</span>
      <span><kbd>esc</kbd> {{.UIsearchClose}}</span>
    </div>
  </div>

  <div class="route-progress" id="route-progress" aria-hidden="true">
    <div class="route-progress-bar"></div>
  </div>
  <button class="sidebar-toggle" aria-label="Toggle navigation menu" aria-controls="sidebar-nav" aria-expanded="false">
    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <line x1="3" y1="6" x2="21" y2="6"></line>
      <line x1="3" y1="12" x2="21" y2="12"></line>
      <line x1="3" y1="18" x2="21" y2="18"></line>
    </svg>
  </button>

  <nav class="sidebar">
    <div class="sidebar-resize-handle" id="sidebar-resize-handle"></div>
    <div class="sidebar-header">
      <div class="sidebar-title-row">
        <h1><a class="sidebar-home-link" href="{{.HomeLink}}">{{.SiteTitle}}</a></h1>
        <button class="sidebar-close" aria-label="{{.UIhideSidebar}}" title="{{.UIhideSidebar}}">✕</button>
      </div>
      {{if .SiteSubtitle}}<div class="sidebar-subtitle">{{.SiteSubtitle}}</div>{{end}}
      {{if or .Author .ThemeName}}<div class="sidebar-meta">
      {{if .Author}}<span class="sidebar-author">{{.Author}}</span>{{end}}
      {{if .ThemeName}}<div class="theme-badge" title="{{.ThemeDescription}}">{{.ThemeName}}</div>{{end}}
      </div>{{end}}
      {{if .SiteDescription}}<div class="sidebar-description">{{.SiteDescription}}</div>{{end}}
    </div>
    <div class="sidebar-nav">
      {{safeHTML .SidebarHTML}}
    </div>
  </nav>

  <main class="main">
    <header class="page-header">
      <nav class="page-breadcrumb" aria-label="Breadcrumb">
        <a href="{{.HomeLink}}">{{.SiteTitle}}</a>
        {{range .Breadcrumbs}}<span class="bc-sep">›</span><a href="{{.Filename}}">{{.Title}}</a>{{end}}
      </nav>
      <div class="header-right">
        <button class="header-search-btn" id="header-search-btn" type="button" aria-label="{{.UIsearchButton}}">
          <svg viewBox="0 0 24 24"><circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line></svg>
          <span>{{.UIsearchPlaceholder}}</span>
          <kbd>{{.UIsearchKbd}}</kbd>
        </button>
        <div class="theme-toggle" aria-label="Theme switcher">
          <button type="button" data-theme="light" title="{{.UIlightMode}}" aria-label="{{.UIlightMode}}"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg></button>
          <button type="button" data-theme="system" title="{{.UIsystemDefault}}" aria-label="{{.UIsystemDefault}}"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect><line x1="8" y1="21" x2="16" y2="21"></line><line x1="12" y1="17" x2="12" y2="21"></line></svg></button>
          <button type="button" data-theme="dark" title="{{.UIdarkMode}}" aria-label="{{.UIdarkMode}}"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg></button>
        </div>
      </div>
    </header>
    <div class="main-body">
      <div class="content" id="main-content">
        {{if .ShowTitle}}<h1>{{.PageTitle}}</h1>{{end}}
        {{safeHTML .Content}}

        <nav class="page-nav" aria-label="Page navigation">
          {{if .PrevLink}}<a class="prev" href="{{.PrevLink}}"><span class="nav-label">{{.UIprevious}}</span><span class="nav-title">{{.PrevTitle}}</span></a>{{else}}<span></span>{{end}}
          {{if .NextLink}}<a class="next" href="{{.NextLink}}"><span class="nav-label">{{.UInext}}</span><span class="nav-title">{{.NextTitle}}</span></a>{{else}}<span></span>{{end}}
        </nav>

        <div class="page-meta">
          <span>{{printf .UIpageOf .CurrentPage .TotalPages}}</span>
        </div>

        <div class="build-meta">
          <span class="build-meta-text">{{printf .UIbuiltWith "mdPress"}}</span>
        </div>
      </div>
      <aside class="page-toc" id="page-toc" role="navigation" aria-label="{{.UIonThisPage}}">
        <div class="page-toc-header">{{.UIonThisPage}}</div>
        <nav class="page-toc-nav" id="page-toc-nav"></nav>
      </aside>
    </div>
  </main>

  <script>
  /* ===== Localized UI Strings ===== */
  var __ui = {
    searchPlaceholder: "{{.UIsearchPlaceholder}}",
    noResults: "{{.UInoResults}}",
    searchUnavailable: "{{.UIsearchUnavailable}}",
    searchResultsOne: "{{.UIsearchResultsOne}}",
    searchResults: "{{.UIsearchResults}}",
    recentPages: "{{.UIrecentPages}}",
    recentEmpty: "{{.UIrecentEmpty}}",
    searchMatchTitle: "{{.UIsearchMatchTitle}}",
    searchMatchPath: "{{.UIsearchMatchPath}}",
    searchMatchText: "{{.UIsearchMatchText}}",
    searchMatched: "{{.UIsearchMatched}}",
    copy: "{{.UIcopy}}",
    copied: "{{.UIcopied}}"
  };
  /* ===== Theme Management ===== */
  (function() {
    var stored = localStorage.getItem('mdpress-theme');
    var prefersDark = window.matchMedia('(prefers-color-scheme: dark)');

    function applyTheme(mode) {
      if (mode === 'dark' || (mode !== 'light' && prefersDark.matches)) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    }

    function setTheme(mode) {
      if (mode === 'system') {
        localStorage.removeItem('mdpress-theme');
      } else {
        localStorage.setItem('mdpress-theme', mode);
      }
      applyTheme(mode);
      updateToggleButtons(mode);
    }

    function updateToggleButtons(mode) {
      var buttons = document.querySelectorAll('.theme-toggle button');
      for (var i = 0; i < buttons.length; i++) {
        var btn = buttons[i];
        if (btn.getAttribute('data-theme') === mode) {
          btn.classList.add('active');
          btn.setAttribute('aria-pressed', 'true');
        } else {
          btn.classList.remove('active');
          btn.setAttribute('aria-pressed', 'false');
        }
      }
    }

    // Listen for system preference changes when in system mode
    prefersDark.addEventListener('change', function() {
      var current = localStorage.getItem('mdpress-theme');
      if (!current) applyTheme('system');
    });

    // Set up toggle buttons after DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
      updateToggleButtons(stored || 'system');
      document.addEventListener('click', function(e) {
        var btn = e.target.closest('.theme-toggle button');
        if (btn) setTheme(btn.getAttribute('data-theme'));
      });
    });

    // Add permalink anchors to headings
    function addHeaderAnchors(root) {
      (root || document).querySelectorAll('.content h1[id], .content h2[id], .content h3[id], .content h4[id]').forEach(function(h) {
        if (h.querySelector('.header-anchor')) return;
        var a = document.createElement('a');
        a.className = 'header-anchor';
        a.href = '#' + h.id;
        a.textContent = '#';
        a.setAttribute('aria-hidden', 'true');
        h.prepend(a);
      });
    }
    addHeaderAnchors();

    window.__addHeaderAnchors = addHeaderAnchors;
    window.__setTheme = setTheme;
  })();

  var sidebar = document.querySelector('.sidebar');
  var body = document.body;
  var mainContent = document.querySelector('.content');
  var routeProgress = document.getElementById('route-progress');
  var sidebarToggle = document.querySelector('.sidebar-toggle');
  var prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  var navUpdateFrame = null;
  var scrollSaveFrame = null;
  var lastActiveLink = null;
  var prefetchedPages = Object.create(null);
  var pageCache = Object.create(null);
  var pendingNavigation = null;
  var internalNavStateKey = 'mdpress-site-nav';
  var scrollStoreKey = 'mdpress-site-scroll';
  var currentFile = '{{.ActiveFile}}';
  var navLinksByCurrentFile = [];
  var navChapterLinks = [];
  var navHeadingLinks = [];
  var headings = [];

  try {
    window.sessionStorage.removeItem(internalNavStateKey);
  } catch (e) {}

  function getInternalPageURL(href) {
    if (!href) return null;
    try {
      var url = new URL(href, window.location.href);
      if (url.origin !== window.location.origin) return null;
      if (url.pathname === window.location.pathname) return null;
      if (!/\.html$/i.test(url.pathname)) return null;
      url.hash = '';
      return url.toString();
    } catch (e) {
      return null;
    }
  }

  function prefetchPage(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL || prefetchedPages[pageURL]) return;
    prefetchedPages[pageURL] = true;

    var link = document.createElement('link');
    link.rel = 'prefetch';
    link.href = pageURL;
    link.as = 'document';
    document.head.appendChild(link);
  }

  function warmPageCache(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL) return;
    var targetURL = new URL(pageURL, window.location.href);
    fetchPagePayload(targetURL).catch(function() {});
  }

  function rememberInternalNavigation(href) {
    var pageURL = getInternalPageURL(href);
    if (!pageURL) return;
    try {
      window.sessionStorage.setItem(internalNavStateKey, JSON.stringify({
        ts: Date.now(),
        href: pageURL
      }));
    } catch (e) {}
  }

  function getFileFromPathname(pathname) {
    if (!pathname) return currentFile || 'index.html';
    var clean = pathname.replace(/\/+$/, '').replace(/^\/+/, '');
    return clean || 'index.html';
  }

  function refreshPageContext() {
    navLinksByCurrentFile = Array.from(document.querySelectorAll('.nav-item[data-file="' + currentFile + '"]'));
    navChapterLinks = Array.from(document.querySelectorAll('.nav-chapter[data-file="' + currentFile + '"]'));
    navHeadingLinks = Array.from(document.querySelectorAll('.nav-heading[data-file="' + currentFile + '"]'));
    headings = Array.from(document.querySelectorAll('.content h1[id], .content h2[id], .content h3[id], .content h4[id], .content h5[id], .content h6[id]'));
  }

  function setNavigating(isNavigating) {
    if (!mainContent) return;
    mainContent.classList.toggle('is-navigating', isNavigating);
  }

  function beginRouteProgress() {
    if (!routeProgress || prefersReducedMotion) return;
    routeProgress.classList.remove('is-finishing');
    routeProgress.classList.add('is-active');
  }

  function endRouteProgress() {
    if (!routeProgress || prefersReducedMotion) return;
    routeProgress.classList.remove('is-active');
    routeProgress.classList.add('is-finishing');
    window.setTimeout(function() {
      routeProgress.classList.remove('is-finishing');
    }, 220);
  }

  function readScrollStore() {
    try {
      return JSON.parse(window.sessionStorage.getItem(scrollStoreKey) || '{}');
    } catch (e) {
      return {};
    }
  }

  function writeScrollStore(store) {
    try {
      window.sessionStorage.setItem(scrollStoreKey, JSON.stringify(store));
    } catch (e) {}
  }

  function saveScrollPosition(pathname) {
    if (!pathname) return;
    var store = readScrollStore();
    store[pathname] = window.scrollY || window.pageYOffset || 0;
    writeScrollStore(store);
  }

  function getSavedScrollPosition(pathname) {
    if (!pathname) return null;
    var store = readScrollStore();
    return typeof store[pathname] === 'number' ? store[pathname] : null;
  }

  function expandGroupChain(group) {
    var current = group;
    while (current) {
      collapseSiblingGroups(current);
      setGroupExpanded(current, true);
      var parent = current.parentElement ? current.parentElement.closest('.nav-group') : null;
      if (!parent) break;
      current = parent;
    }
  }

  function setGroupExpanded(group, shouldExpand) {
    if (!group || !group.querySelector('.nav-children')) return;
    group.classList.toggle('collapsed', !shouldExpand);
    group.classList.toggle('expanded', shouldExpand);
    var toggle = group.querySelector(':scope > .nav-row > .nav-toggle');
    if (toggle) toggle.setAttribute('aria-expanded', shouldExpand ? 'true' : 'false');
  }

  function collapseSiblingGroups(group) {
    if (!group || !group.parentElement) return;
    var container = group.parentElement;
    var siblings = container.querySelectorAll(':scope > .nav-group');
    for (var i = 0; i < siblings.length; i++) {
      if (siblings[i] !== group && siblings[i].classList.contains('expanded')) {
        collapseGroupRecursive(siblings[i]);
      }
    }
  }

  function collapseGroupRecursive(group) {
    var childGroups = group.querySelectorAll('.nav-group.expanded');
    for (var i = 0; i < childGroups.length; i++) {
      setGroupExpanded(childGroups[i], false);
    }
    setGroupExpanded(group, false);
  }

  function toggleGroup(group) {
    if (!group) return;
    var shouldExpand = group.classList.contains('collapsed');
    if (shouldExpand) {
      collapseSiblingGroups(group);
    }
    setGroupExpanded(group, shouldExpand);
  }

  function smoothScrollToElement(element, hash) {
    if (!element) return;
    element.scrollIntoView({
      behavior: prefersReducedMotion ? 'auto' : 'smooth',
      block: 'start'
    });
    if (hash) {
      window.history.pushState(null, '', hash);
    }
  }

  function scrollToHashTarget(hash, shouldPushHistory) {
    if (!hash) {
      window.scrollTo({ top: 0, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
      return;
    }

    var targetId = hash.charAt(0) === '#' ? hash.slice(1) : hash;
    var target = targetId ? document.getElementById(targetId) : null;
    if (!target) {
      if (shouldPushHistory) {
        window.history.pushState(null, '', '#' + targetId);
      }
      return;
    }

    target.scrollIntoView({
      behavior: prefersReducedMotion ? 'auto' : 'smooth',
      block: 'start'
    });
    if (shouldPushHistory) {
      window.history.pushState(null, '', '#' + targetId);
    }
  }

  function keepActiveLinkVisible(link) {
    if (!link || !sidebar) return;
    if (lastActiveLink === link) return;
    lastActiveLink = link;
    link.scrollIntoView({
      block: 'nearest',
      behavior: prefersReducedMotion ? 'auto' : 'smooth'
    });
  }

  function scrollToTopImmediate() {
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
    document.documentElement.scrollTop = 0;
    document.body.scrollTop = 0;
  }

  document.querySelectorAll('.nav-group').forEach(function(group) {
    var toggle = group.querySelector('.nav-toggle');
    var chapterLink = group.querySelector('.nav-chapter[data-group-link="true"]');

    if (toggle) {
      toggle.addEventListener('click', function(e) {
        e.preventDefault();
        e.stopPropagation();
        toggleGroup(group);
      });
    }

    if (chapterLink) {
      chapterLink.addEventListener('pointerenter', function() {
        prefetchPage(chapterLink.href);
      }, { passive: true });
      chapterLink.addEventListener('focus', function() {
        prefetchPage(chapterLink.href);
      });
      chapterLink.addEventListener('touchstart', function() {
        prefetchPage(chapterLink.href);
      }, { passive: true });
      chapterLink.addEventListener('click', function(e) {
        if (group.classList.contains('expanded')) {
          toggleGroup(group);
        } else {
          expandGroupChain(group);
        }
        rememberInternalNavigation(chapterLink.href);
        if (chapterLink.getAttribute('data-file') === currentFile) {
          e.preventDefault();
          window.scrollTo({ top: 0, behavior: 'auto' });
        }
      });
    }
  });

  // --- Heading tracking via IntersectionObserver ---
  var headingObserver = null;
  var visibleHeadings = Object.create(null); // id -> true/false

  function activateNavForHeading(headingId) {
    document.querySelectorAll('.nav-item.active').forEach(function(link) {
      link.classList.remove('active');
    });

    var matched = false;
    if (headingId) {
      for (var j = 0; j < navHeadingLinks.length; j++) {
        if (navHeadingLinks[j].getAttribute('data-target') === headingId) {
          navHeadingLinks[j].classList.add('active');
          var activeGroup = navHeadingLinks[j].closest('.nav-group');
          expandGroupChain(activeGroup);
          keepActiveLinkVisible(navHeadingLinks[j]);
          matched = true;
          break;
        }
      }
    }

    if (!matched && navChapterLinks.length > 0) {
      navChapterLinks[0].classList.add('active');
      var activeChapterGroup = navChapterLinks[0].closest('.nav-group');
      expandGroupChain(activeChapterGroup);
      keepActiveLinkVisible(navChapterLinks[0]);
    }
  }

  function pickActiveHeading() {
    // Among headings in or above the viewport, pick the last one that is visible
    // (i.e. the deepest one the user has scrolled past).
    for (var i = headings.length - 1; i >= 0; i--) {
      if (visibleHeadings[headings[i].id]) {
        return headings[i].id;
      }
    }
    // Fallback: find topmost heading above viewport
    for (var k = headings.length - 1; k >= 0; k--) {
      if (headings[k].getBoundingClientRect().top <= 140) {
        return headings[k].id;
      }
    }
    return null;
  }

  function setupHeadingObserver() {
    if (headingObserver) { headingObserver.disconnect(); }
    visibleHeadings = Object.create(null);

    if (typeof IntersectionObserver === 'undefined') {
      // Fallback for older browsers: use scroll event.
      window.addEventListener('scroll', function() {
        activateNavForHeading(pickActiveHeading());
      }, { passive: true });
      return;
    }

    headingObserver = new IntersectionObserver(function(entries) {
      entries.forEach(function(entry) {
        visibleHeadings[entry.target.id] = entry.isIntersecting;
      });
      activateNavForHeading(pickActiveHeading());
    }, { rootMargin: '-80px 0px -60% 0px', threshold: 0 });

    headings.forEach(function(h) { headingObserver.observe(h); });
  }

  function updateActiveNavigation() {
    activateNavForHeading(pickActiveHeading());
  }

  function syncSidebarForCurrentFile() {
    document.querySelectorAll('.nav-group[data-group-file]').forEach(function(group) {
      if (group.getAttribute('data-group-file') === currentFile) {
        expandGroupChain(group);
      }
    });
  }

  // --- Right-side page TOC ---
  var pageToc = document.getElementById('page-toc');
  var pageTocNav = document.getElementById('page-toc-nav');
  var tocObserver = null;
  var tocVisibleMap = Object.create(null);

  // Single click handler via delegation (never re-attached).
  if (pageTocNav) {
    pageTocNav.addEventListener('click', function(e) {
      var link = e.target.closest('a[data-toc-target]');
      if (!link) return;
      e.preventDefault();
      var target = document.getElementById(link.getAttribute('data-toc-target'));
      if (target) {
        target.scrollIntoView({ behavior: prefersReducedMotion ? 'auto' : 'smooth', block: 'start' });
        window.history.pushState(null, '', '#' + link.getAttribute('data-toc-target'));
      }
    });
  }

  function headingTextForTOC(heading) {
    if (!heading) return '';
    var clone = heading.cloneNode(true);
    clone.querySelectorAll('.header-anchor').forEach(function(anchor) {
      anchor.remove();
    });
    return (clone.textContent || '').trim();
  }

  function buildPageTOC() {
    if (!pageTocNav || !pageToc) return;
    var tocHeadings = Array.from(document.querySelectorAll('.content h2[id], .content h3[id], .content h4[id]'));
    if (tocHeadings.length === 0) {
      pageToc.classList.add('toc-hidden');
      pageTocNav.innerHTML = '';
      return;
    }
    pageToc.classList.remove('toc-hidden');
    var html = '';
    for (var i = 0; i < tocHeadings.length; i++) {
      var h = tocHeadings[i];
      var tag = h.tagName.toLowerCase();
      var depthClass = tag === 'h3' ? ' toc-depth-2' : tag === 'h4' ? ' toc-depth-3' : '';
      html += '<a href="#' + h.id + '" data-toc-target="' + h.id + '" class="toc-link' + depthClass + '">' + headingTextForTOC(h) + '</a>';
    }
    pageTocNav.innerHTML = html;
    setupTocObserver(tocHeadings);
  }

  function setupTocObserver(tocHeadings) {
    if (tocObserver) tocObserver.disconnect();
    tocVisibleMap = Object.create(null);

    if (typeof IntersectionObserver === 'undefined' || !pageTocNav) return;

    tocObserver = new IntersectionObserver(function(entries) {
      entries.forEach(function(entry) {
        tocVisibleMap[entry.target.id] = entry.isIntersecting;
      });
      // Pick the topmost visible heading
      var activeId = null;
      for (var i = 0; i < tocHeadings.length; i++) {
        if (tocVisibleMap[tocHeadings[i].id]) { activeId = tocHeadings[i].id; break; }
      }
      if (!activeId) {
        // Fallback: find topmost heading above viewport
        for (var k = tocHeadings.length - 1; k >= 0; k--) {
          if (tocHeadings[k].getBoundingClientRect().top <= 140) { activeId = tocHeadings[k].id; break; }
        }
      }
      var links = pageTocNav.querySelectorAll('.toc-link');
      for (var j = 0; j < links.length; j++) {
        links[j].classList.toggle('toc-active', links[j].getAttribute('data-toc-target') === activeId);
      }
    }, { rootMargin: '-80px 0px -60% 0px', threshold: 0 });

    tocHeadings.forEach(function(h) { tocObserver.observe(h); });
  }

  var resizeTimer = null;
  function scheduleNavigationUpdate() {
    if (navUpdateFrame !== null) return;
    navUpdateFrame = window.requestAnimationFrame(function() {
      updateActiveNavigation();
      navUpdateFrame = null;
    });
  }

  window.addEventListener('resize', function() {
    if (resizeTimer) clearTimeout(resizeTimer);
    resizeTimer = setTimeout(scheduleNavigationUpdate, 200);
  });
  window.addEventListener('hashchange', function() {
    scheduleNavigationUpdate();
  });

  // loadCDNScript is a helper to load a script from CDN only once.
  // tag: data attribute name used to deduplicate; src: CDN URL; onReady: callback.
  function loadCDNScript(tag, src, onReady) {
    var attrName = 'data-mdpress-' + tag.replace(/[A-Z]/g, function(ch) {
      return '-' + ch.toLowerCase();
    });
    var existing = document.querySelector('script[' + attrName + ']');
    if (existing) {
      if (existing.dataset.mdpressLoaded === 'true') {
        if (onReady) onReady();
      } else if (onReady) {
        existing.addEventListener('load', onReady, { once: true });
      }
      return;
    }

    var s = document.createElement('script');
    s.src = src;
    s.crossOrigin = 'anonymous';
    s.referrerPolicy = 'no-referrer';
    s.setAttribute(attrName, 'true');
    s.addEventListener('load', function() {
      s.dataset.mdpressLoaded = 'true';
      if (onReady) onReady();
    }, { once: true });
    document.body.appendChild(s);
  }

  function ensureMermaid() {
    var nodes = document.querySelectorAll('.mermaid');
    if (!nodes.length) return;

    function runMermaid() {
      if (!window.mermaid) return;
      try {
        window.mermaid.initialize({ startOnLoad: true, theme: 'default', securityLevel: 'strict', themeVariables: { fontFamily: '"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans SC","Noto Sans CJK SC","Source Han Sans SC",sans-serif' } });
        if (window.mermaid.run) {
          window.mermaid.run({ nodes: nodes });
        } else if (window.mermaid.init) {
          window.mermaid.init(undefined, nodes);
        }
      } catch (e) {
        console.warn('[mdpress] Mermaid re-init failed', e);
      }
    }

    if (window.mermaid) { runMermaid(); return; }
    loadCDNScript('mermaid', '` + utils.MermaidCDNURL + `', runMermaid);
  }

  // ensureKaTeX loads KaTeX and triggers auto-render when math elements are found.
  // Called on initial load and after each client-side navigation.
  function ensureKaTeX() {
    if (!document.querySelector('.math')) return;

    function runKaTeX() {
      if (typeof renderMathInElement !== 'function') return;
      try {
        renderMathInElement(document.body, {
          delimiters: [
            {left: '$$', right: '$$', display: true},
            {left: '$',  right: '$',  display: false}
          ],
          throwOnError: false
        });
      } catch (e) {
        console.warn('[mdpress] KaTeX render failed', e);
      }
    }

    if (typeof renderMathInElement === 'function') { runKaTeX(); return; }

    // Load KaTeX CSS if not already loaded.
    if (!document.querySelector('link[data-mdpress-katex-css]')) {
      var link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = '` + utils.KaTeXCSSURL + `';
      link.crossOrigin = 'anonymous';
      link.referrerPolicy = 'no-referrer';
      link.dataset.mdpressKatexCss = 'true';
      document.head.appendChild(link);
    }

    loadCDNScript('katex', '` + utils.KaTeXJSURL + `', function() {
      loadCDNScript('katexAutoRender', '` + utils.KaTeXAutoRenderURL + `', runKaTeX);
    });
  }

  function getClientNavigation(anchor) {
    if (!anchor || !anchor.href) return null;
    try {
      var url = new URL(anchor.href, window.location.href);
      if (url.origin !== window.location.origin) return null;
      if (!/\.html$/i.test(url.pathname) && url.pathname !== window.location.pathname) return null;
      return {
        url: url,
        file: getFileFromPathname(url.pathname),
        hash: url.hash || ''
      };
    } catch (e) {
      return null;
    }
  }

  function parseFetchedPage(html, fallbackURL) {
    var doc = new DOMParser().parseFromString(html, 'text/html');
    var content = doc.querySelector('.content');
    if (!content) return null;
    var breadcrumbNav = doc.querySelector('.page-breadcrumb');
    var breadcrumbHTML = breadcrumbNav ? breadcrumbNav.innerHTML : '';
    return {
      title: doc.title || document.title,
      contentHTML: content.innerHTML,
      breadcrumbHTML: breadcrumbHTML,
      url: fallbackURL
    };
  }

  function getCachedPage(cacheKey) {
    return pageCache[cacheKey] || null;
  }

  function cachePage(cacheKey, payload) {
    pageCache[cacheKey] = payload;
    return payload;
  }

  function fetchPagePayload(targetURL, signal) {
    var cacheKey = targetURL.origin + targetURL.pathname;
    var cached = getCachedPage(cacheKey);
    if (cached) return Promise.resolve(cached);

    return fetch(cacheKey, {
      credentials: 'same-origin',
      signal: signal
    }).then(function(response) {
      if (!response.ok) throw new Error('HTTP ' + response.status);
      return response.text().then(function(html) {
        var responseURL = new URL(response.url || cacheKey, window.location.href);
        var payload = parseFetchedPage(html, responseURL);
        if (!payload) throw new Error('Missing .content in fetched page');
        return cachePage(cacheKey, payload);
      });
    });
  }

  function finalizeNavigation(targetURL, options) {
    currentFile = getFileFromPathname(targetURL.pathname);
    refreshPageContext();
    syncSidebarForCurrentFile();
    setupHeadingObserver();
    // buildPageTOC + addCopyButtons are called inside applySwap() so they
    // run before the new frame is painted (inside startViewTransition).
    updateActiveNavigation();
    ensureMermaid();
    ensureKaTeX();

    if (window.innerWidth <= 768) {
      sidebar.classList.remove('open');
    }

    if (options.updateHistory === 'push') {
      window.history.pushState({ path: targetURL.pathname }, '', targetURL.pathname + targetURL.search + targetURL.hash);
    } else if (options.updateHistory === 'replace') {
      window.history.replaceState({ path: targetURL.pathname }, '', targetURL.pathname + targetURL.search + targetURL.hash);
    }

    if (options.hash) {
      scrollToHashTarget(options.hash, false);
    } else if (options.restoreScroll === true) {
      var savedScroll = getSavedScrollPosition(targetURL.pathname);
      window.scrollTo({
        top: savedScroll || 0,
        behavior: prefersReducedMotion ? 'auto' : 'smooth'
      });
    } else if (options.scrollToTop !== false) {
      window.scrollTo({ top: 0, behavior: 'auto' });
    }

    if (typeof saveRecentPage === 'function') {
      saveRecentPage();
    }
    if (typeof window.__mdpressRefreshServePanel === 'function') {
      window.__mdpressRefreshServePanel();
    }
  }

  function swapPageContent(payload, targetURL, options) {
    function applySwap() {
      mainContent.innerHTML = payload.contentHTML;
      document.title = payload.title;
      var breadcrumbNav = document.querySelector('.page-breadcrumb');
      if (breadcrumbNav && payload.breadcrumbHTML) {
        breadcrumbNav.innerHTML = payload.breadcrumbHTML;
      }
      // Mutate DOM while the old snapshot is still displayed (inside the
      // view-transition callback) so wrapping <pre> and building the TOC
      // never cause a visible layout shift.
      if (window.__addCopyButtons) window.__addCopyButtons(mainContent);
      if (window.__addHeaderAnchors) window.__addHeaderAnchors(mainContent);
      buildPageTOC();
    }

    applySwap();
    finalizeNavigation(targetURL, options);
    return Promise.resolve();
  }

  function navigateClientSide(target, options) {
    options = options || {};
    if (!mainContent) {
      window.location.href = target.url.toString();
      return Promise.resolve();
    }

    if (pendingNavigation) {
      pendingNavigation.abort();
    }
    saveScrollPosition(window.location.pathname);
    if (!target.hash && options.scrollToTop !== false && options.restoreScroll !== true) {
      scrollToTopImmediate();
    }
    pendingNavigation = new AbortController();
    setNavigating(true);
    beginRouteProgress();

    return fetchPagePayload(target.url, pendingNavigation.signal)
      .then(function(payload) {
        if (pendingNavigation.signal.aborted) return;
        var targetURL = new URL(target.url.toString(), window.location.href);
        return swapPageContent(payload, targetURL, {
          updateHistory: options.updateHistory || 'push',
          hash: target.hash,
          scrollToTop: options.scrollToTop,
          restoreScroll: options.restoreScroll === true
        });
      })
      .catch(function(err) {
        if (err && err.name === 'AbortError') return;
        console.warn('[mdpress] Falling back to full navigation', err);
        window.location.href = target.url.toString();
      })
      .finally(function() {
        setNavigating(false);
        endRouteProgress();
      });
  }

  refreshPageContext();
  window.history.replaceState({ path: window.location.pathname }, '', window.location.pathname + window.location.search + window.location.hash);
  syncSidebarForCurrentFile();
  setupHeadingObserver();
  buildPageTOC();
  if (window.__addCopyButtons) window.__addCopyButtons(mainContent);
  updateActiveNavigation();
  ensureMermaid();
  ensureKaTeX();

  document.addEventListener('mouseover', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  }, { passive: true });

  document.addEventListener('focusin', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  });

  document.addEventListener('touchstart', function(e) {
    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    prefetchPage(link.href);
    warmPageCache(link.href);
  }, { passive: true });

  document.addEventListener('click', function(e) {
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;

    var link = e.target.closest('.sidebar-nav a, .page-nav a, .content a');
    if (!link) return;
    if (link.target && link.target !== '_self') return;
    if (link.hasAttribute('download')) return;

    var target = getClientNavigation(link);
    if (!target) return;

    rememberInternalNavigation(link.href);

    if (target.file === currentFile) {
      if (target.hash) {
        var samePageTarget = document.getElementById(target.hash.slice(1));
        if (samePageTarget) {
          e.preventDefault();
          expandGroupChain(link.closest('.nav-group'));
          scrollToHashTarget(target.hash, true);
          scheduleNavigationUpdate();
        }
      } else if (target.url.pathname === window.location.pathname) {
        e.preventDefault();
        window.scrollTo({ top: 0, behavior: 'auto' });
      }
      return;
    }

    e.preventDefault();
    expandGroupChain(link.closest('.nav-group'));
    navigateClientSide(target, {
      updateHistory: 'push',
      scrollToTop: !target.hash
    });
  });

  window.addEventListener('popstate', function() {
    var target = getClientNavigation({ href: window.location.href });
    if (!target) return;
    if (target.file === currentFile) {
      if (target.hash) {
        scrollToHashTarget(target.hash, false);
      } else {
        window.scrollTo({ top: 0, behavior: 'auto' });
      }
      scheduleNavigationUpdate();
      return;
    }
    navigateClientSide(target, {
      updateHistory: null,
      scrollToTop: !target.hash,
      restoreScroll: !target.hash
    });
  });

  window.addEventListener('scroll', function() {
    if (scrollSaveFrame !== null) return;
    scrollSaveFrame = window.requestAnimationFrame(function() {
      saveScrollPosition(window.location.pathname);
      scrollSaveFrame = null;
    });
  }, { passive: true });

  // Keyboard navigation: Arrow Left/Right for prev/next, "/" for search
  document.addEventListener('keydown', function(e) {
    var navLinks = document.querySelectorAll('.page-nav a');
    var prevLink = null;
    var nextLink = null;
    for (var i = 0; i < navLinks.length; i++) {
      if (navLinks[i].classList.contains('prev')) prevLink = navLinks[i];
      if (navLinks[i].classList.contains('next')) nextLink = navLinks[i];
    }

    if (e.key === 'ArrowLeft' && prevLink && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      prevLink.click();
    } else if (e.key === 'ArrowRight' && nextLink && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      nextLink.click();
    } else if ((e.key === '/' || (e.key === 'k' && (e.metaKey || e.ctrlKey))) && e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA' && !e.target.isContentEditable) {
      e.preventDefault();
      openSearch('');
    }
  });

  // Sidebar toggle management
  var sidebarClose = document.querySelector('.sidebar-close');
  var sidebarCollapsedKey = 'mdpress-sidebar-collapsed';

  // Restore collapsed state from localStorage.
  try {
    if (window.localStorage.getItem(sidebarCollapsedKey) === '1') {
      body.classList.add('sidebar-collapsed');
    }
  } catch (e) {}

  function isMobile() { return window.innerWidth <= 768; }

  function toggleSidebar(forceState) {
    if (!sidebar) return;
    if (isMobile()) {
      if (typeof forceState === 'boolean') {
        sidebar.classList.toggle('open', forceState);
        body.classList.toggle('sidebar-open', forceState);
      } else {
        var isOpen = sidebar.classList.toggle('open');
        body.classList.toggle('sidebar-open', isOpen);
      }
    } else {
      var shouldCollapse = typeof forceState === 'boolean' ? forceState : !body.classList.contains('sidebar-collapsed');
      body.classList.toggle('sidebar-collapsed', shouldCollapse);
      try { window.localStorage.setItem(sidebarCollapsedKey, shouldCollapse ? '1' : '0'); } catch (e) {}
    }
    if (sidebarToggle) {
      var expanded = isMobile() ? sidebar.classList.contains('open') : !body.classList.contains('sidebar-collapsed');
      sidebarToggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
    }
  }

  if (sidebarToggle) {
    sidebarToggle.addEventListener('click', function(e) {
      e.preventDefault();
      toggleSidebar();
    });
  }

  if (sidebarClose) {
    sidebarClose.addEventListener('click', function(e) {
      e.preventDefault();
      toggleSidebar(true);
    });
  }

  // Close sidebar on escape key
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && sidebar) {
      if (isMobile() && sidebar.classList.contains('open')) {
        toggleSidebar(false);
      } else if (!isMobile() && !body.classList.contains('sidebar-collapsed')) {
        toggleSidebar(true);
      }
    }
  });

  // Handle sidebar close on overlay click (mobile only)
  document.addEventListener('click', function(e) {
    if (isMobile() && sidebar && sidebar.classList.contains('open') && !sidebar.contains(e.target) && sidebarToggle && !sidebarToggle.contains(e.target)) {
      toggleSidebar(false);
    }
  });

  // Sidebar resize by dragging
  (function() {
    var handle = document.getElementById('sidebar-resize-handle');
    var mainEl = document.querySelector('.main');
    if (!handle || !sidebar || !mainEl) return;
    var sidebarWidthKey = 'mdpress-sidebar-width';
    var minW = 200, maxW = Math.floor(window.innerWidth * 0.5);

    // Restore saved width
    var saved = localStorage.getItem(sidebarWidthKey);
    if (saved) {
      var w = parseInt(saved, 10);
      if (w >= minW && w <= maxW) {
        sidebar.style.width = w + 'px';
        document.documentElement.style.setProperty('--sidebar-width', w + 'px');
      }
    }

    var startX, startW;
    function onMouseMove(e) {
      var newW = Math.min(maxW, Math.max(minW, startW + (e.clientX - startX)));
      sidebar.style.width = newW + 'px';
      document.documentElement.style.setProperty('--sidebar-width', newW + 'px');
    }
    function onMouseUp() {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
      handle.classList.remove('active');
      document.body.classList.remove('sidebar-resizing');
      var finalW = parseInt(sidebar.style.width, 10);
      if (finalW) localStorage.setItem(sidebarWidthKey, finalW);
    }
    handle.addEventListener('mousedown', function(e) {
      e.preventDefault();
      startX = e.clientX;
      startW = sidebar.getBoundingClientRect().width;
      maxW = Math.floor(window.innerWidth * 0.5);
      handle.classList.add('active');
      document.body.classList.add('sidebar-resizing');
      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
    });
    // Double-click to reset width
    handle.addEventListener('dblclick', function() {
      sidebar.style.width = '';
      document.documentElement.style.setProperty('--sidebar-width', '280px');
      localStorage.removeItem(sidebarWidthKey);
    });
  })();

  /* ===== Full-Text Search ===== */
  (function() {
    var overlay = document.getElementById('search-overlay');
    var modalInput = document.getElementById('search-input');
    var resultsBox = document.getElementById('search-results');
    var searchStatus = document.getElementById('search-status');
    var recentPagesKey = 'mdpress-recent-pages';
    var searchJumpKey = 'mdpress-search-jump';
    var searchIndex = null;
    var activeIdx = -1;
    var debounceTimer = null;
    var homeLink = document.querySelector('.sidebar-home-link');
    var basePath = (homeLink ? homeLink.getAttribute('href') : '/').replace(/[^/]*$/, '');

    function loadIndex() {
      if (searchIndex) return Promise.resolve(searchIndex);
      return fetch(basePath + 'search-index.json').then(function(r) {
        if (!r.ok) throw new Error('HTTP ' + r.status);
        return r.json();
      }).then(function(data) {
        searchIndex = data;
        return data;
      }).catch(function(err) {
        console.warn('[mdpress] Failed to load search index:', err);
        searchIndex = [];
        return searchIndex;
      });
    }

    function updateSearchStatus(count) {
      if (!searchStatus) return;
      if (typeof count === 'string') {
        searchStatus.textContent = count;
        return;
      }
      if (count < 0) {
        searchStatus.textContent = '';
        return;
      }
      searchStatus.textContent = count === 1 ? __ui.searchResultsOne : __ui.searchResults.replace('%d', String(count));
    }

    function getRecentPages() {
      try {
        var raw = window.localStorage.getItem(recentPagesKey);
        if (!raw) return [];
        var parsed = JSON.parse(raw);
        return Array.isArray(parsed) ? parsed : [];
      } catch (e) {
        return [];
      }
    }

    function saveRecentPage() {
      try {
        var titleNode = document.querySelector('.chapter-title, .content h1');
        var title = titleNode ? titleNode.textContent.trim() : (document.title || '').replace(/\s+-\s+.*$/, '');
        var href = window.location.pathname.replace(/^\//, '') || 'index.html';
        if (href === 'index.html') return;
        var path = Array.from(document.querySelectorAll('.page-breadcrumb a')).slice(0, -1).map(function(node) { return node.textContent.trim(); }).join(' > ');
        if (!title || !href) return;
        var recent = getRecentPages().filter(function(item) { return item && item.href !== href; });
        recent.unshift({ title: title, href: href, path: path });
        window.localStorage.setItem(recentPagesKey, JSON.stringify(recent.slice(0, 5)));
      } catch (e) {}
    }

    function showSearchJumpNotice() {
      try {
        var raw = sessionStorage.getItem(searchJumpKey);
        if (!raw) return;
        sessionStorage.removeItem(searchJumpKey);
        var query = JSON.parse(raw);
        if (!query) return;
        var content = document.querySelector('.content');
        if (!content) return;
        var notice = document.createElement('div');
        notice.className = 'search-jump-notice';
        notice.textContent = __ui.searchMatched.replace('%s', query);
        content.insertBefore(notice, content.firstChild);
        setTimeout(function() {
          if (notice.parentNode) notice.parentNode.removeChild(notice);
        }, 2400);
      } catch (e) {}
    }

    function renderRecentPages() {
      var recent = getRecentPages();
      if (!recent.length) {
        resultsBox.innerHTML = '<div class="search-empty">' + __ui.recentEmpty + '</div>';
        updateSearchStatus(__ui.recentPages);
        activeIdx = -1;
        return;
      }
      var html = '';
      for (var i = 0; i < recent.length; i++) {
        var item = recent[i];
        html += '<a class="search-result" href="' + escapeHTML(item.href) + '">';
        if (item.path) html += '<div class="search-result-path">' + escapeHTML(item.path) + '</div>';
        html += '<div class="search-result-title">' + escapeHTML(item.title) + '</div>';
        html += '</a>';
      }
      resultsBox.innerHTML = html;
      activeIdx = 0;
      updateActive(resultsBox.querySelectorAll('.search-result'));
      updateSearchStatus(__ui.recentPages);
    }

    // Header search button opens modal
    var headerSearchBtn = document.getElementById('header-search-btn');
    if (headerSearchBtn) {
      headerSearchBtn.addEventListener('click', function() {
        openSearch('');
      });
    }

    var backdrop = document.getElementById('search-backdrop');

    window.openSearch = function(initialQuery) {
      overlay.classList.add('open');
      backdrop.classList.add('open');
      modalInput.value = initialQuery || '';
      activeIdx = -1;
      loadIndex().catch(function() {});
      if (initialQuery) {
        doSearch();
      } else {
        renderRecentPages();
      }
      requestAnimationFrame(function() { modalInput.focus(); });
    };

    function closeSearch() {
      overlay.classList.remove('open');
      backdrop.classList.remove('open');
      activeIdx = -1;
    }

    backdrop.addEventListener('click', function() {
      closeSearch();
    });

    var leaveTimer = null;
    overlay.addEventListener('mouseleave', function() {
      if (document.activeElement === modalInput) return;
      leaveTimer = setTimeout(closeSearch, 500);
    });
    overlay.addEventListener('mouseenter', function() {
      if (leaveTimer) { clearTimeout(leaveTimer); leaveTimer = null; }
    });

    modalInput.addEventListener('keydown', function(e) {
      if (e.key === 'Escape') { closeSearch(); return; }
      var items = resultsBox.querySelectorAll('.search-result');
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        activeIdx = Math.min(activeIdx + 1, items.length - 1);
        updateActive(items);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        activeIdx = Math.max(activeIdx - 1, 0);
        updateActive(items);
      } else if (e.key === 'Enter' && activeIdx >= 0 && items[activeIdx]) {
        e.preventDefault();
        items[activeIdx].click();
      }
    });

    function updateActive(items) {
      for (var i = 0; i < items.length; i++) {
        items[i].classList.toggle('search-active', i === activeIdx);
      }
      if (items[activeIdx]) items[activeIdx].scrollIntoView({ block: 'nearest' });
    }

    modalInput.addEventListener('input', function() {
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(doSearch, 80);
    });

    function escapeHTML(s) {
      return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    function buildSnippet(text, query, maxLen) {
      var lower = text.toLowerCase();
      var qLower = query.toLowerCase();
      var idx = lower.indexOf(qLower);
      if (idx < 0) return '';
      var start = Math.max(0, idx - 40);
      var end = Math.min(text.length, idx + query.length + 80);
      var snippet = (start > 0 ? '\u2026' : '') + text.slice(start, end) + (end < text.length ? '\u2026' : '');
      // Highlight matches
      var re = new RegExp('(' + query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'gi');
      return escapeHTML(snippet).replace(re, '<mark>$1</mark>');
    }

    function doSearch() {
      var query = modalInput.value.trim();
      if (!query) {
        renderRecentPages();
        return;
      }
      loadIndex().then(function(index) {
        var qLower = query.toLowerCase();
        var matches = [];
        for (var i = 0; i < index.length; i++) {
          var entry = index[i];
          var titleMatch = entry.t.toLowerCase().indexOf(qLower) >= 0;
          var pathMatch = (entry.p || '').toLowerCase().indexOf(qLower) >= 0;
          var textMatch = entry.x.toLowerCase().indexOf(qLower) >= 0;
          if (titleMatch || pathMatch || textMatch) {
            matches.push({
              title: entry.t,
              filename: entry.f,
              path: entry.p || '',
              snippet: buildSnippet(entry.x, query),
              titleMatch: titleMatch,
              pathMatch: pathMatch
            });
          }
          if (matches.length >= 20) break;
        }
        // Sort by title > breadcrumb path > body
        matches.sort(function(a, b) {
          var aScore = (a.titleMatch ? 2 : 0) + (a.pathMatch ? 1 : 0);
          var bScore = (b.titleMatch ? 2 : 0) + (b.pathMatch ? 1 : 0);
          return bScore - aScore;
        });
        updateSearchStatus(matches.length);

        if (matches.length === 0) {
          resultsBox.innerHTML = '<div class="search-empty">' + __ui.noResults + ' \u201c' + escapeHTML(query) + '\u201d</div>';
          activeIdx = -1;
          return;
        }

        var html = '';
        for (var j = 0; j < matches.length; j++) {
          var m = matches[j];
          var badges = [];
          if (m.titleMatch) badges.push(__ui.searchMatchTitle);
          if (m.pathMatch) badges.push(__ui.searchMatchPath);
          if (!m.titleMatch && !m.pathMatch) badges.push(__ui.searchMatchText);
          html += '<a class="search-result" href="' + escapeHTML(basePath + m.filename) + '">';
          if (badges.length) {
            html += '<div class="search-result-meta">';
            for (var k = 0; k < badges.length; k++) {
              html += '<span class="search-badge">' + escapeHTML(badges[k]) + '</span>';
            }
            html += '</div>';
          }
          if (m.path) html += '<div class="search-result-path">' + escapeHTML(m.path) + '</div>';
          html += '<div class="search-result-title">' + escapeHTML(m.title) + '</div>';
          if (m.snippet) html += '<div class="search-result-snippet">' + m.snippet + '</div>';
          html += '</a>';
        }
        resultsBox.innerHTML = html;
        activeIdx = 0;
        updateActive(resultsBox.querySelectorAll('.search-result'));
      }).catch(function() {
        resultsBox.innerHTML = '<div class="search-empty">' + __ui.searchUnavailable + '</div>';
        updateSearchStatus(-1);
      });
    }

    saveRecentPage();
    showSearchJumpNotice();

    // Click result → navigate via SPA
    resultsBox.addEventListener('click', function(e) {
      var link = e.target.closest('.search-result');
      if (!link) return;
      e.preventDefault();
      closeSearch();
      var href = link.getAttribute('href');
      try {
        sessionStorage.setItem(searchJumpKey, JSON.stringify(modalInput.value.trim()));
      } catch (e) {}
      // Use SPA navigation if available
      if (typeof navigateClientSide === 'function') {
        var url = new URL(href, window.location.href);
        navigateClientSide({ url: url });
      } else {
        window.location.href = href;
      }
    });

    // Global ESC closes search
    document.addEventListener('keydown', function(e) {
      if (e.key === 'Escape' && overlay.classList.contains('open')) {
        closeSearch();
      }
    });
  })();

  /* ===== Code Block Copy Buttons ===== */
  (function() {
    function addCopyButtons(root) {
      var pres = (root || document).querySelectorAll('pre');
      for (var i = 0; i < pres.length; i++) {
        var pre = pres[i];
        if (pre.parentNode.classList.contains('code-wrapper')) continue;
        var wrapper = document.createElement('div');
        wrapper.className = 'code-wrapper';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);
        var btn = document.createElement('button');
        btn.className = 'copy-btn';
        btn.textContent = __ui.copy;
        btn.type = 'button';
        btn.setAttribute('aria-label', __ui.copy);
        wrapper.appendChild(btn);
      }
    }
    document.addEventListener('click', function(e) {
      var btn = e.target.closest('.copy-btn');
      if (!btn) return;
      var pre = btn.parentNode.querySelector('pre');
      if (!pre) return;
      var text = pre.textContent || pre.innerText;
      navigator.clipboard.writeText(text).then(function() {
        btn.textContent = __ui.copied;
        btn.classList.add('copied');
        setTimeout(function() { btn.textContent = __ui.copy; btn.classList.remove('copied'); }, 2000);
      });
    });
    addCopyButtons();
    window.__addCopyButtons = addCopyButtons;
  })();
  </script>
</body>
</html>`
