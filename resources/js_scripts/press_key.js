// press_key.js — Dispatch a keyboard event on the focused element.
//
// Params:
//   key — key name per KeyboardEvent spec (e.g. "Enter", "Escape", "ArrowDown", "a")

const key = params?.key;
if (!key) {
  return 'Error: "key" param is required';
}

const target = document.activeElement || document.body;
// Include both key and code; for single characters use the character as code,
// for named keys (Enter, Escape, etc.) use the key name as code.
const code = key.length === 1 ? `Key${key.toUpperCase()}` : key;
const init = { key, code, bubbles: true, cancelable: true };

target.dispatchEvent(new KeyboardEvent('keydown', init));
await wait(30);
target.dispatchEvent(new KeyboardEvent('keyup', init));

return `Key "${key}" pressed on ${target.tagName.toLowerCase()}`;
