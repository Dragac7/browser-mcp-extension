// fill_form.js — Fill multiple form fields sequentially.
//
// Params:
//   fields — array of { elementIndex: number, text: string, clear?: boolean }

const elements = window.__observedElements;
if (!elements || !Array.isArray(elements)) {
  return 'Error: No observed elements found. Run observe (browser_snapshot) first.';
}

const fields = params?.fields;
if (!fields || !Array.isArray(fields) || fields.length === 0) {
  return 'Error: "fields" must be a non-empty array of {elementIndex, text}';
}

const results = [];

for (const field of fields) {
  const idx = field.elementIndex;
  if (idx === undefined || idx === null) {
    results.push(`Error: missing elementIndex in field`);
    continue;
  }
  if (idx < 0 || idx >= elements.length) {
    results.push(`Error: elementIndex ${idx} out of range`);
    continue;
  }
  const el = elements[idx];
  if (!el || !document.contains(el)) {
    results.push(`Error: element #${idx} not in DOM`);
    continue;
  }
  const clearFirst = field.clear !== false;
  if (typeof type === 'function') {
    await type(el, field.text, { clear: clearFirst });
  } else {
    el.focus();
    if (clearFirst) el.value = '';
    el.value = field.text;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
  }
  await wait(randomDelay(80, 150));
  results.push(`Filled #${idx} with "${field.text}"`);
}

return results.join('; ');
