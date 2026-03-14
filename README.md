# browser-mcp-extension

Browser automation via MCP (Model Context Protocol) and HTTP API, powered by a Chrome extension + Go binary.

The system has three components:
- **Chrome Extension (MV3):** Injects scripts into web pages and communicates with the Go binary via WebSocket.
- **Go Binary:** Exposes an MCP server (stdio) for AI agents and an HTTP API for curl/scripting.
- **JS Scripts:** Page-level automation scripts (click, type, navigate, observe, etc.).

## Quick Start

```bash
# Build
make build

# Wrap JS scripts into extension (required before first run)
make sync-scripts

# Start HTTP API + WebSocket server
make serve

# Or start MCP server on stdio (for AI agents)
make mcp
```

### Load the Chrome Extension

1. Open `chrome://extensions/` in Chrome
2. Enable "Developer mode"
3. Click "Load unpacked" and select the `extension/` directory
4. The extension connects to the Go binary via WebSocket on the configured port

## Configuration

All configuration is via environment variables. See `env.example` for defaults.

| Variable | Default | Description |
|---|---|---|
| `WS_PORT` | `9001` | WebSocket port the Chrome extension connects to |
| `WS_TOKEN` | _(empty)_ | Optional bearer token for WebSocket authentication |
| `HTTP_PORT` | `9082` | HTTP API server port |
| `JS_SCRIPTS_PATH` | `./resources/js_scripts` | Path to JS scripts directory |
| `OBSERVATIONS_DIR` | _(empty)_ | Directory for page snapshots. Empty = in-memory only |

## MCP Tools

Available tools when running in MCP mode (`./browser-cmd --mcp`):

| Tool | Description |
|---|---|
| `browser_navigate` | Navigate to a URL |
| `browser_navigate_back` | Go back in browser history |
| `browser_snapshot` | Capture page state (interactive elements, visible text) |
| `browser_get_state` | Get current page URL and title |
| `browser_click` | Click an element by index |
| `browser_type` | Type text into an element |
| `browser_hover` | Hover over an element |
| `browser_press_key` | Press a keyboard key |
| `browser_select_option` | Select an option in a dropdown |
| `browser_scroll` | Scroll the page |
| `browser_drag` | Drag and drop between elements |
| `browser_fill_form` | Fill multiple form fields at once |
| `browser_wait_for` | Wait for text to appear on the page |
| `browser_evaluate` | Execute arbitrary JavaScript |
| `browser_take_screenshot` | Capture a screenshot |
| `browser_tabs` | List, create, select, or close tabs |
| `browser_execute_script` | Execute a named JS script |
| `browser_list_scripts` | List available JS scripts |

## HTTP API

The HTTP API binds to `127.0.0.1` only. Endpoints:

| Method | Path | Description |
|---|---|---|
| GET | `/api/state` | Current page state |
| POST | `/api/observe` | Run observe script and return snapshot |
| POST | `/api/execute` | Execute a named script |
| POST | `/api/execute-raw` | Execute raw JavaScript |
| GET | `/api/scripts` | List available scripts |
| GET | `/api/screenshot` | Take a screenshot |
| GET | `/api/tabs` | List browser tabs |

## E2E Tests

E2E tests require Chromium (installed automatically by Playwright):

```bash
make e2e
```

## Development

```bash
make test    # Unit tests
make vet     # Go vet
make clean   # Remove binary, observations, wrapped scripts
```
