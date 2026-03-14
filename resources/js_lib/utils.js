// Shared utility functions for automation snippets
// These are prepended to every snippet by the Go command at runtime.
// They live as local variables inside the IIFE wrapper -- zero global footprint.

const wait = (ms) => new Promise(resolve => setTimeout(resolve, ms));
const randomDelay = (min, max) => Math.floor(Math.random() * (max - min + 1)) + min;

/**
 * Type text into an input element with human-like keyboard simulation.
 * Uses native value setter + full keyboard event sequence so React-style
 * frameworks pick up the change through their synthetic event system.
 *
 * @param {HTMLElement} element - The input/textarea element
 * @param {string} text - The text to type
 * @param {Object} [opts] - Options
 * @param {number} [opts.minDelay=40] - Min delay between keystrokes (ms)
 * @param {number} [opts.maxDelay=130] - Max delay between keystrokes (ms)
 * @param {boolean} [opts.clear=true] - Whether to clear existing value first
 */
async function type(element, text, opts = {}) {
  const minDelay = opts.minDelay || 40;
  const maxDelay = opts.maxDelay || 130;
  const clear = opts.clear !== false;

  // Grab the *native* value setter – React overrides the element's own setter,
  // so we need the original from the prototype to actually update internal state.
  const nativeSetter =
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set ||
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set;

  element.focus();
  element.dispatchEvent(new Event('focus', { bubbles: true }));
  await wait(randomDelay(200, 400));

  if (clear && element.value) {
    // Select-all + delete to clear, just like a real user
    element.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', code: 'KeyA', ctrlKey: true, bubbles: true }));
    element.dispatchEvent(new KeyboardEvent('keyup',   { key: 'a', code: 'KeyA', ctrlKey: true, bubbles: true }));
    nativeSetter.call(element, '');
    element.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'deleteContentBackward' }));
    element.dispatchEvent(new Event('change', { bubbles: true }));
    await wait(randomDelay(50, 120));
  }

  for (const char of text) {
    const keyEventInit = { key: char, code: `Key${char.toUpperCase()}`, bubbles: true, cancelable: true };

    // keydown → keypress → (set value) → input → keyup
    element.dispatchEvent(new KeyboardEvent('keydown',  keyEventInit));
    element.dispatchEvent(new KeyboardEvent('keypress', keyEventInit));

    nativeSetter.call(element, element.value + char);

    element.dispatchEvent(new InputEvent('input', {
      bubbles: true,
      inputType: 'insertText',
      data: char,
    }));

    element.dispatchEvent(new KeyboardEvent('keyup', keyEventInit));

    await wait(randomDelay(minDelay, maxDelay));
  }

  // Final change event – fired when the field loses focus in a real browser,
  // but some frameworks listen for it immediately.
  element.dispatchEvent(new Event('change', { bubbles: true }));
}

/**
 * Click an element with a realistic human-like event sequence.
 * Dispatches the full pointer → mouse → click chain that React-style
 * frameworks rely on through their synthetic event system.
 *
 * @param {HTMLElement} element - The element to click
 * @param {Object} [opts] - Options
 * @param {number[]} [opts.hoverTime=[100,200]] - [min, max] hover delay before click (ms)
 */
async function click(element, opts = {}) {
  const hoverTime = opts.hoverTime || [100, 200];

  // Compute a plausible click position (centre of the element)
  const rect = element.getBoundingClientRect();
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;
  const shared = { bubbles: true, cancelable: true, view: window, clientX: x, clientY: y };

  // 1. Hover phase — pointer + mouse enter / over
  element.dispatchEvent(new PointerEvent('pointerover', { ...shared, pointerId: 1 }));
  element.dispatchEvent(new PointerEvent('pointerenter', { ...shared, pointerId: 1, bubbles: false }));
  element.dispatchEvent(new MouseEvent('mouseover', shared));
  element.dispatchEvent(new MouseEvent('mouseenter', { ...shared, bubbles: false }));
  await wait(randomDelay(hoverTime[0], hoverTime[1]));

  // 2. Press phase — pointerdown + mousedown
  element.dispatchEvent(new PointerEvent('pointerdown', { ...shared, pointerId: 1, button: 0, buttons: 1 }));
  element.dispatchEvent(new MouseEvent('mousedown', { ...shared, button: 0, buttons: 1 }));
  if (element.focus) element.focus();
  await wait(randomDelay(30, 80));

  // 3. Release phase — pointerup + mouseup + click
  element.dispatchEvent(new PointerEvent('pointerup', { ...shared, pointerId: 1, button: 0, buttons: 0 }));
  element.dispatchEvent(new MouseEvent('mouseup', { ...shared, button: 0, buttons: 0 }));
  element.dispatchEvent(new MouseEvent('click', { ...shared, button: 0, buttons: 0 }));
}

/**
 * Hover over an element for a random duration.
 * @param {HTMLElement} element - The element to hover
 * @param {number[]} [duration=[200,500]] - [min, max] hover duration (ms)
 */
async function hover(element, duration = [200, 500]) {
  element.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
  await wait(randomDelay(duration[0], duration[1]));
}
