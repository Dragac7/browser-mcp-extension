// interact.js — Generic element interaction by snapshot index.
// Requires observe.js to have been run first (populates window.__observedElements).
//
// Params:
//   action       — "click" | "double_click" | "type" | "hover" | "focus" | "scroll_to"
//   elementIndex — numeric index from the last observe.js snapshot
//   text         — (for "type" action) the text to type
//   clear        — (for "type" action) whether to clear existing value first (default: true)
//
// Returns a status string on success.

const elements = window.__observedElements;
if (!elements || !Array.isArray(elements)) {
  return 'Error: No observed elements found. Run observe.js first.';
}

const idx = params.elementIndex;
if (idx === undefined || idx === null) {
  return 'Error: elementIndex is required';
}

if (idx < 0 || idx >= elements.length) {
  return `Error: elementIndex ${idx} out of range (0-${elements.length - 1})`;
}

const el = elements[idx];
if (!el) {
  return `Error: Element at index ${idx} is no longer in the DOM`;
}

// Verify element is still attached to the document
if (!document.contains(el)) {
  return `Error: Element at index ${idx} has been removed from the DOM`;
}

const action = (params.action || 'click').toLowerCase();
const tag = el.tagName.toLowerCase();
const desc = el.getAttribute('aria-label') || el.textContent?.trim().substring(0, 50) || tag;

switch (action) {
  case 'click': {
    // Use the shared click() utility if available, otherwise direct click
    if (typeof click === 'function') {
      await click(el);
    } else {
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
      await wait(randomDelay(100, 200));
      el.click();
    }
    return `Clicked element #${idx} (${desc})`;
  }

  case 'double_click': {
    // Two rapid clicks to simulate a double-click
    if (typeof click === 'function') {
      await click(el);
      await wait(randomDelay(60, 120));
      await click(el);
    } else {
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
      await wait(50);
      el.click();
      await wait(randomDelay(60, 120));
      el.click();
    }
    el.dispatchEvent(new MouseEvent('dblclick', { bubbles: true, cancelable: true }));
    return `Double-clicked element #${idx} (${desc})`;
  }

  case 'type': {
    const text = params.text;
    if (text === undefined || text === null) {
      return 'Error: "text" param is required for type action';
    }
    const clearFirst = params.clear !== false;
    // Use the shared type() utility if available
    if (typeof type === 'function') {
      await type(el, text, { clear: clearFirst });
    } else {
      el.focus();
      if (clearFirst) el.value = '';
      el.value = text;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
    }
    return `Typed "${text}" into element #${idx} (${desc})`;
  }

  case 'hover': {
    if (typeof hover === 'function') {
      await hover(el);
    } else {
      el.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
      await wait(randomDelay(200, 500));
    }
    return `Hovered over element #${idx} (${desc})`;
  }

  case 'focus': {
    el.focus();
    el.dispatchEvent(new Event('focus', { bubbles: true }));
    return `Focused element #${idx} (${desc})`;
  }

  case 'scroll_to': {
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    await wait(500);
    return `Scrolled to element #${idx} (${desc})`;
  }

  default:
    return `Error: Unknown action "${action}". Use: click, double_click, type, hover, focus, scroll_to`;
}
