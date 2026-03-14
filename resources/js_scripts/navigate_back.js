// navigate_back.js — Go back one step in browser history.
// Returns a status string, or an error if there is no history to go back to.

const before = window.location.href;
window.history.back();
await wait(1500);
const after = window.location.href;
if (after === before) {
  return `Error: Could not navigate back — already at the first history entry`;
}
return `Navigated back (was: ${before}, now: ${after})`;
