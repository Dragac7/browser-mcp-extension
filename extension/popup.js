// Popup UI for configuring WebSocket port and token.
// Values saved here override config.json in background.js.

const portInput = document.getElementById('port');
const tokenInput = document.getElementById('token');
const saveBtn = document.getElementById('save');
const clearBtn = document.getElementById('clear');
const statusEl = document.getElementById('status');

// Load current values on open
chrome.storage.local.get(['wsPort', 'wsToken'], (result) => {
  if (result.wsPort) portInput.value = result.wsPort;
  if (result.wsToken) tokenInput.value = result.wsToken;
});

function showStatus(message, className) {
  statusEl.textContent = message;
  statusEl.className = 'status ' + className;
  setTimeout(() => { statusEl.className = 'status'; }, 2000);
}

saveBtn.addEventListener('click', () => {
  const port = parseInt(portInput.value, 10);
  if (!port || port < 1024 || port > 65535) {
    showStatus('Port must be 1024-65535', 'cleared');
    return;
  }
  chrome.storage.local.set({
    wsPort: port,
    wsToken: tokenInput.value.trim()
  }, () => {
    showStatus('Saved — reconnecting…', 'ok');
  });
});

clearBtn.addEventListener('click', () => {
  chrome.storage.local.remove(['wsPort', 'wsToken'], () => {
    portInput.value = '';
    tokenInput.value = '';
    showStatus('Cleared — using config.json or defaults', 'cleared');
  });
});
