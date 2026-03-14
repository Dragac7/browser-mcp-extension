// scroll.js — Scroll the page or a specific element.
//
// Params:
//   elementIndex — (optional) index from last observe.js snapshot; if omitted, scrolls window
//   deltaX       — (optional) horizontal scroll delta in pixels (default: 0)
//   deltaY       — (optional) vertical scroll delta in pixels (default: 300)

const deltaX = params?.deltaX ?? 0;
const deltaY = params?.deltaY ?? 300;
const idx = params?.elementIndex;

if (idx !== undefined && idx !== null) {
  const elements = window.__observedElements;
  if (!elements || !Array.isArray(elements)) {
    return 'Error: No observed elements found. Run observe (browser_snapshot) first.';
  }
  if (idx < 0 || idx >= elements.length) {
    return `Error: elementIndex ${idx} out of range (0-${elements.length - 1})`;
  }
  const el = elements[idx];
  if (!el) {
    return `Error: Element at index ${idx} is no longer in the DOM`;
  }
  el.scrollIntoView({ behavior: 'smooth', block: 'center' });
  await wait(500);
  return `Scrolled element #${idx} into view`;
}

window.scrollBy({ left: deltaX, top: deltaY, behavior: 'smooth' });
await wait(500);
return `Scrolled window by (${deltaX}, ${deltaY})`;
