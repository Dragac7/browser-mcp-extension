// Background Service Worker for WebSocket Browser Controller
// Connects to WebSocket server and executes commands

// Default connection settings — overridden by config.json or chrome.storage.local
let wsPort = 9001;
let wsToken = '';
let ws = null;
let reconnectAttempts = 0;
const RECONNECT_DELAY = 3000; // 3 seconds - fixed interval

console.log('[BACKGROUND] Service worker initialized');

// ── Load connection config ──────────────────────────────────────────────
// Priority: chrome.storage.local (popup override) > config.json > defaults
async function loadConfig() {
  // 1. Try chrome.storage.local (set by popup UI)
  try {
    const stored = await chrome.storage.local.get(['wsPort', 'wsToken']);
    if (stored.wsPort) {
      wsPort = Number(stored.wsPort);
      wsToken = stored.wsToken || '';
      console.log('[BACKGROUND] Config from chrome.storage.local — port:', wsPort);
      return;
    }
  } catch (e) {
    console.warn('[BACKGROUND] chrome.storage.local read failed:', e.message);
  }

  // 2. Try bundled config.json (written by Runner or manually)
  try {
    const resp = await fetch(chrome.runtime.getURL('config.json'));
    if (resp.ok) {
      const cfg = await resp.json();
      if (cfg.port) wsPort = cfg.port;
      if (cfg.token) wsToken = cfg.token;
      console.log('[BACKGROUND] Config from config.json — port:', wsPort);
      return;
    }
  } catch (e) {
    // config.json may not exist — that's fine, use defaults
    console.log('[BACKGROUND] No config.json found, using defaults');
  }

  console.log('[BACKGROUND] Using default config — port:', wsPort);
}

// Listen for config changes from the popup
chrome.storage.onChanged.addListener((changes, area) => {
  if (area !== 'local') return;
  let changed = false;
  if (changes.wsPort?.newValue) {
    wsPort = Number(changes.wsPort.newValue);
    changed = true;
  }
  if (changes.wsToken) {
    wsToken = changes.wsToken.newValue || '';
    changed = true;
  }
  if (changed) {
    console.log('[BACKGROUND] Config updated from storage — port:', wsPort, '— reconnecting…');
    connectWebSocket();
  }
});

// ── MV3 keep-alive: use chrome.alarms so the service worker doesn't die ──
chrome.alarms.create('keepAlive', { periodInMinutes: 0.25 }); // every 15s
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'keepAlive') {
    console.log('[BACKGROUND] keepAlive tick — ws state:', ws?.readyState);
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.log('[BACKGROUND] keepAlive: reconnecting…');
      connectWebSocket();
    }
  }
});

// Initialize WebSocket connection on startup
chrome.runtime.onStartup.addListener(() => {
  console.log('[BACKGROUND] Chrome startup event detected');
  loadConfig().then(() => connectWebSocket());
});

// Also connect when extension is installed/updated
chrome.runtime.onInstalled.addListener(() => {
  console.log('[BACKGROUND] Extension installed/updated');
  loadConfig().then(() => connectWebSocket());
});

// Connect immediately when service worker starts
console.log('[BACKGROUND] Attempting initial connection after config load');
loadConfig().then(() => connectWebSocket());

function connectWebSocket() {
  // Close any existing connection before opening a new one —
  // prevents duplicate connections that race on the same server.
  if (ws) {
    console.log('[BACKGROUND] Closing previous WebSocket before reconnecting');
    const old = ws;
    ws = null;
    old.onclose = null; // prevent scheduleReconnect from firing
    old.close();
  }

  const wsUrl = wsToken
    ? `ws://localhost:${wsPort}?token=${encodeURIComponent(wsToken)}`
    : `ws://localhost:${wsPort}`;
  console.log(`[BACKGROUND] Connecting to WebSocket server at ws://localhost:${wsPort}`);

  try {
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      console.log('[BACKGROUND] ✓ WebSocket connection established');
      reconnectAttempts = 0;
      
      // Send initial connection message
      const welcomeMsg = {
        type: 'connected',
        timestamp: new Date().toISOString(),
        extensionId: chrome.runtime.id
      };
      console.log('[BACKGROUND] Sending welcome message:', welcomeMsg);
      ws.send(JSON.stringify(welcomeMsg));
    };
    
    ws.onmessage = (event) => {
      console.log('[BACKGROUND] ← Received message from server:', event.data);
      
      try {
        const message = JSON.parse(event.data);
        console.log('[BACKGROUND] Parsed message:', message);
        
        handleMessage(message);
      } catch (error) {
        console.error('[BACKGROUND] ✗ Error parsing message:', error);
        console.error('[BACKGROUND] Raw message data:', event.data);
      }
    };
    
    ws.onerror = (error) => {
      console.error('[BACKGROUND] ✗ WebSocket error:', error);
      console.error('[BACKGROUND] Error details:', {
        type: error.type,
        target: error.target?.readyState,
        url: `ws://localhost:${wsPort}`
      });
    };
    
    ws.onclose = (event) => {
      console.log('[BACKGROUND] WebSocket connection closed');
      console.log('[BACKGROUND] Close details:', {
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean
      });
      
      ws = null;
      scheduleReconnect();
    };
    
  } catch (error) {
    console.error('[BACKGROUND] ✗ Failed to create WebSocket connection:', error);
    scheduleReconnect();
  }
}

function scheduleReconnect() {
  reconnectAttempts++;
  
  console.log(`[BACKGROUND] Scheduling reconnection attempt ${reconnectAttempts} in ${RECONNECT_DELAY}ms`);
  
  setTimeout(() => {
    console.log(`[BACKGROUND] Reconnection attempt ${reconnectAttempts}`);
    connectWebSocket();
  }, RECONNECT_DELAY);
}

function waitForTabLoad(tabId, maxWait = 30000) {
  console.log('[BACKGROUND] waitForTabLoad: Waiting for tab', tabId, 'to finish loading');
  
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => {
      console.warn('[BACKGROUND] waitForTabLoad: Timeout waiting for tab to load');
      chrome.tabs.onUpdated.removeListener(listener);
      resolve(); // Resolve anyway, don't block
    }, maxWait);
    
    const listener = (updatedTabId, changeInfo, tab) => {
      if (updatedTabId === tabId) {
        console.log('[BACKGROUND] waitForTabLoad: Tab update:', {
          tabId: updatedTabId,
          status: changeInfo.status,
          url: tab.url
        });
        
        if (changeInfo.status === 'complete') {
          console.log('[BACKGROUND] waitForTabLoad: ✓ Tab loading complete');
          clearTimeout(timeout);
          chrome.tabs.onUpdated.removeListener(listener);
          resolve();
        }
      }
    };
    
    // Check if tab is already loaded
    chrome.tabs.get(tabId, (tab) => {
      if (chrome.runtime.lastError) {
        console.error('[BACKGROUND] waitForTabLoad: Error getting tab:', chrome.runtime.lastError);
        clearTimeout(timeout);
        reject(chrome.runtime.lastError);
        return;
      }
      
      console.log('[BACKGROUND] waitForTabLoad: Current tab status:', tab.status);
      
      if (tab.status === 'complete') {
        console.log('[BACKGROUND] waitForTabLoad: ✓ Tab already complete');
        clearTimeout(timeout);
        resolve();
      } else {
        console.log('[BACKGROUND] waitForTabLoad: Listening for tab updates...');
        chrome.tabs.onUpdated.addListener(listener);
      }
    });
  });
}

async function handleMessage(message) {
  console.log('[BACKGROUND] Handling message type:', message.type);

  if (message.type === 'screenshot') {
    try {
      const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
      if (!tab) {
        sendResponse({ type: 'result', success: false, error: 'No active tab for screenshot' });
        return;
      }
      const dataUrl = await chrome.tabs.captureVisibleTab(tab.windowId, { format: 'png' });
      sendResponse({ type: 'result', success: true, data: dataUrl, timestamp: new Date().toISOString() });
    } catch (err) {
      sendResponse({ type: 'result', success: false, error: `screenshot error: ${err.message}` });
    }
    return;
  }

  if (message.type === 'tabs') {
    try {
      switch (message.action) {
        case 'list': {
          const tabs = await chrome.tabs.query({});
          const list = tabs.map((t, i) => ({
            index: i,
            id: t.id,
            url: t.url,
            title: t.title,
            active: t.active,
            windowId: t.windowId,
          }));
          sendResponse({ type: 'result', success: true, data: list, timestamp: new Date().toISOString() });
          break;
        }
        case 'create': {
          const newTab = await chrome.tabs.create({ url: message.url || 'about:blank', active: true });
          await waitForTabLoad(newTab.id);
          const allTabs = await chrome.tabs.query({});
          const newIndex = allTabs.findIndex(t => t.id === newTab.id);
          sendResponse({ type: 'result', success: true, data: { id: newTab.id, index: newIndex }, timestamp: new Date().toISOString() });
          break;
        }
        case 'close': {
          const tabs = await chrome.tabs.query({});
          const idx = message.index !== undefined ? message.index : -1;
          if (idx < 0 || idx >= tabs.length) {
            sendResponse({ type: 'result', success: false, error: `Tab index ${idx} out of range (0-${tabs.length - 1})` });
            break;
          }
          await chrome.tabs.remove(tabs[idx].id);
          sendResponse({ type: 'result', success: true, data: `Tab ${idx} closed`, timestamp: new Date().toISOString() });
          break;
        }
        case 'select': {
          const tabs = await chrome.tabs.query({});
          const idx = message.index !== undefined ? message.index : -1;
          if (idx < 0 || idx >= tabs.length) {
            sendResponse({ type: 'result', success: false, error: `Tab index ${idx} out of range (0-${tabs.length - 1})` });
            break;
          }
          await chrome.tabs.update(tabs[idx].id, { active: true });
          sendResponse({ type: 'result', success: true, data: `Tab ${idx} selected`, timestamp: new Date().toISOString() });
          break;
        }
        default:
          sendResponse({ type: 'result', success: false, error: `Unknown tabs action: ${message.action}` });
      }
    } catch (err) {
      sendResponse({ type: 'result', success: false, error: `tabs error: ${err.message}` });
    }
    return;
  }

  if (message.type === 'execute') {
    console.log('[BACKGROUND] Execute command received');
    console.log('[BACKGROUND] Code to execute:', message.code);
    console.log('[BACKGROUND] Parameters:', message.params || '(none)');
    
    try {
      // Check if this is a navigation command (via params or code pattern)
      const navUrl = message.params?.url;
      const isNavigationCommand = navUrl || 
                                   message.code.includes('window.location.href =') || 
                                   message.code.includes('window.location =');
      
      if (isNavigationCommand && navUrl) {
        console.log('[BACKGROUND] Navigation command detected via params');
        console.log('[BACKGROUND] Target URL:', navUrl);
        
        // Get or create a suitable tab
        let [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
        
        if (!tab) {
          console.log('[BACKGROUND] No active tab, creating new one');
          tab = await chrome.tabs.create({ url: navUrl, active: true });
        } else {
          console.log('[BACKGROUND] Navigating tab', tab.id, 'to', navUrl);
          tab = await chrome.tabs.update(tab.id, { url: navUrl });
        }
        
        console.log('[BACKGROUND] Waiting for navigation to complete...');
        await waitForTabLoad(tab.id);
        
        console.log('[BACKGROUND] ✓ Navigation completed successfully');
        sendResponse({
          type: 'result',
          success: true,
          data: 'Navigating to ' + navUrl,
          url: navUrl,
          timestamp: new Date().toISOString()
        });
        return;
      }
      
      // Get the active tab
      console.log('[BACKGROUND] Querying for active tab');
      let [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
      
      if (!tab) {
        console.error('[BACKGROUND] ✗ No active tab found, creating new tab');
        tab = await chrome.tabs.create({ url: 'https://www.google.com', active: true });
        console.log('[BACKGROUND] ✓ Created new tab:', tab.id);
        await waitForTabLoad(tab.id);
      }
      
      console.log('[BACKGROUND] Active tab:', { id: tab.id, url: tab.url, status: tab.status });
      
      // Check if the tab is a restricted page
      const isRestrictedPage = tab.url.startsWith('chrome://') || 
                               tab.url.startsWith('chrome-extension://') ||
                               tab.url.startsWith('edge://') ||
                               tab.url.startsWith('about:');
      
      if (isRestrictedPage) {
        console.warn('[BACKGROUND] ⚠ Active tab is a restricted page:', tab.url);
        tab = await chrome.tabs.update(tab.id, { url: 'https://www.google.com' });
        await waitForTabLoad(tab.id);
      }
      
      // Wait for page to settle
      await new Promise(resolve => setTimeout(resolve, 500));

      // ── Strategy 0: File-based execution (CSP-immune) ──────────────────
      // Extension-bundled files bypass page CSP when loaded via
      // chrome.scripting.executeScript({ files }). The wrapped script stores
      // its result in window.__scriptResult.
      // Defence in depth: only accept scriptFile paths under "scripts/".
      if (message.scriptFile && message.scriptFile.startsWith('scripts/')) {
        let strategy0Success = false;
        try {
          console.log('[BACKGROUND] Trying Strategy 0: file-based execution for', message.scriptFile);

          // Step 1: Inject params + clear stale result in one call
          await chrome.scripting.executeScript({
            target: { tabId: tab.id },
            func: (p) => {
              delete window.__scriptResult;
              if (p && Object.keys(p).length > 0) window.__scriptParams = p;
            },
            args: [message.params || null],
            world: 'MAIN'
          });

          // Step 2: Load and execute the bundled script file
          await chrome.scripting.executeScript({
            target: { tabId: tab.id },
            files: [message.scriptFile],
            world: 'MAIN'
          });

          // Step 3: Read result — poll window.__scriptResult via setInterval.
          // For sync scripts (observe.js), result is set immediately.
          // For async scripts, poll until the async IIFE completes.
          const readResults = await chrome.scripting.executeScript({
            target: { tabId: tab.id },
            func: () => {
              return new Promise((resolve) => {
                let resolved = false;
                const interval = setInterval(() => {
                  if (window.__scriptResult !== undefined) {
                    clearInterval(interval);
                    if (!resolved) {
                      resolved = true;
                      const r = window.__scriptResult;
                      delete window.__scriptResult;
                      resolve(r);
                    }
                  }
                }, 50);
                setTimeout(() => {
                  clearInterval(interval);
                  if (!resolved) {
                    resolved = true;
                    // One final check before reporting timeout
                    if (window.__scriptResult !== undefined) {
                      const r = window.__scriptResult;
                      delete window.__scriptResult;
                      resolve(r);
                    } else {
                      resolve({ success: false, error: 'Script result timeout (55s)' });
                    }
                  }
                }, 55000);
              });
            },
            world: 'MAIN'
          });

          const result = readResults?.[0]?.result;
          if (result && typeof result === 'object' && 'success' in result) {
            console.log('[BACKGROUND] \u2713 Strategy 0 succeeded');
            strategy0Success = true;
            sendResponse({
              type: 'result',
              success: result.success,
              data: result.data,
              error: result.error,
              url: tab.url,
              timestamp: new Date().toISOString()
            });
          }
        } catch (e) {
          console.warn('[BACKGROUND] Strategy 0 failed:', e.message);
        }

        if (strategy0Success) return;
        console.log('[BACKGROUND] Strategy 0 failed, falling back to code-based execution');
      }
      // ── End Strategy 0 ─────────────────────────────────────────────────

      // Execute code using chrome.scripting in MAIN world
      // Uses nonce-stealing + blob URL + eval fallback chain to bypass CSP
      //
      // Fallback mechanism: some pages (e.g. Instagram) trigger history.pushState
      // during async script execution. This causes chrome.scripting.executeScript
      // to resolve with {result: null}. The injected <script> tag keeps running
      // but the return channel is broken.
      //
      // To handle this, the wrapper ALSO posts the result via window.postMessage().
      // A content script (ISOLATED world) relays the message to the background
      // script via chrome.runtime.sendMessage(). This channel is unaffected by
      // pushState because content scripts live in the ISOLATED world.
      console.log('[BACKGROUND] Executing code in tab', tab.id);
      
      const cbId = '__r' + Math.random().toString(36).slice(2);
      const execId = '__exec_' + Math.random().toString(36).slice(2);
      
      try {
        // Set up background message listener for the relay fallback.
        // If the primary executeScript return is null, we await this promise.
        let relayResolve;
        const relayPromise = new Promise((resolve) => {
          relayResolve = resolve;
          setTimeout(() => resolve(null), 65000);
        });
        
        const messageListener = (msg) => {
          if (msg && msg.type === '__ext_result_relay' && msg.execId === execId) {
            console.log('[BACKGROUND] ✓ Received relay result for', execId);
            chrome.runtime.onMessage.removeListener(messageListener);
            relayResolve(msg.result);
          }
        };
        chrome.runtime.onMessage.addListener(messageListener);
        
        // Inject relay content script (ISOLATED world).
        // It listens for window.postMessage from the MAIN world and forwards
        // the result to the background via chrome.runtime.sendMessage.
        await chrome.scripting.executeScript({
          target: { tabId: tab.id },
          func: (execId) => {
            const handler = (event) => {
              if (event.data && event.data.__ext_relay && event.data.execId === execId) {
                window.removeEventListener('message', handler);
                chrome.runtime.sendMessage({
                  type: '__ext_result_relay',
                  execId: execId,
                  result: event.data.result
                });
              }
            };
            window.addEventListener('message', handler);
          },
          args: [execId],
          world: 'ISOLATED'
        });
        
        // Execute the actual user script in MAIN world
        const results = await chrome.scripting.executeScript({
          target: { tabId: tab.id },
          func: (code, cbId, execId) => {
            return new Promise((resolve) => {
              let settled = false;
              
              // Callback for async result from injected script
              window[cbId] = (result) => {
                if (settled) return;
                settled = true;
                delete window[cbId];
                resolve(result);
              };
              
              // Wrap code: suppresses history.pushState/replaceState/back/forward/go
              // during execution so Chrome does not see a URL change and does not
              // return null from chrome.scripting.executeScript.
              // The relay via postMessage is kept as a safety net.
              const execCode = `(async()=>{` +
                `const __hps=history.pushState.bind(history);` +
                `const __hrs=history.replaceState.bind(history);` +
                `const __hb=history.back.bind(history);` +
                `const __hf=history.forward.bind(history);` +
                `const __hg=history.go.bind(history);` +
                `history.pushState=function(){};` +
                `history.replaceState=function(){};` +
                `history.back=function(){};` +
                `history.forward=function(){};` +
                `history.go=function(){};` +
                `try{const __r=await ${code};` +
                `const __res={success:true,data:__r};` +
                `window.postMessage({__ext_relay:true,execId:'${execId}',result:__res},'*');` +
                `window['${cbId}'](__res)` +
                `}catch(e){` +
                `const __res={success:false,error:e.message};` +
                `window.postMessage({__ext_relay:true,execId:'${execId}',result:__res},'*');` +
                `window['${cbId}'](__res)` +
                `}finally{` +
                `history.pushState=__hps;` +
                `history.replaceState=__hrs;` +
                `history.back=__hb;` +
                `history.forward=__hf;` +
                `history.go=__hg;` +
                `}})();`;
              
              // Strategy 1: Inline script with stolen nonce (works on nonce-based CSP sites like Instagram)
              let executed = false;
              try {
                const nonceEl = document.querySelector('script[nonce]');
                const nonce = nonceEl?.nonce;
                if (nonce) {
                  console.log('[PAGE-EXEC] Using nonce strategy');
                  const el = document.createElement('script');
                  el.nonce = nonce;
                  el.textContent = execCode;
                  document.documentElement.appendChild(el);
                  el.remove();
                  executed = true;
                }
              } catch (e) {
                console.warn('[PAGE-EXEC] Nonce strategy failed:', e.message);
              }
              
              // Strategy 2: Blob URL (works on sites with blob: in CSP)
              if (!executed) {
                try {
                  console.log('[PAGE-EXEC] Using blob URL strategy');
                  const blob = new Blob([execCode], { type: 'text/javascript' });
                  const url = URL.createObjectURL(blob);
                  const el = document.createElement('script');
                  el.src = url;
                  el.onload = () => { URL.revokeObjectURL(url); el.remove(); };
                  el.onerror = () => {
                    URL.revokeObjectURL(url);
                    el.remove();
                    // Strategy 3: Direct eval fallback
                    console.log('[PAGE-EXEC] Blob failed, trying eval');
                    try {
                      const result = eval(code);
                      Promise.resolve(result).then(
                        (data) => { if (!settled) { settled = true; delete window[cbId]; resolve({ success: true, data }); } },
                        (err) => { if (!settled) { settled = true; delete window[cbId]; resolve({ success: false, error: err.message }); } }
                      );
                    } catch (e) {
                      if (!settled) { settled = true; delete window[cbId]; resolve({ success: false, error: e.message }); }
                    }
                  };
                  document.documentElement.appendChild(el);
                  executed = true;
                } catch (e) {
                  console.warn('[PAGE-EXEC] Blob strategy failed:', e.message);
                }
              }
              
              // Strategy 3: Direct eval (works on permissive CSP sites)
              if (!executed) {
                console.log('[PAGE-EXEC] Using eval strategy');
                try {
                  const result = eval(code);
                  Promise.resolve(result).then(
                    (data) => { if (!settled) { settled = true; delete window[cbId]; resolve({ success: true, data }); } },
                    (err) => { if (!settled) { settled = true; delete window[cbId]; resolve({ success: false, error: err.message }); } }
                  );
                } catch (e) {
                  if (!settled) { settled = true; delete window[cbId]; resolve({ success: false, error: e.message }); }
                }
              }
              
              // Timeout
              setTimeout(() => {
                if (!settled) {
                  settled = true;
                  delete window[cbId];
                  resolve({ success: false, error: 'Execution timeout (60s)' });
                }
              }, 60000);
            });
          },
          args: [message.code, cbId, execId],
          world: 'MAIN'
        });
        
        console.log('[BACKGROUND] ✓ Script execution completed');
        
        let result = results?.[0]?.result ?? null;
        
        // If the primary return is null (likely due to pushState navigation),
        // wait for the relay from the content script via chrome.runtime.onMessage.
        if (result == null) {
          console.log('[BACKGROUND] Primary result null — waiting for relay fallback…');
          result = await relayPromise;
        }
        
        // Clean up the listener in case it wasn't triggered
        chrome.runtime.onMessage.removeListener(messageListener);
        
        if (result != null) {
          console.log('[BACKGROUND] Result:', result);
          sendResponse({
            type: 'result',
            success: result.success,
            data: result.data,
            error: result.error,
            url: tab.url,
            timestamp: new Date().toISOString()
          });
        } else {
          sendResponse({ type: 'result', success: false, error: 'Script execution context lost — no result received' });
        }
        
      } catch (error) {
        console.error('[BACKGROUND] ✗ Failed to execute script:', error.message);
        sendResponse({ type: 'result', success: false, error: error.message });
      }
      
    } catch (error) {
      console.error('[BACKGROUND] ✗ Error:', error.message);
      sendResponse({ type: 'result', success: false, error: error.message });
    }
  } else {
    console.warn('[BACKGROUND] ⚠ Unknown message type:', message.type);
  }
}

function sendResponse(response) {
  if (ws && ws.readyState === WebSocket.OPEN) {
    console.log('[BACKGROUND] → Sending response to server:', response);
    ws.send(JSON.stringify(response));
    console.log('[BACKGROUND] ✓ Response sent successfully');
  } else {
    console.error('[BACKGROUND] ✗ Cannot send response - WebSocket not connected');
    console.error('[BACKGROUND] WebSocket state:', ws?.readyState);
  }
}

// Log when service worker becomes inactive (for debugging)
chrome.runtime.onSuspend.addListener(() => {
  console.log('[BACKGROUND] Service worker suspending - closing WebSocket');
  if (ws) {
    ws.close();
  }
});

console.log('[BACKGROUND] Background script loaded and ready');

