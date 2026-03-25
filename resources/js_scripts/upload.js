// upload.js — Set files on <input type="file"> via DataTransfer API.
//
// Params:
//   elementIndex — numeric index from the last observe.js snapshot
//   files        — array of {name: string, content: string (base64), mimeType?: string}
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
if (!document.contains(el)) {
  return `Error: Element at index ${idx} has been removed from the DOM`;
}
if (el.tagName.toLowerCase() !== 'input' || el.type !== 'file') {
  return `Error: Element at index ${idx} is not a file input (found <${el.tagName.toLowerCase()} type="${el.type || ''}">)`;
}

const filesParam = params.files;
if (!Array.isArray(filesParam) || filesParam.length === 0) {
  return 'Error: "files" must be a non-empty array of {name, content, mimeType?}';
}

// Check multiple attribute when more than one file
if (filesParam.length > 1 && !el.multiple) {
  return `Error: Element at index ${idx} does not have the "multiple" attribute but ${filesParam.length} files were provided`;
}

const MAX_BASE64_LENGTH = 14 * 1024 * 1024; // ~10 MB decoded
const dt = new DataTransfer();
const fileNames = [];

for (let i = 0; i < filesParam.length; i++) {
  const f = filesParam[i];
  if (!f.name || typeof f.name !== 'string') {
    return `Error: files[${i}].name is required and must be a string`;
  }
  if (!f.content || typeof f.content !== 'string') {
    return `Error: files[${i}].content is required and must be a base64 string`;
  }

  if (f.content.length > MAX_BASE64_LENGTH) {
    return `Error: files[${i}].content exceeds 10 MB limit (${Math.round(f.content.length / 1024 / 1024)} MB base64)`;
  }

  let bytes;
  try {
    const binary = atob(f.content);
    bytes = Uint8Array.from(binary, c => c.charCodeAt(0));
  } catch (e) {
    return `Error: files[${i}].content is not valid base64: ${e.message}`;
  }

  const mimeType = f.mimeType || 'application/octet-stream';
  const file = new File([bytes], f.name, { type: mimeType });
  dt.items.add(file);
  fileNames.push(f.name);
}

el.files = dt.files;

// Dispatch events so frameworks (React, Vue, etc.) detect the change
el.dispatchEvent(new Event('input', { bubbles: true }));
el.dispatchEvent(new Event('change', { bubbles: true }));

const desc = el.getAttribute('aria-label') || el.id || 'file input';
return `Uploaded ${fileNames.length} file(s) [${fileNames.join(', ')}] to element #${idx} (${desc})`;
