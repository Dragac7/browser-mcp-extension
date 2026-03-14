// select_option.js — Select one or more options in a <select> element by index.
//
// Params:
//   elementIndex — index of the <select> from the last observe.js snapshot
//   values       — array of option values to select (string[])

const elements = window.__observedElements;
if (!elements || !Array.isArray(elements)) {
  return 'Error: No observed elements found. Run observe (browser_snapshot) first.';
}

const idx = params?.elementIndex;
if (idx === undefined || idx === null) {
  return 'Error: "elementIndex" param is required';
}
if (idx < 0 || idx >= elements.length) {
  return `Error: elementIndex ${idx} out of range (0-${elements.length - 1})`;
}

const el = elements[idx];
if (!el || el.tagName.toLowerCase() !== 'select') {
  return `Error: Element at index ${idx} is not a <select>`;
}
if (!document.contains(el)) {
  return `Error: Element at index ${idx} has been removed from the DOM`;
}

const values = params?.values;
if (!values || !Array.isArray(values) || values.length === 0) {
  return 'Error: "values" must be a non-empty array';
}

const valueSet = new Set(values.map(String));
let matched = 0;
if (!el.multiple && values.length > 1) {
  return `Error: <select> at index ${idx} is not a multi-select but ${values.length} values were provided`;
}

for (const opt of el.options) {
  opt.selected = valueSet.has(opt.value);
  if (opt.selected) matched++;
}

if (matched === 0) {
  return `Error: None of the provided values matched any option in <select> #${idx}`;
}

el.dispatchEvent(new Event('change', { bubbles: true }));
el.dispatchEvent(new Event('input', { bubbles: true }));

return `Selected ${matched} option(s) in <select> #${idx} (values: ${values.join(', ')})`;
