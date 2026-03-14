// drag.js — Simulate a drag-and-drop between two elements by snapshot index.
//
// Params:
//   startElementIndex — index of the source element
//   endElementIndex   — index of the target element

const elements = window.__observedElements;
if (!elements || !Array.isArray(elements)) {
  return 'Error: No observed elements found. Run observe (browser_snapshot) first.';
}

const startIdx = params?.startElementIndex;
const endIdx   = params?.endElementIndex;

if (startIdx === undefined || startIdx === null) return 'Error: "startElementIndex" is required';
if (endIdx   === undefined || endIdx   === null) return 'Error: "endElementIndex" is required';

if (startIdx < 0 || startIdx >= elements.length) return `Error: startElementIndex ${startIdx} out of range`;
if (endIdx   < 0 || endIdx   >= elements.length) return `Error: endElementIndex ${endIdx} out of range`;

const src = elements[startIdx];
const dst = elements[endIdx];

if (!src || !document.contains(src)) return `Error: Source element at index ${startIdx} not in DOM`;
if (!dst || !document.contains(dst)) return `Error: Target element at index ${endIdx} not in DOM`;

const srcRect = src.getBoundingClientRect();
const dstRect = dst.getBoundingClientRect();

const srcX = srcRect.left + srcRect.width  / 2;
const srcY = srcRect.top  + srcRect.height / 2;
const dstX = dstRect.left + dstRect.width  / 2;
const dstY = dstRect.top  + dstRect.height / 2;

function makeMouseEvent(type, x, y) {
  return new MouseEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y });
}
function makeDragEvent(type, x, y) {
  // Note: dataTransfer is read-only on constructed DragEvents; synthetic drag may not
  // work with libraries that call event.dataTransfer.setData() in their dragstart handler.
  return new DragEvent(type, { bubbles: true, cancelable: true, clientX: x, clientY: y });
}

src.dispatchEvent(makeMouseEvent('mousedown', srcX, srcY));
src.dispatchEvent(makeDragEvent('dragstart', srcX, srcY));
await wait(randomDelay(100, 200));
dst.dispatchEvent(makeDragEvent('dragover',  dstX, dstY));
dst.dispatchEvent(makeDragEvent('drop',      dstX, dstY));
src.dispatchEvent(makeDragEvent('dragend',   dstX, dstY));
dst.dispatchEvent(makeMouseEvent('mouseup',  dstX, dstY));
await wait(200);

return `Dragged element #${startIdx} → element #${endIdx}`;
