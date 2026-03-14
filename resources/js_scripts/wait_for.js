// wait_for.js — Wait for a fixed time, text to appear, or text to disappear.
//
// Params:
//   time     — (optional) seconds to wait unconditionally
//   text     — (optional) text to wait for (polls until visible or timeout)
//   textGone — (optional) text to wait for to disappear (polls until gone or timeout)
//   timeout  — (optional) max seconds to poll (default: 10)

// timeout=0 or absent defaults to 10s; minimum effective poll timeout is 1s.
const timeoutSec = (params?.timeout != null && params.timeout > 0) ? params.timeout : 10;
const pollInterval = 300; // ms

if (params?.time != null) {
  await wait(params.time * 1000);
  return `Waited ${params.time}s`;
}

if (params?.text != null) {
  const target = params.text;
  const deadline = Date.now() + timeoutSec * 1000;
  // Use textContent (no reflow) — includes hidden text, which is acceptable
  // for automation purposes. Visible-only check would require innerText (costly reflow).
  do {
    if (document.body.textContent.includes(target)) {
      return `Text "${target}" found`;
    }
    await wait(pollInterval);
  } while (Date.now() < deadline);
  return `Error: Timeout waiting for text "${target}" (${timeoutSec}s)`;
}

if (params?.textGone != null) {
  const target = params.textGone;
  const deadline = Date.now() + timeoutSec * 1000;
  do {
    if (!document.body.textContent.includes(target)) {
      return `Text "${target}" is gone`;
    }
    await wait(pollInterval);
  } while (Date.now() < deadline);
  return `Error: Timeout waiting for text "${target}" to disappear (${timeoutSec}s)`;
}

return 'Error: Provide at least one of: "time", "text", "textGone"';
