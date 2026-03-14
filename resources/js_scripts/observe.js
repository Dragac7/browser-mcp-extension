// observe.js — Extracts a structured, AI-readable snapshot of the current page.
// Returns a JSON string with URL, title, interactive elements map, visible text,
// and page sections. Each interactive element receives a unique numeric index
// that can be used by interact.js to target it.

(function () {
  'use strict';

  const MAX_TEXT_LENGTH = 200;
  const MAX_VISIBLE_TEXT = 5000;

  // ── Helpers ──────────────────────────────────────────────────────────

  function isVisible(el) {
    if (!el || !el.getBoundingClientRect) return false;
    const style = window.getComputedStyle(el);
    if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return false;
    const rect = el.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) return false;
    return true;
  }

  function trimText(str, max) {
    if (!str) return '';
    const cleaned = str.replace(/\s+/g, ' ').trim();
    if (cleaned.length <= max) return cleaned;
    return cleaned.slice(0, max) + '…';
  }

  function getElementText(el) {
    // Prefer aria-label, then textContent, then value, then placeholder
    return el.getAttribute('aria-label')
      || el.textContent?.trim()
      || el.value
      || el.getAttribute('placeholder')
      || '';
  }

  function getElementType(el) {
    const tag = el.tagName.toLowerCase();
    if (tag === 'a') return 'link';
    if (tag === 'button' || el.getAttribute('role') === 'button') return 'button';
    if (tag === 'input') return 'input';
    if (tag === 'textarea') return 'textarea';
    if (tag === 'select') return 'select';
    if (el.getAttribute('role') === 'link') return 'link';
    if (el.getAttribute('role') === 'tab') return 'tab';
    if (el.getAttribute('role') === 'menuitem') return 'menuitem';
    if (el.getAttribute('role') === 'checkbox') return 'checkbox';
    if (el.getAttribute('role') === 'radio') return 'radio';
    if (el.getAttribute('role') === 'switch') return 'switch';
    if (el.getAttribute('contenteditable') === 'true') return 'editable';
    if (el.onclick || el.getAttribute('tabindex') !== null) return 'clickable';
    return tag;
  }

  function getImgAlt(el) {
    const img = el.querySelector('img');
    if (img) return img.getAttribute('alt') || '';
    return '';
  }

  function getSvgLabel(el) {
    const svg = el.querySelector('svg');
    if (svg) return svg.getAttribute('aria-label') || '';
    return '';
  }

  // ── Interactive elements ─────────────────────────────────────────────

  // Selectors for interactive elements
  const INTERACTIVE_SELECTORS = [
    'a[href]',
    'button',
    'input',
    'textarea',
    'select',
    '[role="button"]',
    '[role="link"]',
    '[role="tab"]',
    '[role="menuitem"]',
    '[role="checkbox"]',
    '[role="radio"]',
    '[role="switch"]',
    '[contenteditable="true"]',
  ].join(', ');

  const allInteractive = [...document.querySelectorAll(INTERACTIVE_SELECTORS)];

  // Filter to visible, non-duplicate elements
  const seen = new Set();
  const interactiveElements = [];

  for (const el of allInteractive) {
    if (!isVisible(el)) continue;
    // Deduplicate by reference
    if (seen.has(el)) continue;
    seen.add(el);

    const tag = el.tagName.toLowerCase();
    const type = getElementType(el);
    const rawText = getElementText(el);
    const text = trimText(rawText, MAX_TEXT_LENGTH);

    const entry = {
      index: interactiveElements.length,
      tag: tag,
      type: type,
      text: text,
    };

    // Conditional fields — only include when present to keep output compact
    if (tag === 'a' && el.href) {
      try {
        const url = new URL(el.href);
        entry.href = url.pathname + url.search + url.hash;
      } catch (_) {
        entry.href = el.getAttribute('href') || '';
      }
    }

    if (tag === 'input' || tag === 'textarea') {
      if (el.placeholder) entry.placeholder = el.placeholder;
      if (el.type && el.type !== 'text') entry.inputType = el.type;
      if (el.value) entry.value = trimText(el.value, 100);
    }

    const ariaLabel = el.getAttribute('aria-label');
    if (ariaLabel && ariaLabel !== text) entry.ariaLabel = ariaLabel;

    const imgAlt = getImgAlt(el);
    if (imgAlt) entry.imgAlt = trimText(imgAlt, 100);

    const svgLabel = getSvgLabel(el);
    if (svgLabel) entry.svgLabel = svgLabel;

    const role = el.getAttribute('role');
    if (role) entry.role = role;

    interactiveElements.push(entry);
  }

  // Store element references on the window so interact.js can find them by index
  window.__observedElements = interactiveElements.map((_, idx) => {
    let count = 0;
    for (const el of allInteractive) {
      if (!isVisible(el)) continue;
      if (count === idx) return el;
      count++;
    }
    return null;
  });

  // ── Sections / visible text ──────────────────────────────────────────

  const sections = [];
  const sectionEls = document.querySelectorAll(
    'nav, main, header, footer, aside, [role="navigation"], [role="main"], [role="banner"], [role="contentinfo"], article, section'
  );

  for (const sec of sectionEls) {
    if (!isVisible(sec)) continue;
    const role = sec.getAttribute('role') || sec.tagName.toLowerCase();
    const text = trimText(sec.innerText || sec.textContent || '', 500);
    if (text.length < 5) continue; // skip near-empty sections
    sections.push({ role, text });
  }

  // Fallback: if no landmark sections found, grab the body text
  if (sections.length === 0) {
    sections.push({
      role: 'body',
      text: trimText(document.body.innerText || '', MAX_VISIBLE_TEXT),
    });
  }

  // Overall visible text (truncated)
  const visibleText = trimText(document.body.innerText || '', MAX_VISIBLE_TEXT);

  // ── Assemble snapshot ────────────────────────────────────────────────

  const snapshot = {
    url: window.location.href,
    title: document.title,
    timestamp: new Date().toISOString(),
    interactiveElements: interactiveElements,
    totalInteractiveElements: interactiveElements.length,
    visibleText: visibleText,
    sections: sections,
  };

  return JSON.stringify(snapshot);
})();
