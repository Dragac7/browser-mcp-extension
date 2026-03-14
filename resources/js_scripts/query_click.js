// query_click.js — Find and interact with elements by CSS selector + optional text match.
// Does NOT depend on observe.js or window.__observedElements.
// Useful for elements not captured by observe.js (radio buttons, hidden inputs, etc.).
//
// Params:
//   selector     — CSS selector to find elements (e.g. "label", "input[type=radio]", "button")
//   text         — (optional) text content to match (case-insensitive, partial match)
//   textExact    — (optional) if true, match text exactly instead of partial
//   action       — "click" | "type" | "hover" | "focus" | "scroll_to" (default: "click")
//   typeText     — (for "type" action) the text to type
//   matchIndex   — (optional) if multiple matches, pick this index (default: 0 = first)
//   parentSelector — (optional) scope the search within this parent element
//
// Returns a status string on success.

const selector = params?.selector;
if (!selector) {
  return 'Error: "selector" param is required';
}

// Scope the search
const parent = params.parentSelector
  ? document.querySelector(params.parentSelector)
  : document;

if (!parent) {
  return `Error: parent element not found for selector "${params.parentSelector}"`;
}

let candidates = [...parent.querySelectorAll(selector)];

// Filter by text content if provided
if (params.text !== undefined && params.text !== null) {
  const needle = String(params.text);
  candidates = candidates.filter(el => {
    const content = el.textContent?.trim() || '';
    if (params.textExact) {
      return content === needle;
    }
    return content.toLowerCase().includes(needle.toLowerCase());
  });
}

if (candidates.length === 0) {
  const textClause = params.text ? ` with text "${params.text}"` : '';
  return `Error: No elements found matching "${selector}"${textClause}`;
}

const matchIndex = params.matchIndex || 0;
if (matchIndex < 0 || matchIndex >= candidates.length) {
  return `Error: matchIndex ${matchIndex} out of range (0-${candidates.length - 1}), found ${candidates.length} match(es)`;
}

const el = candidates[matchIndex];
const action = (params.action || 'click').toLowerCase();
const desc = el.getAttribute('aria-label') || el.textContent?.trim().substring(0, 50) || el.tagName.toLowerCase();

switch (action) {
  case 'click': {
    if (typeof click === 'function') {
      await click(el);
    } else {
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
      await wait(randomDelay(100, 200));
      el.click();
    }
    return `Clicked "${selector}" match #${matchIndex} (${desc})`;
  }

  case 'type': {
    const text = params.typeText;
    if (text === undefined || text === null) {
      return 'Error: "typeText" param is required for type action';
    }
    if (typeof type === 'function') {
      await type(el, text, { clear: true });
    } else {
      el.focus();
      el.value = '';
      el.value = text;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return `Typed "${text}" into "${selector}" match #${matchIndex} (${desc})`;
  }

  case 'hover': {
    if (typeof hover === 'function') {
      await hover(el);
    } else {
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
      await wait(randomDelay(200, 500));
    }
    return `Hovered over "${selector}" match #${matchIndex} (${desc})`;
  }

  case 'focus': {
    el.focus();
    el.dispatchEvent(new Event('focus', { bubbles: true }));
    return `Focused "${selector}" match #${matchIndex} (${desc})`;
  }

  case 'scroll_to': {
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    await wait(500);
    return `Scrolled to "${selector}" match #${matchIndex} (${desc})`;
  }

  default:
    return `Error: Unknown action "${action}". Use: click, type, hover, focus, scroll_to`;
}
