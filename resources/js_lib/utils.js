// Shared utility functions for automation snippets
// These are prepended to every snippet by the Go command at runtime.
// They live as local variables inside the IIFE wrapper -- zero global footprint.

const wait = (ms) => new Promise(resolve => setTimeout(resolve, ms));
const randomDelay = (min, max) => Math.floor(Math.random() * (max - min + 1)) + min;

/**
 * Type text into an input element with human-like keyboard simulation.
 * Uses native value setter + full keyboard event sequence so React / Instagram
 * style frameworks pick up the change through their synthetic event system.
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
 * Dispatches the full pointer → mouse → click chain that React / Instagram
 * style frameworks rely on through their synthetic event system.
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

/**
 * Given an SVG element, find the closest interactive button ancestor.
 * Instagram typically wraps action SVGs inside <button> or [role="button"].
 * @param {SVGElement} svg - The SVG element
 * @returns {HTMLElement} The button element (or the SVG's parent as fallback)
 */
function getButtonFromSvg(svg) {
  return svg.closest('button')
    || svg.closest('[role="button"]')
    || svg.parentElement;
}

/**
 * Close a modal by clicking the Close (X) button or pressing Escape.
 * Works for Instagram post modals, story viewers, etc.
 */
async function closeModal() {
  const closeSvg = document.querySelector('svg[aria-label="Close"]');
  if (closeSvg) {
    const closeBtn = getButtonFromSvg(closeSvg);
    console.log('[SNIPPET] Closing modal');
    await click(closeBtn);
  } else {
    console.log('[SNIPPET] Close button not found, pressing Escape');
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
  }
}

/**
 * Scroll the Instagram comment list to reveal more comments.
 * Scrolls in 4 steps of 600px with random delays between each.
 */
async function scrollComments() {
  const commentList = document.querySelector('ul._a9z6');
  if (!commentList) {
    console.log('[SNIPPET] No comment list found to scroll');
    return;
  }

  const scrollable = commentList.closest('[style*="overflow"]')
    || commentList.parentElement;

  if (!scrollable) {
    console.log('[SNIPPET] No scrollable container found');
    return;
  }

  const scrollSteps = 4;
  for (let i = 0; i < scrollSteps; i++) {
    scrollable.scrollTop += 600;
    console.log(`[SNIPPET] Scrolled comments (step ${i + 1}/${scrollSteps})`);
    await wait(randomDelay(1000, 2000));
  }
}

/**
 * Find unique profile links in the Instagram comment section.
 * Retries up to 10 times, deduplicates by href.
 * @returns {HTMLAnchorElement[]} Array of unique profile link elements
 */
async function findPeopleLinks() {
  let people = [];
  for (let attempt = 0; attempt < 10; attempt++) {
    const commentSection = document.querySelector('ul._a9z6');
    if (commentSection) {
      const allLinks = [...commentSection.querySelectorAll('a[href]')];
      people = allLinks.filter(a => {
        const href = a.getAttribute('href');
        return href && /^\/[A-Za-z0-9_.]+\/$/.test(href)
          && !href.startsWith('/p/')
          && !href.startsWith('/explore/')
          && !href.startsWith('/reels/');
      });
      // Deduplicate by href
      const seen = new Set();
      people = people.filter(a => {
        const href = a.getAttribute('href');
        if (seen.has(href)) return false;
        seen.add(href);
        return true;
      });
    }
    if (people.length > 0) break;
    console.log('[SNIPPET] No people links yet, retrying…');
    await wait(800);
  }

  if (people.length === 0) {
    console.warn('[SNIPPET] No people links found');
    return [];
  }
  return people;
}
